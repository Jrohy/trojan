package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"trojan/core"
	"trojan/util"
	"trojan/web/controller"

	"github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	identityKey       = "username"
	roleKey           = "role"
	userIDKey         = "user_id"
	sessionVersionKey = "session_version"
	roleAdmin         = "admin"
	roleUser          = "user"
)

var (
	authMiddleware *jwt.GinJWTMiddleware
	err            error
)

// Login contains the browser-provided username and SHA-224 password digest.
type Login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type authIdentity struct {
	UserID         uint
	Username       string
	Role           string
	SessionVersion uint64
}

type loginAttempt struct {
	Count        int
	WindowStart  time.Time
	BlockedUntil time.Time
}

var loginAttempts = struct {
	sync.Mutex
	items map[string]loginAttempt
}{items: make(map[string]loginAttempt)}

func getSecretKey() string {
	sk, _ := core.GetValue("secretKey")
	if sk == "" {
		sk = util.RandString(32, util.ALL)
		core.SetValue("secretKey", sk)
	}
	return sk
}

func claimString(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}

func identityFromClaims(claims jwt.MapClaims) *authIdentity {
	userID, _ := strconv.ParseUint(claimString(claims, userIDKey), 10, 64)
	sessionVersion, _ := strconv.ParseUint(claimString(claims, sessionVersionKey), 10, 64)
	return &authIdentity{
		UserID:         uint(userID),
		Username:       claimString(claims, identityKey),
		Role:           claimString(claims, roleKey),
		SessionVersion: sessionVersion,
	}
}

func loginAttemptKey(c *gin.Context, username string) string {
	return c.ClientIP() + "\x00" + strings.ToLower(strings.TrimSpace(username))
}

func loginAllowed(key string, now time.Time) bool {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	entry, exists := loginAttempts.items[key]
	if !exists {
		return true
	}
	if entry.BlockedUntil.After(now) {
		return false
	}
	if now.Sub(entry.WindowStart) > 10*time.Minute {
		delete(loginAttempts.items, key)
	}
	return true
}

func recordLoginFailure(key string, now time.Time) {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	entry := loginAttempts.items[key]
	if entry.WindowStart.IsZero() || now.Sub(entry.WindowStart) > 10*time.Minute {
		entry = loginAttempt{WindowStart: now}
	}
	entry.Count++
	if entry.Count >= 5 {
		entry.BlockedUntil = now.Add(10 * time.Minute)
	}
	loginAttempts.items[key] = entry
	if len(loginAttempts.items) > 2048 {
		for attemptKey, attempt := range loginAttempts.items {
			if now.Sub(attempt.WindowStart) > 20*time.Minute && !attempt.BlockedUntil.After(now) {
				delete(loginAttempts.items, attemptKey)
			}
		}
	}
}

func clearLoginFailures(key string) {
	loginAttempts.Lock()
	delete(loginAttempts.items, key)
	loginAttempts.Unlock()
}

func userSessionVersion(userID uint, passwordDigest string) uint64 {
	mac := hmac.New(sha256.New, []byte(getSecretKey()))
	_, _ = fmt.Fprintf(mac, "%d\x00%s", userID, passwordDigest)
	return binary.BigEndian.Uint64(mac.Sum(nil)[:8])
}

func passwordDigestMatches(expected, actual string) bool {
	return len(expected) == 56 && len(actual) == 56 &&
		subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func jwtInit(timeout int, secureCookie bool) {
	authMiddleware, err = jwt.New(&jwt.GinJWTMiddleware{
		Realm:          "trojan-manager",
		Key:            []byte(getSecretKey()),
		Timeout:        time.Minute * time.Duration(timeout),
		MaxRefresh:     time.Minute * time.Duration(timeout),
		IdentityKey:    identityKey,
		SendCookie:     true,
		CookieName:     "jwt",
		SecureCookie:   secureCookie,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			identity, ok := data.(*authIdentity)
			if !ok {
				return jwt.MapClaims{}
			}
			return jwt.MapClaims{
				identityKey:       identity.Username,
				roleKey:           identity.Role,
				userIDKey:         strconv.FormatUint(uint64(identity.UserID), 10),
				sessionVersionKey: strconv.FormatUint(identity.SessionVersion, 10),
			}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			return identityFromClaims(jwt.ExtractClaims(c))
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals Login
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}
			loginVals.Username = strings.TrimSpace(loginVals.Username)
			attemptKey := loginAttemptKey(c, loginVals.Username)
			now := time.Now()
			if !loginAllowed(attemptKey, now) {
				return nil, jwt.ErrFailedAuthentication
			}

			if loginVals.Username == roleAdmin {
				password, passwordErr := core.GetValue("admin_pass")
				if passwordErr == nil && password == loginVals.Password {
					clearLoginFailures(attemptKey)
					return &authIdentity{Username: roleAdmin, Role: roleAdmin}, nil
				}
				recordLoginFailure(attemptKey, now)
				return nil, jwt.ErrFailedAuthentication
			}

			user, userErr := core.GetMysql().PortalUserIdentityByUsername(loginVals.Username)
			if userErr == nil && passwordDigestMatches(user.PasswordDigest, loginVals.Password) {
				clearLoginFailures(attemptKey)
				return &authIdentity{
					UserID:         user.UserID,
					Username:       user.Username,
					Role:           roleUser,
					SessionVersion: userSessionVersion(user.UserID, user.PasswordDigest),
				}, nil
			}
			recordLoginFailure(attemptKey, now)
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			identity, ok := data.(*authIdentity)
			return ok && (identity.Role == roleAdmin || identity.Role == roleUser)
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": "认证失败或登录已过期",
			})
		},
		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})

	if err != nil {
		fmt.Println("JWT Error:" + err.Error())
	}
}

func currentIdentity(c *gin.Context) *authIdentity {
	if value, exists := c.Get(identityKey); exists {
		if identity, ok := value.(*authIdentity); ok {
			return identity
		}
	}
	return identityFromClaims(jwt.ExtractClaims(c))
}

func validatePortalIdentity(identity *authIdentity) bool {
	if identity == nil || identity.Role != roleUser || identity.UserID == 0 {
		return false
	}
	user, err := core.GetMysql().PortalUserIdentityByID(identity.UserID)
	return err == nil && user.Username == identity.Username &&
		userSessionVersion(user.UserID, user.PasswordDigest) == identity.SessionVersion
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if identity := currentIdentity(c); identity == nil || identity.Role != roleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "无权访问管理功能"})
			return
		}
		c.Next()
	}
}

func PortalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := currentIdentity(c)
		if identity == nil || identity.Role != roleUser {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "无权访问个人账户"})
			return
		}
		if !validatePortalIdentity(identity) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "账户登录已失效"})
			return
		}
		c.Set("portalUserID", identity.UserID)
		c.Next()
	}
}

func PortalUserID(c *gin.Context) uint {
	value, _ := c.Get("portalUserID")
	userID, _ := value.(uint)
	return userID
}

func updateUser(c *gin.Context) {
	responseBody := controller.ResponseBody{Msg: "success"}
	defer controller.TimeCost(time.Now(), &responseBody)
	pass := c.PostForm("password")
	if err := core.SetValue("admin_pass", pass); err != nil {
		responseBody.Msg = err.Error()
	}
	c.JSON(200, responseBody)
}

func adminPass() (string, bool, error) {
	pass, err := core.GetValue("admin_pass")
	if err == leveldb.ErrNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return pass, pass != "", nil
}

func registerAdmin(c *gin.Context) {
	responseBody := controller.ResponseBody{Msg: "success"}
	defer controller.TimeCost(time.Now(), &responseBody)
	if _, exists, err := adminPass(); err != nil {
		responseBody.Msg = err.Error()
		c.JSON(503, responseBody)
		return
	} else if exists {
		responseBody.Msg = "administrator account already exists"
		c.JSON(409, responseBody)
		return
	}
	pass := c.PostForm("password")
	if pass == "" {
		responseBody.Msg = "password is required"
		c.JSON(400, responseBody)
		return
	}
	if err := core.SetValue("admin_pass", pass); err != nil {
		responseBody.Msg = err.Error()
		c.JSON(503, responseBody)
		return
	}
	c.JSON(200, responseBody)
}

// RequestUsername returns the authenticated username for administrator-only controller calls.
func RequestUsername(c *gin.Context) string {
	return currentIdentity(c).Username
}

// Auth registers public authentication endpoints and returns the shared JWT middleware.
func Auth(r *gin.Engine, timeout int, secureCookie bool) *jwt.GinJWTMiddleware {
	jwtInit(timeout, secureCookie)

	newInstall := gin.H{"code": 201, "message": "No administrator account found inside the database", "data": nil}
	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 404, "message": "Page not found"})
	})
	r.GET("/auth/check", func(c *gin.Context) {
		if _, exists, err := adminPass(); err != nil {
			c.JSON(503, gin.H{"code": 503, "message": err.Error(), "data": nil})
		} else if !exists {
			c.JSON(201, newInstall)
		} else {
			title, err := core.GetValue("login_title")
			if err != nil {
				title = "trojan 管理平台"
			}
			c.JSON(200, gin.H{
				"code":    200,
				"message": "success",
				"data":    map[string]string{"title": title},
			})
		}
	})
	r.POST("/auth/login", authMiddleware.LoginHandler)
	r.POST("/auth/register", registerAdmin)
	authenticated := r.Group("/auth")
	authenticated.Use(authMiddleware.MiddlewareFunc())
	{
		authenticated.GET("/loginUser", func(c *gin.Context) {
			identity := currentIdentity(c)
			if identity.Role == roleUser && !validatePortalIdentity(identity) {
				c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "账户登录已失效"})
				return
			}
			c.JSON(200, gin.H{
				"code":    200,
				"message": "success",
				"data": map[string]string{
					"username": identity.Username,
					"role":     identity.Role,
				},
			})
		})
		authenticated.POST("/reset_pass", AdminOnly(), updateUser)
		authenticated.POST("/logout", authMiddleware.LogoutHandler)
		authenticated.POST("/refresh_token", authMiddleware.RefreshHandler)
	}
	return authMiddleware
}

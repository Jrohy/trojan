package web

import (
	"fmt"
	"github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"time"
	"trojan/core"
)

var (
	identityKey    = "id"
	authMiddleware *jwt.GinJWTMiddleware
	err            error
)

// Login auth用户验证结构体
type Login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func init() {
	authMiddleware, err = jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "k8s-manager",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		SendCookie:  true,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*Login); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &Login{
				Username: claims[identityKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals Login
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			userID := loginVals.Username
			pass := loginVals.Password
			if err != nil {
				return nil, err
			}
			if value, err := core.GetValue(userID + "_pass"); err != nil {
				return nil, err
			} else if value == pass {
				return &loginVals, nil
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*Login); ok && v.Username == "admin" {
				return true
			}
			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
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

func updateUser(c *gin.Context) {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	username := c.DefaultPostForm("username", "admin")
	pass := c.PostForm("password")
	err := core.SetValue(fmt.Sprintf("%s_pass", username), pass)
	if err != nil {
		responseBody.Msg = err.Error()
	}
	c.JSON(200, responseBody)
}

// Auth 权限router
func Auth(r *gin.Engine) *jwt.GinJWTMiddleware {
	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		fmt.Printf("NoRoute claims: %#v\n", claims)
		c.JSON(404, gin.H{"code": 404, "message": "Page not found"})
	})
	r.GET("/auth/check", func(c *gin.Context) {
		result, _ := core.GetValue("admin_pass")
		if result == "" {
			c.JSON(201, gin.H{"code": 201, "message": "No administrator account found inside the database"})
		} else {
			c.JSON(200, gin.H{"code": 200, "message": "administrator is ok!"})
		}
	})
	r.POST("/auth/login", authMiddleware.LoginHandler)
	r.POST("/auth/register", updateUser)
	authO := r.Group("/auth")
	authO.Use(authMiddleware.MiddlewareFunc())
	{
		authO.POST("/reset_pass", updateUser)
		authO.POST("/logout", authMiddleware.LogoutHandler)
		authO.POST("/refresh_token", authMiddleware.RefreshHandler)
	}
	return authMiddleware
}

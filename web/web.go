package web

import (
	"embed"
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/util"
	"trojan/web/controller"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed templates/*
var f embed.FS

func userRouter(router *gin.RouterGroup) {
	user := router.Group("/trojan/user")
	{
		user.GET("", func(c *gin.Context) {
			requestUser := RequestUsername(c)
			c.JSON(200, controller.UserList(requestUser))
		})
		user.GET("/page", func(c *gin.Context) {
			curPageStr := c.DefaultQuery("curPage", "1")
			pageSizeStr := c.DefaultQuery("pageSize", "10")
			curPage, _ := strconv.Atoi(curPageStr)
			pageSize, _ := strconv.Atoi(pageSizeStr)
			c.JSON(200, controller.PageUserList(curPage, pageSize))
		})
		user.POST("", func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")
			c.JSON(200, controller.CreateUser(username, password))
		})
		user.POST("/update", func(c *gin.Context) {
			sid := c.PostForm("id")
			username := c.PostForm("username")
			password := c.PostForm("password")
			id, _ := strconv.Atoi(sid)
			c.JSON(200, controller.UpdateUser(uint(id), username, password))
		})
		user.POST("/expire", func(c *gin.Context) {
			sid := c.PostForm("id")
			sDays := c.PostForm("useDays")
			id, _ := strconv.Atoi(sid)
			useDays, _ := strconv.Atoi(sDays)
			c.JSON(200, controller.SetExpire(uint(id), uint(useDays)))
		})
		user.DELETE("/expire", func(c *gin.Context) {
			sid := c.Query("id")
			id, _ := strconv.Atoi(sid)
			c.JSON(200, controller.CancelExpire(uint(id)))
		})
		user.DELETE("", func(c *gin.Context) {
			stringId := c.Query("id")
			id, _ := strconv.Atoi(stringId)
			c.JSON(200, controller.DelUser(uint(id), RequestUsername(c)))
		})
	}
}

func trojanRouter(router *gin.RouterGroup) {
	router.POST("/trojan/start", func(c *gin.Context) {
		c.JSON(200, controller.Start())
	})
	router.POST("/trojan/stop", func(c *gin.Context) {
		c.JSON(200, controller.Stop())
	})
	router.POST("/trojan/restart", func(c *gin.Context) {
		c.JSON(200, controller.Restart())
	})
	router.GET("/trojan/loglevel", func(c *gin.Context) {
		c.JSON(200, controller.GetLogLevel())
	})
	router.GET("/trojan/h2", func(c *gin.Context) {
		c.JSON(200, controller.H2Profile())
	})
	router.GET("/trojan/export", func(c *gin.Context) {
		result := controller.ExportCsv(c)
		if result != nil {
			c.JSON(200, result)
		}
	})
	router.POST("/trojan/import", func(c *gin.Context) {
		c.JSON(200, controller.ImportCsv(c))
	})
	router.POST("/trojan/update", func(c *gin.Context) {
		c.JSON(200, controller.Update())
	})
	router.POST("/trojan/switch", func(c *gin.Context) {
		tType := c.DefaultPostForm("type", "trojan")
		c.JSON(200, controller.SetTrojanType(tType))
	})
	router.POST("/trojan/loglevel", func(c *gin.Context) {
		slevel := c.DefaultPostForm("level", "1")
		level, _ := strconv.Atoi(slevel)
		c.JSON(200, controller.SetLogLevel(level))
	})
	router.POST("/trojan/h2/apply", func(c *gin.Context) {
		c.JSON(200, controller.ApplyH2Profile())
	})
	router.POST("/trojan/domain", func(c *gin.Context) {
		c.JSON(200, controller.SetDomain(c.PostForm("domain")))
	})
	router.GET("/trojan/log", func(c *gin.Context) {
		controller.Log(c)
	})
}

func dataRouter(router *gin.RouterGroup) {
	data := router.Group("/trojan/data")
	{
		data.POST("", func(c *gin.Context) {
			sID := c.PostForm("id")
			sQuota := c.PostForm("quota")
			id, _ := strconv.Atoi(sID)
			quota, _ := strconv.Atoi(sQuota)
			c.JSON(200, controller.SetData(uint(id), quota))
		})
		data.DELETE("", func(c *gin.Context) {
			sID := c.Query("id")
			id, _ := strconv.Atoi(sID)
			c.JSON(200, controller.CleanData(uint(id)))
		})
		data.POST("/resetDay", func(c *gin.Context) {
			dayStr := c.DefaultPostForm("day", "1")
			day, _ := strconv.Atoi(dayStr)
			c.JSON(200, controller.UpdateResetDay(uint(day)))
		})
		data.GET("/resetDay", func(c *gin.Context) {
			c.JSON(200, controller.GetResetDay())
		})
	}
}

func accessHistoryRouter(router *gin.RouterGroup) {
	history := router.Group("/trojan/access-history")
	{
		history.GET("", func(c *gin.Context) {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
			c.JSON(200, controller.AccessHistoryList(
				c.Query("startDate"),
				c.Query("endDate"),
				c.Query("userHash"),
				c.Query("targetHost"),
				c.Query("detailUserHash"),
				c.Query("sortBy"),
				c.Query("sortDir"),
				page,
				pageSize,
			))
		})
		history.GET("/status", func(c *gin.Context) {
			c.JSON(200, controller.AccessHistoryStatus())
		})
		history.POST("/collect", func(c *gin.Context) {
			c.JSON(200, controller.AccessHistoryCollect())
		})
	}
}

func commonRouter(router *gin.RouterGroup) {
	common := router.Group("/common")
	{
		common.GET("/ops", func(c *gin.Context) {
			c.JSON(200, controller.OpsOverview(c.DefaultQuery("range", "24h")))
		})
		common.GET("/version", func(c *gin.Context) {
			c.JSON(200, controller.Version())
		})
		common.GET("/serverInfo", func(c *gin.Context) {
			c.JSON(200, controller.ServerInfo())
		})
		common.GET("/overview", func(c *gin.Context) {
			c.JSON(200, controller.DashboardOverview(c.Query("refresh") == "1"))
		})
		common.GET("/activeUsers", func(c *gin.Context) {
			c.JSON(200, controller.ActiveUsers())
		})
		common.GET("/processMetrics", func(c *gin.Context) {
			c.JSON(200, controller.ProcessMetrics())
		})
		common.GET("/certInfo", func(c *gin.Context) {
			c.JSON(200, controller.CertInfo())
		})
		common.GET("/qrcode", func(c *gin.Context) {
			controller.QRCode(c)
		})
		common.POST("/cert/renew", func(c *gin.Context) {
			c.JSON(200, controller.RenewCert())
		})
		common.GET("/clashRules", func(c *gin.Context) {
			c.JSON(200, controller.GetClashRules())
		})
		common.POST("/clashRules", func(c *gin.Context) {
			rules := c.PostForm("rules")
			c.JSON(200, controller.SetClashRules(rules))
		})
		common.DELETE("/clashRules", func(c *gin.Context) {
			c.JSON(200, controller.ResetClashRules())
		})
		common.POST("/loginInfo", func(c *gin.Context) {
			c.JSON(200, controller.SetLoginInfo(c.PostForm("title")))
		})
	}
}

func portalRouter(router *gin.RouterGroup) {
	portal := router.Group("/portal")
	portal.Use(PortalOnly())
	portal.GET("/me/summary", func(c *gin.Context) {
		c.JSON(200, controller.PortalOverview(PortalUserID(c)))
	})
}

func staticRouter(router *gin.Engine) {
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || c.Request.URL.Path == "/portal" || strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	staticFs, _ := fs.Sub(f, "templates/static")
	router.StaticFS("/static", http.FS(staticFs))

	router.GET("/", func(c *gin.Context) {
		indexHTML, _ := f.ReadFile("templates/" + "index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	router.GET("/portal", func(c *gin.Context) {
		portalHTML, _ := f.ReadFile("templates/" + "portal.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", portalHTML)
	})
}

func noTokenRouter(router *gin.Engine) {
	router.GET("/trojan/user/subscribe", func(c *gin.Context) {
		controller.ClashSubInfo(c)
	})
}

func newRouter(timeout int, isSSL bool) *gin.Engine {
	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(router)
	noTokenRouter(router)
	auth := Auth(router, timeout, isSSL)
	protected := router.Group("")
	protected.Use(auth.MiddlewareFunc())
	portalRouter(protected)
	admin := protected.Group("")
	admin.Use(AdminOnly())
	trojanRouter(admin)
	userRouter(admin)
	dataRouter(admin)
	accessHistoryRouter(admin)
	commonRouter(admin)
	return router
}

// Start web启动入口
func Start(host string, port, timeout int, isSSL bool) {
	router := newRouter(timeout, isSSL)
	controller.ScheduleTask()
	controller.AccessHistoryScheduleTask()
	controller.CollectTask()
	controller.StartOpsMonitor()
	util.OpenPort(port)
	if isSSL {
		config := core.GetConfig()
		ssl := &config.SSl
		router.RunTLS(fmt.Sprintf("%s:%d", host, port), ssl.Cert, ssl.Key)
	} else {
		if port == 80 {
			go func() {
				addr := "127.0.0.1:81"
				server := &http.Server{
					Addr:    addr,
					Handler: h2c.NewHandler(router, &http2.Server{}),
				}
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					fmt.Println("h2c server error: " + err.Error())
				}
			}()
		}
		router.Run(fmt.Sprintf("%s:%d", host, port))
	}
}

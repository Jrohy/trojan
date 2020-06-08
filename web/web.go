package web

import (
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
	"trojan/core"
	"trojan/util"
	"trojan/web/controller"
)

func userRouter(router *gin.Engine) {
	user := router.Group("/trojan/user")
	{
		user.GET("", func(c *gin.Context) {
			requestUser := RequestUsername(c)
			if requestUser == "admin" {
				c.JSON(200, controller.UserList(""))
			} else {
				c.JSON(200, controller.UserList(requestUser))
			}
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
		user.DELETE("", func(c *gin.Context) {
			stringId := c.Query("id")
			id, _ := strconv.Atoi(stringId)
			c.JSON(200, controller.DelUser(uint(id)))
		})
	}
}

func trojanRouter(router *gin.Engine) {
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
	router.POST("/trojan/update", func(c *gin.Context) {
		c.JSON(200, controller.Update())
	})
	router.POST("/trojan/loglevel", func(c *gin.Context) {
		slevel := c.DefaultPostForm("level", "1")
		level, _ := strconv.Atoi(slevel)
		c.JSON(200, controller.SetLogLevel(level))
	})
	router.GET("/trojan/log", func(c *gin.Context) {
		controller.Log(c)
	})
}

func dataRouter(router *gin.Engine) {
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
	}
}

func dateRouter(router *gin.Engine) {
	data := router.Group("/trojan/enddate")
	{
		data.POST("", func(c *gin.Context) {
			sID := c.PostForm("id")
			sEnddate := c.PostForm("enddate")
			id, _ := strconv.Atoi(sID)
			enddata, _ := time.Parse("2006-01-02", sEnddate)
			c.JSON(200, controller.EndDate(uint(id), enddata))
		})
	}
}

func commonRouter(router *gin.Engine) {
	common := router.Group("/common")
	{
		common.GET("/version", func(c *gin.Context) {
			c.JSON(200, controller.Version())
		})
		common.GET("/serverInfo", func(c *gin.Context) {
			c.JSON(200, controller.ServerInfo())
		})
		common.POST("/loginInfo", func(c *gin.Context) {
			c.JSON(200, controller.SetLoginInfo(c.PostForm("title")))
		})
	}
}

func staticRouter(router *gin.Engine) {
	box := packr.New("trojanBox", "./templates")
	router.Use(func(c *gin.Context) {
		requestUrl := c.Request.URL.Path
		if box.Has(requestUrl) || requestUrl == "/" {
			http.FileServer(box).ServeHTTP(c.Writer, c.Request)
			c.Abort()
		}
	})
}

// Start web启动入口
func Start(port int, isSSL bool) {
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(router)
	router.Use(Auth(router).MiddlewareFunc())
	trojanRouter(router)
	userRouter(router)
	dataRouter(router)
	dateRouter(router)
	commonRouter(router)
	util.OpenPort(port)
	if isSSL {
		config := core.Load("")
		ssl := &config.SSl
		router.RunTLS(fmt.Sprintf(":%d", port), ssl.Cert, ssl.Key)
	} else {
		router.Run(fmt.Sprintf(":%d", port))
	}
}

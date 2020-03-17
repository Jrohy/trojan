package web

import (
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
	"net/http"
	"strconv"
	"trojan/web/controller"
)

func userRouter(router *gin.Engine) {
	user := router.Group("/trojan/user")
	{
		user.GET("", func(c *gin.Context) {
			c.JSON(200, controller.UserList())
		})
		user.POST("", func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")
			c.JSON(200, controller.CreateUser(username, password))
		})
		user.DELETE("", func(c *gin.Context) {
			stringId := c.Query("id")
			id, _ := strconv.Atoi(stringId)
			c.JSON(200, controller.DelUser(uint(id)))
		})
	}
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

func commonRouter(router *gin.Engine) {
	common := router.Group("/common")
	{
		common.GET("/version", func(c *gin.Context) {
			c.JSON(200, controller.Version())
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
func Start(port int) {
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(router)
	router.Use(Auth(router).MiddlewareFunc())
	userRouter(router)
	dataRouter(router)
	commonRouter(router)
	_ = router.Run(fmt.Sprintf(":%d", port))
}

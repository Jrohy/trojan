package web

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"trojan/core"
)

// ResponseBody 结构体
type ResponseBody struct {
	Duration string
	Data     interface{}
	Msg      string
}

// TimeCost web函数执行用时统计方法
func TimeCost(start time.Time, body *ResponseBody) {
	body.Duration = time.Since(start).String()
}

func userList() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	userList := *mysql.GetData()
	responseBody.Data = userList
	return &responseBody
}

func createUser(username string, password string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	pass, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		responseBody.Msg = "Base64解码失败: " + err.Error()
		return &responseBody
	}
	if err := mysql.CreateUser(username, string(pass)); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func delUser(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.DeleteUser(id); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func setData(id uint, quota int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.SetQuota(id, quota); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func cleanData(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.CleanData(id); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func userRouter(router *gin.Engine) {
	user := router.Group("/trojan/user")
	{
		user.GET("", func(c *gin.Context) {
			c.JSON(200, userList())
		})
		user.POST("", func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")
			c.JSON(200, createUser(username, password))
		})
		user.DELETE("", func(c *gin.Context) {
			stringId := c.PostForm("id")
			id, _ := strconv.Atoi(stringId)
			c.JSON(200, delUser(uint(id)))
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
			c.JSON(200, setData(uint(id), quota))
		})
		data.DELETE("", func(c *gin.Context) {
			sID := c.PostForm("id")
			id, _ := strconv.Atoi(sID)
			c.JSON(200, cleanData(uint(id)))
		})
	}
}

// Start web启动入口
func Start() {
	router := gin.Default()
	router.Use(Auth(router).MiddlewareFunc())
	userRouter(router)
	dataRouter(router)
	_ = router.Run(":80")
}

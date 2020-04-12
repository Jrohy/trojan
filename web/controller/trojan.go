package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"log"
	"time"
	"trojan/core"
	"trojan/trojan"
	websocket "trojan/util"
)

// Start 启动trojan
func Start() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Start()
	return &responseBody
}

// Stop 停止trojan
func Stop() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Stop()
	return &responseBody
}

// Restart 重启trojan
func Restart() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Restart()
	return &responseBody
}

// Status trojan状态
func Status() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	responseBody.Data = map[string]interface{}{
		"status":  trojan.Status(false),
		"version": trojan.Version(),
	}
	return &responseBody
}

// Update trojan更新
func Update() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.InstallTrojan()
	return &responseBody
}

// LogLevel 修改trojan日志等级
func LogLevel(level int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	core.WriteLogLevel(level)
	trojan.Restart()
	return &responseBody
}

// Log 通过ws查看trojan实时日志
func Log(c *gin.Context) {
	var (
		wsConn *websocket.WsConnection
		err    error
	)
	if wsConn, err = websocket.InitWebsocket(c.Writer, c.Request); err != nil {
		fmt.Println(err)
		return
	}
	defer wsConn.WsClose()
	param := c.DefaultQuery("line", "300")
	if param == "-1" {
		param = "--no-tail"
	} else {
		param = "-n " + param
	}
	result, err := trojan.LogChan(param)
	if err != nil {
		fmt.Println(err)
		wsConn.WsClose()
		return
	}
	for line := range *result {
		if err := wsConn.WsWrite(ws.TextMessage, []byte(line)); err != nil {
			log.Println("can't send: ", line)
			break
		}
	}
}

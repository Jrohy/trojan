package controller

import (
	"time"
	"trojan/trojan"
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

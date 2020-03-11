package controller

import (
	"time"
	"trojan/core"
)

// SetData 设置流量限制
func SetData(id uint, quota int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.SetQuota(id, quota); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// CleanData 清空流量
func CleanData(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.CleanData(id); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

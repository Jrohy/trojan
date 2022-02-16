package controller

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"strconv"
	"time"
	"trojan/core"
	"trojan/trojan"
)

// UserList 获取用户列表
func UserList(requestUser string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	userList, err := mysql.GetData()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if requestUser != "admin" {
		findUser := false
		for _, user := range userList {
			if user.Username == requestUser {
				userList = []*core.User{user}
				findUser = true
				break
			}
		}
		if !findUser {
			userList = []*core.User{}
		}
	}
	domain, port := trojan.GetDomainAndPort()
	responseBody.Data = map[string]interface{}{
		"domain":   domain,
		"port":     port,
		"userList": userList,
	}
	return &responseBody
}

// PageUserList 分页查询获取用户列表
func PageUserList(curPage int, pageSize int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	pageData, err := mysql.PageList(curPage, pageSize)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	domain, port := trojan.GetDomainAndPort()
	responseBody.Data = map[string]interface{}{
		"domain":   domain,
		"port":     port,
		"pageData": pageData,
	}
	return &responseBody
}

// CreateUser 创建用户
func CreateUser(username string, password string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if username == "admin" {
		responseBody.Msg = "不能创建用户名为admin的用户!"
		return &responseBody
	}
	mysql := core.GetMysql()
	if user := mysql.GetUserByName(username); user != nil {
		responseBody.Msg = "已存在用户名为: " + username + " 的用户!"
		return &responseBody
	}
	pass, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		responseBody.Msg = "Base64解码失败: " + err.Error()
		return &responseBody
	}
	if user := mysql.GetUserByPass(password); user != nil {
		responseBody.Msg = "已存在密码为: " + string(pass) + " 的用户!"
		return &responseBody
	}
	if err := mysql.CreateUser(username, password, string(pass)); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// UpdateUser 更新用户
func UpdateUser(id uint, username string, password string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if username == "admin" {
		responseBody.Msg = "不能更改用户名为admin的用户!"
		return &responseBody
	}
	mysql := core.GetMysql()
	userList, err := mysql.GetData(strconv.Itoa(int(id)))
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if userList[0].Username != username {
		if user := mysql.GetUserByName(username); user != nil {
			responseBody.Msg = "已存在用户名为: " + username + " 的用户!"
			return &responseBody
		}
	}
	pass, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		responseBody.Msg = "Base64解码失败: " + err.Error()
		return &responseBody
	}
	if userList[0].Password != password {
		if user := mysql.GetUserByPass(password); user != nil {
			responseBody.Msg = "已存在密码为: " + string(pass) + " 的用户!"
			return &responseBody
		}
	}
	if err := mysql.UpdateUser(id, username, password, string(pass)); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// DelUser 删除用户
func DelUser(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.DeleteUser(id); err != nil {
		responseBody.Msg = err.Error()
	} else {
		trojan.Restart()
	}
	return &responseBody
}

// SetExpire 设置用户过期
func SetExpire(id uint, useDays uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.SetExpire(id, useDays); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// CancelExpire 取消设置用户过期
func CancelExpire(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.CancelExpire(id); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// ClashSubInfo 获取clash订阅信息
func ClashSubInfo(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(200, "token is null")
		return
	}
	decodeByte, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		c.String(200, "token is error")
		return
	}
	if !gjson.GetBytes(decodeByte, "user").Exists() || !gjson.GetBytes(decodeByte, "pass").Exists() {
		c.String(200, "token is error")
		return
	}
	username := gjson.GetBytes(decodeByte, "user").String()
	password := gjson.GetBytes(decodeByte, "pass").String()

	mysql := core.GetMysql()
	user := mysql.GetUserByName(username)
	if user != nil {
		pass, _ := base64.StdEncoding.DecodeString(user.Password)
		if password == string(pass) {
			var wsData, wsHost string
			userInfo := fmt.Sprintf("upload=%d, download=%d", user.Upload, user.Download)
			if user.Quota != -1 {
				userInfo = fmt.Sprintf("%s, total=%d", userInfo, user.Quota)
			}
			if user.ExpiryDate != "" {
				utc, _ := time.LoadLocation("Asia/Shanghai")
				t, _ := time.ParseInLocation("2006-01-02", user.ExpiryDate, utc)
				userInfo = fmt.Sprintf("%s, expire=%d", userInfo, t.Unix())
			}
			c.Header("content-disposition", fmt.Sprintf("attachment; filename=%s", user.Username))
			c.Header("subscription-userinfo", userInfo)

			domain, port := trojan.GetDomainAndPort()
			name := fmt.Sprintf("%s:%d", domain, port)
			configData := string(core.Load(""))
			if gjson.Get(configData, "websocket").Exists() && gjson.Get(configData, "websocket.enabled").Bool() {
				if gjson.Get(configData, "websocket.host").Exists() {
					hostTemp := gjson.Get(configData, "websocket.host").String()
					if hostTemp != "" {
						wsHost = fmt.Sprintf(", headers: {Host: %s}", hostTemp)
					}
				}
				wsOpt := fmt.Sprintf("{path: %s%s}", gjson.Get(configData, "websocket.path").String(), wsHost)
				wsData = fmt.Sprintf(", network: ws, udp: true, ws-opts: %s", wsOpt)
			}
			proxyData := fmt.Sprintf("  - {name: %s, server: %s, port: %d, type: trojan, password: %s, sni: %s%s}",
				name, domain, port, password, domain, wsData)
			result := fmt.Sprintf(`proxies:
%s

proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - %s

%s
`, proxyData, name, clashRules())
			c.String(200, result)
			return
		}
	}
	c.String(200, "token is error")
}

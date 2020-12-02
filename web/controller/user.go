package controller

import (
	"encoding/base64"
	"strconv"
	"time"
	"trojan/core"
	"trojan/trojan"
)

// UserList 获取用户列表
func UserList(findUser string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	userList, err := mysql.GetData()
	if findUser != "" {
		for _, user := range userList {
			if user.Username == findUser {
				userList = []*core.User{user}
				break
			}
		}
	}
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
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
	}
	return &responseBody
}

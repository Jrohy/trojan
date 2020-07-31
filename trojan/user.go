package trojan

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"trojan/core"
	"trojan/util"
)

// UserMenu 用户管理菜单
func UserMenu() {
	fmt.Println()
	menu := []string{"新增用户", "删除用户", "限制流量", "清空流量"}
	switch util.LoopInput("请选择: ", menu, true) {
	case 1:
		AddUser()
	case 2:
		DelUser()
	case 3:
		SetUserQuota()
	case 4:
		CleanData()
	}
}

// AddUser 添加用户
func AddUser() {
	randomUser := util.RandString(4)
	randomPass := util.RandString(8)
	inputUser := util.Input(fmt.Sprintf("生成随机用户名: %s, 使用直接回车, 否则输入自定义用户名: ", randomUser), randomUser)
	if inputUser == "admin" {
		fmt.Println(util.Yellow("不能新建用户名为'admin'的用户!"))
		return
	}
	mysql := core.GetMysql()
	if user := mysql.GetUserByName(inputUser); user != nil {
		fmt.Println(util.Yellow("已存在用户名为: " + inputUser + " 的用户!"))
		return
	}
	inputPass := util.Input(fmt.Sprintf("生成随机密码: %s, 使用直接回车, 否则输入自定义密码: ", randomPass), randomPass)
	base64Pass := base64.StdEncoding.EncodeToString([]byte(inputPass))
	if user := mysql.GetUserByPass(base64Pass); user != nil {
		fmt.Println(util.Yellow("已存在密码为: " + inputPass + " 的用户!"))
		return
	}
	if mysql.CreateUser(inputUser, base64Pass, inputPass) == nil {
		fmt.Println("新增用户成功!")
	}
}

// DelUser 删除用户
func DelUser() {
	userList := UserList()
	mysql := core.GetMysql()
	choice := util.LoopInput("请选择要删除的用户序号: ", userList, true)
	if choice == -1 {
		return
	}
	if mysql.DeleteUser(userList[choice-1].ID) == nil {
		fmt.Println("删除用户成功!")
	}
}

// SetUserQuota 限制用户流量
func SetUserQuota() {
	var (
		limit int
		err   error
	)
	userList := UserList()
	mysql := core.GetMysql()
	choice := util.LoopInput("请选择要限制流量的用户序号: ", userList, true)
	if choice == -1 {
		return
	}
	for {
		quota := util.Input("请输入用户"+userList[choice-1].Username+"限制的流量大小(单位byte)", "")
		limit, err = strconv.Atoi(quota)
		if err != nil {
			fmt.Printf("%s 不是数字, 请重新输入!\n", quota)
		} else {
			break
		}
	}
	if mysql.SetQuota(userList[choice-1].ID, limit) == nil {
		fmt.Println("成功设置用户" + userList[choice-1].Username + "限制流量" + util.Bytefmt(uint64(limit)))
	}
}

// CleanData 清空用户流量
func CleanData() {
	userList := UserList()
	mysql := core.GetMysql()
	choice := util.LoopInput("请选择要清空流量的用户序号: ", userList, true)
	if choice == -1 {
		return
	}
	if mysql.CleanData(userList[choice-1].ID) == nil {
		fmt.Println("清空流量成功!")
	}
}

// UserList 获取用户列表并打印显示
func UserList(ids ...string) []*core.User {
	mysql := core.GetMysql()
	userList, err := mysql.GetData(ids...)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	domain, port := GetDomainAndPort()
	for i, k := range userList {
		pass, err := base64.StdEncoding.DecodeString(k.Password)
		if err != nil {
			pass = []byte("")
		}
		fmt.Printf("%d.\n", i+1)
		fmt.Println("用户名: " + k.Username)
		fmt.Println("密码: " + string(pass))
		fmt.Println("上传流量: " + util.Cyan(util.Bytefmt(k.Upload)))
		fmt.Println("下载流量: " + util.Cyan(util.Bytefmt(k.Download)))
		if k.Quota < 0 {
			fmt.Println("流量限额: " + util.Cyan("无限制"))
		} else {
			fmt.Println("流量限额: " + util.Cyan(util.Bytefmt(uint64(k.Quota))))
		}
		fmt.Println("分享链接: " + util.Green(fmt.Sprintf("trojan://%s@%s:%d", string(pass), domain, port)))
		fmt.Println()
	}
	return userList
}

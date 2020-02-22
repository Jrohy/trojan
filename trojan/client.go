package trojan

import (
	"fmt"
	"trojan/core"
	"trojan/database"
	"trojan/util"
)

var clientPath = "/root/config.json"

func GenClientJson() {
	fmt.Println()
	var user core.User
	domain, err := database.GetValue("domain")
	if err != nil {
		fmt.Println("无域名记录, 生成的配置文件需手填域名字段(ssl.sni)")
		domain = ""
	}
	ip := util.GetLocalIP()
	mysql := core.GetMysql()
	userList := *mysql.GetData()
	if len(userList) == 1 {
		user = userList[0]
	} else {
		UserList()
		choice := util.LoopInput("请选择要生成配置文件的用户序号: ", userList)
		if choice < 0 {
			return
		}
		user = userList[choice -1]
	}
	password, err := database.GetValue(user.Username + "_pass")
	if err != nil {
		fmt.Println("无法获取选择用户的原始密码, 生成配置文件失败!")
		return
	}
	if !core.WriteClient(password, ip, domain, clientPath) {
		fmt.Println("生成配置文件失败!")
	} else {
		fmt.Println("成功生成配置文件: " + clientPath)
	}
}

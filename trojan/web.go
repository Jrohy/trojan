package trojan

import (
	"crypto/sha256"
	"fmt"
	"trojan/core"
	"trojan/util"
)

// WebMenu web管理菜单
func WebMenu() {
	fmt.Println()
	menu := []string{"重置web管理员密码", "修改显示的域名(非申请证书)"}
	switch util.LoopInput("请选择: ", menu, true) {
	case 1:
		ResetAdminPass()
	case 2:
		SetDomain("")
	}
}

// ResetAdminPass 重置管理员密码
func ResetAdminPass() {
	inputPass := util.Input("请输入admin用户密码: ", "")
	if inputPass == "" {
		fmt.Println("撤销更改!")
	} else {
		encryPass := sha256.Sum224([]byte(inputPass))
		err := core.SetValue("admin_pass", fmt.Sprintf("%x", encryPass))
		if err == nil {
			fmt.Println(util.Green("重置admin密码成功!"))
		} else {
			fmt.Println(err)
		}
	}
}

// SetDomain 设置显示的域名
func SetDomain(domain string) {
	if domain == "" {
		domain = util.Input("请输入要显示的域名地址: ", "")
	}
	if domain == "" {
		fmt.Println("撤销更改!")
	} else {
		core.WriteDomain(domain)
		Restart()
		fmt.Println("修改domain成功!")
	}
}

// GetDomainAndPort 获取域名和端口
func GetDomainAndPort() (string, int) {
	config := core.Load("")
	return config.SSl.Sni, config.LocalPort
}

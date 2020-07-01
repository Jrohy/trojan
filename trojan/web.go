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

package trojan

import (
	"fmt"
	"trojan/util"
)

func ControllMenu() {
	fmt.Println()
	menu := []string{"启动trojan", "停止trojan", "重启trojan", "查看trojan状态"}
	switch util.LoopInput("请选择: ", menu, true) {
	case 1:
		Start()
	case 2:
		Stop()
	case 3:
		Restart()
	case 4:
		Status()
	}
}

func Restart() {
	if err := util.ExecCommand("systemctl restart trojan"); err != nil {
		fmt.Println(util.Red("重启trojan失败!"))
	} else {
		fmt.Println(util.Green("重启trojan成功!"))
	}
}

func Start() {
	if err := util.ExecCommand("systemctl start trojan"); err != nil {
		fmt.Println(util.Red("启动trojan失败!"))
	} else {
		fmt.Println(util.Green("启动trojan成功!"))
	}
}

func Stop() {
	if err := util.ExecCommand("systemctl stop trojan"); err != nil {
		fmt.Println(util.Red("停止trojan失败!"))
	} else {
		fmt.Println(util.Green("停止trojan成功!"))
	}
}

func Status() {
	util.ExecCommand("systemctl status trojan")
}

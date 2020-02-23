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
	util.ExecCommand("systemctl restart trojan")
}

func Start() {
	util.ExecCommand("systemctl start trojan")
}

func Stop() {
	util.ExecCommand("systemctl stop trojan")
}

func Status() {
	util.ExecCommand("systemctl status trojan")
}

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
	if util.IsExists("/.dockerenv") {
		Stop()
		Start()
		fmt.Println(util.Green("重启trojan成功!"))
	} else {
		if err := util.ExecCommand("systemctl restart trojan"); err != nil {
			fmt.Println(util.Red("重启trojan失败!"))
		} else {
			fmt.Println(util.Green("重启trojan成功!"))
		}
	}
}

func Start() {
	if util.IsExists("/.dockerenv") {
		check := util.ExecCommandWithResult(`ps aux|grep "/usr/bin/trojan/trojan"|grep -v grep`)
		if check != "" {
			Stop()
		}
		util.ExecCommand(`echo "" > /.run.log`)
		util.StartProcess("/usr/bin/trojan/trojan", "-c", "/usr/local/etc/trojan/config.json", "-l", "/.run.log")
		fmt.Println(util.Green("启动trojan成功!"))
	} else {
		if err := util.ExecCommand("systemctl start trojan"); err != nil {
			fmt.Println(util.Red("启动trojan失败!"))
		} else {
			fmt.Println(util.Green("启动trojan成功!"))
		}
	}
}

func Stop() {
	if util.IsExists("/.dockerenv") {
		util.ExecCommandWithResult(`ps aux|grep "/usr/bin/trojan/trojan"|grep -v grep|awk '{print $2}'|xargs -r kill -9`)
		fmt.Println(util.Green("停止trojan成功!"))
	} else {
		if err := util.ExecCommand("systemctl stop trojan"); err != nil {
			fmt.Println(util.Red("停止trojan失败!"))
		} else {
			fmt.Println(util.Green("停止trojan成功!"))
		}
	}
}

func Status() {
	if util.IsExists("/.dockerenv") {
		if util.ExecCommandWithResult("cat /.run.log|grep FATAL") != "" {
			fmt.Println(util.ExecCommandWithResult("cat /.run.log"))
		} else {
			fmt.Println(util.Green("trojan running..."))
		}
	} else {
		util.ExecCommand("systemctl status trojan")
	}
}

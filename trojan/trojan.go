package trojan

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/util"
)

// ControllMenu Trojan控制菜单
func ControllMenu() {
	fmt.Println()
	tType := Type()
	if tType == "trojan" {
		tType = "trojan-go"
	} else {
		tType = "trojan"
	}
	menu := []string{"启动trojan", "停止trojan", "重启trojan", "查看trojan状态", "查看trojan日志", "修改trojan端口"}
	menu = append(menu, "切换为"+tType)
	switch util.LoopInput("请选择: ", menu, true) {
	case 1:
		Start()
	case 2:
		Stop()
	case 3:
		Restart()
	case 4:
		Status(true)
	case 5:
		go util.Log("trojan", 300)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		//阻塞
		<-c
	case 6:
		ChangePort()
	case 7:
		if err := SwitchType(tType); err != nil {
			fmt.Println(err)
		}
	}
}

// Restart 重启trojan
func Restart() {
	util.OpenPort(core.GetConfig().LocalPort)
	util.SystemctlRestart("trojan")
}

// Start 启动trojan
func Start() {
	util.OpenPort(core.GetConfig().LocalPort)
	util.SystemctlStart("trojan")
}

// Stop 停止trojan
func Stop() {
	util.SystemctlStop("trojan")
}

// Status 获取trojan状态
func Status(isPrint bool) string {
	result := util.SystemctlStatus("trojan")
	if isPrint {
		fmt.Println(result)
	}
	return result
}

// UpTime Trojan运行时间
func UpTime() string {
	result := strings.TrimSpace(util.ExecCommandWithResult("ps -Ao etime,args|grep -v grep|grep /usr/local/etc/trojan/config.json"))
	resultSlice := strings.Split(result, " ")
	if len(resultSlice) > 0 {
		return resultSlice[0]
	}
	return ""
}

// ChangePort 修改trojan端口
func ChangePort() {
	config := core.GetConfig()
	oldPort := config.LocalPort
	randomPort := util.RandomPort()
	fmt.Println("当前trojan端口: " + util.Green(strconv.Itoa(oldPort)))
	newPortStr := util.Input(fmt.Sprintf("请输入新的trojan端口(若要使用随机端口%s直接回车即可): ", util.Blue(strconv.Itoa(randomPort))), strconv.Itoa(randomPort))
	newPort, err := strconv.Atoi(newPortStr)
	if err != nil {
		fmt.Println("修改端口失败: " + err.Error())
		return
	}
	if core.WritePort(newPort) {
		util.OpenPort(newPort)
		fmt.Println(util.Green("端口修改成功!"))
		Restart()
	} else {
		fmt.Println(util.Red("端口修改成功!"))
	}
}

// Version Trojan版本
func Version() string {
	flag := "-v"
	if Type() == "trojan-go" {
		flag = "-version"
	}
	result := strings.TrimSpace(util.ExecCommandWithResult("/usr/bin/trojan/trojan " + flag))
	if len(result) == 0 {
		return ""
	}
	firstLine := strings.Split(result, "\n")[0]
	tempSlice := strings.Split(firstLine, " ")
	return tempSlice[len(tempSlice)-1]
}

// SwitchType 切换Trojan类型
func SwitchType(tType string) error {
	ARCH := runtime.GOARCH
	if ARCH != "amd64" && ARCH != "arm64" {
		return errors.New("not support " + ARCH + " machine")
	}
	if tType == "trojan" && ARCH != "amd64" {
		return errors.New("trojan not support " + ARCH + " machine")
	}
	if err := core.SetValue("trojanType", tType); err != nil {
		return err
	}
	InstallTrojan("")
	return nil
}

// Type Trojan类型
func Type() string {
	tType, _ := core.GetValue("trojanType")
	if tType == "" {
		if strings.Contains(Status(false), "trojan-go") {
			tType = "trojan-go"
		} else {
			tType = "trojan"
		}
		_ = core.SetValue("trojanType", tType)
	}
	return tType
}

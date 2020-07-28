package trojan

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/util"
)

// ControllMenu Trojan控制菜单
func ControllMenu() {
	fmt.Println()
	menu := []string{"启动trojan", "停止trojan", "重启trojan", "查看trojan状态", "查看trojan日志"}
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
		go Log(300)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		//阻塞
		<-c
	}
}

// Restart 重启trojan
func Restart() {
	if err := util.ExecCommand("systemctl restart trojan"); err != nil {
		fmt.Println(util.Red("重启trojan失败!"))
	} else {
		fmt.Println(util.Green("重启trojan成功!"))
	}
}

// Start 启动trojan
func Start() {
	if err := util.ExecCommand("systemctl start trojan"); err != nil {
		fmt.Println(util.Red("启动trojan失败!"))
	} else {
		fmt.Println(util.Green("启动trojan成功!"))
	}
}

// Stop 停止trojan
func Stop() {
	if err := util.ExecCommand("systemctl stop trojan"); err != nil {
		fmt.Println(util.Red("停止trojan失败!"))
	} else {
		fmt.Println(util.Green("停止trojan成功!"))
	}
}

// Status 获取trojan状态
func Status(isPrint bool) string {
	result := util.ExecCommandWithResult("systemctl status trojan")
	if isPrint {
		fmt.Println(result)
	}
	return result
}

// RunTime Trojan运行时间
func RunTime() string {
	result := strings.TrimSpace(util.ExecCommandWithResult("ps -Ao etime,args|grep -v grep|grep /usr/local/etc/trojan/config.json"))
	resultSlice := strings.Split(result, " ")
	if len(resultSlice) > 0 {
		return resultSlice[0]
	}
	return ""
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

// Log 实时打印trojan日志
func Log(line int) {
	result, _ := LogChan("-n "+strconv.Itoa(line), make(chan byte))
	for line := range result {
		fmt.Println(line)
	}
}

// LogChan trojan实时日志, 返回chan
func LogChan(param string, closeChan chan byte) (chan string, error) {
	cmd := exec.Command("bash", "-c", "journalctl -f -u trojan "+param)

	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err: ", err.Error())
		return nil, err
	}
	ch := make(chan string, 100)
	stdoutScan := bufio.NewScanner(stdout)
	go func() {
		for stdoutScan.Scan() {
			select {
			case <-closeChan:
				stdout.Close()
				return
			default:
				ch <- stdoutScan.Text()
			}
		}
	}()
	return ch, nil
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

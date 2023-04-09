package util

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

func systemctlReplace(out string) (bool, error) {
	var (
		err       error
		isReplace bool
	)
	if IsExists("/.dockerenv") && strings.Contains(out, "Failed to get D-Bus") {
		isReplace = true
		fmt.Println(Yellow("正在下载并替换适配的systemctl。。"))
		if err = ExecCommand("curl -L https://raw.githubusercontent.com/gdraheim/docker-systemctl-replacement/master/files/docker/systemctl.py -o /usr/bin/systemctl && chmod +x /usr/bin/systemctl"); err != nil {
			return isReplace, err
		}
		fmt.Println()
	}
	return isReplace, err
}

func systemctlBase(name, operate string) (string, error) {
	out, err := exec.Command("bash", "-c", fmt.Sprintf("systemctl %s %s", operate, name)).CombinedOutput()
	if v, _ := systemctlReplace(string(out)); v {
		out, err = exec.Command("bash", "-c", fmt.Sprintf("systemctl %s %s", operate, name)).CombinedOutput()
	}
	return string(out), err
}

// SystemctlStart 服务启动
func SystemctlStart(name string) {
	if _, err := systemctlBase(name, "start"); err != nil {
		fmt.Println(Red(fmt.Sprintf("启动%s失败!", name)))
	} else {
		fmt.Println(Green(fmt.Sprintf("启动%s成功!", name)))
	}
}

// SystemctlStop 服务停止
func SystemctlStop(name string) {
	if _, err := systemctlBase(name, "stop"); err != nil {
		fmt.Println(Red(fmt.Sprintf("停止%s失败!", name)))
	} else {
		fmt.Println(Green(fmt.Sprintf("停止%s成功!", name)))
	}
}

// SystemctlRestart 服务重启
func SystemctlRestart(name string) {
	if _, err := systemctlBase(name, "restart"); err != nil {
		fmt.Println(Red(fmt.Sprintf("重启%s失败!", name)))
	} else {
		fmt.Println(Green(fmt.Sprintf("重启%s成功!", name)))
	}
}

// SystemctlEnable 服务设置开机自启
func SystemctlEnable(name string) {
	if _, err := systemctlBase(name, "enable"); err != nil {
		fmt.Println(Red(fmt.Sprintf("设置%s开机自启失败!", name)))
	}
}

// SystemctlStatus 服务状态查看
func SystemctlStatus(name string) string {
	out, _ := systemctlBase(name, "status")
	return out
}

// CheckCommandExists 检查命令是否存在
func CheckCommandExists(command string) bool {
	if _, err := exec.LookPath(command); err != nil {
		return false
	}
	return true
}

// RunWebShell 运行网上的脚本
func RunWebShell(webShellPath string) {
	if !strings.HasPrefix(webShellPath, "http") && !strings.HasPrefix(webShellPath, "https") {
		fmt.Printf("shell path must start with http or https!")
		return
	}
	resp, err := http.Get(webShellPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()
	installShell, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	ExecCommand(string(installShell))
}

// ExecCommand 运行命令并实时查看运行结果
func ExecCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err: ", err.Error())
		return err
	}
	ch := make(chan string, 100)
	stdoutScan := bufio.NewScanner(stdout)
	stderrScan := bufio.NewScanner(stderr)
	go func() {
		for stdoutScan.Scan() {
			line := stdoutScan.Text()
			ch <- line
		}
	}()
	go func() {
		for stderrScan.Scan() {
			line := stderrScan.Text()
			ch <- line
		}
	}()
	var err error
	go func() {
		err = cmd.Wait()
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			fmt.Println("wait:", err.Error())
		}
		close(ch)
	}()
	for line := range ch {
		fmt.Println(line)
	}
	return err
}

// ExecCommandWithResult 运行命令并获取结果
func ExecCommandWithResult(command string) string {
	out, err := exec.Command("bash", "-c", command).CombinedOutput()
	if strings.Contains(command, "systemctl") {
		if v, _ := systemctlReplace(string(out)); v {
			out, err = exec.Command("bash", "-c", command).CombinedOutput()
		}
	}
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		fmt.Println("err: " + err.Error())
		return ""
	}
	return string(out)
}

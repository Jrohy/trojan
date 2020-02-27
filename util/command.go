package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

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
	installShell, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	ExecCommand(string(installShell))
}

func asyncLog(reader io.ReadCloser) error {
	cache := "" //缓存不足一行的日志信息
	buf := make([]byte, 1024)
	for {
		num, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if num > 0 {
			b := buf[:num]
			s := strings.Split(string(b), "\n")
			line := strings.Join(s[:len(s)-1], "\n") //取出整行的日志
			fmt.Printf("%s%s\n", cache, line)
			cache = s[len(s)-1]
		}
	}
	return nil
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

	go asyncLog(stdout)
	go asyncLog(stderr)

	if err := cmd.Wait(); err != nil {
		if !strings.Contains(err.Error(), "exit status") {
			fmt.Println("wait:", err.Error())
		}
		return err
	}
	return nil
}

func ExecCommandWithResult(command string) string {
	out, err := exec.Command("bash", "-c", command).CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

func StartProcess(name string, args ...string) {
	env := os.Environ()
	procAttr := &os.ProcAttr{
		Env: env,
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
	}
	_, err := os.StartProcess(name, args, procAttr)
	if err != nil {
		fmt.Println(err)
	}
}

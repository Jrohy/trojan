package util

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"
)

func PortIsUse(port int) bool {
	if _, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), time.Millisecond*50); err == nil {
		return true
	} else {
		return false
	}
}

func RandomPort() int {
	for {
		rand.Seed(time.Now().UnixNano())
		newPort := rand.Intn(65536)
		if !PortIsUse(newPort) {
			return newPort
		}
	}
}

// IsExists 检测指定路径文件或者文件夹是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func GetLocalIP() string {
	resp, err := http.Get("http://api.ipify.org")
	if err != nil {
		resp, err = http.Get("http://icanhazip.com")
	}
	defer resp.Body.Close()
	s, err := ioutil.ReadAll(resp.Body)
	return string(s)
}

// CheckIP 检测ipv4地址的合法性
func CheckIP(ip string) bool {
	isOk, err := regexp.Match(`^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`, []byte(ip))
	if err != nil {
		fmt.Println(err)
	}
	return isOk
}

func InstallPack(name string) {
	if !CheckCommandExists(name) {
		if CheckCommandExists("yum") {
			ExecCommand("yum install -y " + name)
		} else if CheckCommandExists("apt-get") {
			ExecCommand("apt-get update")
			ExecCommand("apt-get install -y " + name)
		}
	}
}

func OpenPort(port int) {
	if CheckCommandExists("firewall-cmd") {
		ExecCommand(fmt.Sprintf("firewall-cmd --zone=public --add-port=%d/tcp --add-port=%d/udp --permanent >/dev/null 2>&1", port, port))
		ExecCommand("firewall-cmd --reload >/dev/null 2>&1")
	} else {
		if len(ExecCommandWithResult(fmt.Sprintf(`iptables -nvL --line-number|grep -w "%d"`, port))) > 0 {
			return
		}
		ExecCommand(fmt.Sprintf("iptables -I INPUT -p tcp --dport %d -j ACCEPT", port))
		ExecCommand(fmt.Sprintf("iptables -I INPUT -p udp --dport %d -j ACCEPT", port))
		ExecCommand(fmt.Sprintf("iptables -I OUTPUT -p udp --sport %d -j ACCEPT", port))
		ExecCommand(fmt.Sprintf("iptables -I OUTPUT -p tcp --sport %d -j ACCEPT", port))
	}
}

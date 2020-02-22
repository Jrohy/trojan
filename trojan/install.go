package trojan

import (
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"net"
	"strconv"
	"strings"
	"time"
	"trojan/core"
	"trojan/util"
)

var (
	dockerInstallUrl = "https://git.io/docker-install"
	mysqlDodkcerRun  = "docker run --name trojan-mysql --restart=always -p %d:3306 -v /home/mysql:/var/lib/mysql -e MYSQL_ROOT_PASSWORD=%s -e MYSQL_ROOT_HOST=%% -e MYSQL_DATABASE=trojan -d mysql/mysql-server:5.7"
)

func InstallMenu() {
	fmt.Println()
	menu := []string{"更新trojan", "证书申请", "安装mysql"}
	switch util.LoopInput("请选择: ", menu) {
	case 1:
		InstallTrojan()
	case 2:
		InstallTls()
	case 3:
		InstallMysql()
	}
}

func InstallDocker() {
	if !util.CheckCommandExists("docker") {
		util.RunWebShell(dockerInstallUrl)
		fmt.Println()
	}
}

func InstallTrojan() {
	box := packr.New("trojan-install", "../asset")
	data, err := box.FindString("trojan-install.sh")
	if err != nil {
		fmt.Println(err)
	}
	util.ExecCommand(data)
	util.ExecCommand("systemctl restart trojan")
	util.ExecCommand("systemctl enable trojan")
	fmt.Println()
}

func InstallTls() {
	domain := ""
	choice := util.LoopInput("请选择使用证书方式: ", []string{"Let's Encrypt 证书", "自定义证书路径"})
	if choice == 1 {
		localIP := util.GetLocalIP()
		fmt.Printf("本机ip: %s\n", localIP)
		for {
			domain = util.Input("请输入申请证书的域名: ", "")
			ipList, err := net.LookupIP(domain)
			fmt.Printf("%s 解析到的ip: %v\n", domain, ipList)
			if err != nil {
				fmt.Println(err)
				fmt.Println("域名有误,请重新输入")
				continue
			}
			checkIp := false
			for _, ip := range ipList {
				if localIP == ip.String() {
					checkIp = true
				}
			}
			if checkIp {
				break
			} else {
				fmt.Println("输入的域名和本机ip不一致, 请重新输入!")
			}
		}
		util.InstallPack("socat")
		if !util.IsExists("/root/.acme.sh/acme.sh") {
			util.RunWebShell("https://get.acme.sh")
		}
		util.OpenPort(80)
		util.ExecCommand(fmt.Sprintf("bash /root/.acme.sh/acme.sh --issue -d %s --debug --standalone --keylength ec-256", domain))
		crtFile := "/root/.acme.sh/" + domain + "_ecc" + "/fullchain.cer"
		keyFile := "/root/.acme.sh/" + domain + "_ecc" + "/" + domain + ".key"
		core.WriterTls(crtFile, keyFile)
	} else if choice == 2 {
		crtFile := util.Input("请输入证书的cert文件路径: ", "")
		keyFile := util.Input("请输入证书的key文件路径: ", "")
		if !util.IsExists(crtFile) || !util.IsExists(keyFile) {
			fmt.Println("输入的cert或者key文件不存在!")
		} else {
			domain = util.Input("请输入此证书对应的域名: ", "")
			if domain == "" {
				fmt.Println("输入域名为空!")
				return
			}
			core.WriterTls(crtFile, keyFile)
		}
	}
	core.SetValue("domain", domain)
	fmt.Println()
}

func InstallMysql() {
	var mysql core.Mysql
	choice := util.LoopInput("请选择: ", []string{"安装docker版mysql", "输入自定义mysql连接"})
	if choice < 0 {
		return
	} else if choice == 1 {
		mysql = core.Mysql{ServerAddr: "127.0.0.1", ServerPort: util.RandomPort(), Password: util.RandString(5), Username: "root", Database: "trojan"}
		InstallDocker()
		fmt.Println(fmt.Sprintf(mysqlDodkcerRun, mysql.ServerPort, mysql.Password))
		if util.CheckCommandExists("setenforce") {
			util.ExecCommand("setenforce 0")
		}
		util.ExecCommand(fmt.Sprintf(mysqlDodkcerRun, mysql.ServerPort, mysql.Password))
		fmt.Println("mysql启动中, 请稍等...")
		for {
			db := mysql.GetDB()
			err := db.Ping()
			if err == nil {
				db.Close()
				break
			} else {
				time.Sleep(2 * time.Second)
			}
		}
		fmt.Println("mysql启动成功!")
	} else if choice == 2 {
		mysql = core.Mysql{Username: "root"}
		for {
			for {
				mysqlUrl := util.Input("请输入mysql连接地址(格式: host:port), 默认连接地址为127.0.0.1:3306, 使用直接回车, 否则输入自定义连接地址: ",
					"127.0.0.1:3306")
				urlInfo := strings.Split(mysqlUrl, ":")
				if len(urlInfo) != 2 {
					fmt.Printf("输入的%s不符合匹配格式(host:port)", mysqlUrl)
					continue
				}
				port, err := strconv.Atoi(urlInfo[1])
				if err != nil {
					fmt.Printf("%s不是数字: ", urlInfo[1])
					continue
				}
				mysql.ServerAddr, mysql.ServerPort = urlInfo[0], port
				break
			}
			mysql.Password = util.Input("请输入mysql root用户的密码: ", "")
			db := mysql.GetDB()
			if db == nil {
				continue
			} else {
				db.Exec("CREATE DATABASE IF NOT EXISTS trojan;")
			}
			break
		}
	}
	mysql.Database = "trojan"
	mysql.CreateTable()
	core.WriterMysql(&mysql)
	AddUser()
	fmt.Println()
}

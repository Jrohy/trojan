package core

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
	"io/ioutil"
)

var configPath = "/usr/local/etc/trojan/config.json"

// ServerConfig 结构体
type ServerConfig struct {
	Config
	SSl   ServerSSL `json:"ssl"`
	Tcp   ServerTCP `json:"tcp"`
	Mysql Mysql     `json:"mysql"`
}

// ServerSSL 结构体
type ServerSSL struct {
	SSL
	Key                string `json:"key"`
	KeyPassword        string `json:"key_password"`
	PreferServerCipher bool   `json:"prefer_server_cipher"`
	SessionTimeout     int    `json:"session_timeout"`
	PlainHttpResponse  string `json:"plain_http_response"`
	Dhparam            string `json:"dhparam"`
}

// ServerTCP 结构体
type ServerTCP struct {
	TCP
	PreferIPv4 bool `json:"prefer_ipv4"`
}

// Load 加载服务端配置文件
func Load(path string) []byte {
	if path == "" {
		path = configPath
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return data
}

// Save 保存服务端配置文件
func Save(data []byte, path string) bool {
	if path == "" {
		path = configPath
	}
	if err := ioutil.WriteFile(path, pretty.Pretty(data), 0644); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// GetConfig 获取config配置
func GetConfig() *ServerConfig {
	data := Load("")
	config := ServerConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println(err)
		return nil
	}
	return &config
}

// GetMysql 获取mysql连接
func GetMysql() *Mysql {
	return &GetConfig().Mysql
}

// WriteMysql 写mysql配置
func WriteMysql(mysql *Mysql) bool {
	mysql.Enabled = true
	data := Load("")
	result, _ := sjson.SetBytes(data, "mysql", mysql)
	return Save(result, "")
}

// WriteTls 写tls配置
func WriteTls(cert, key, domain string) bool {
	data := Load("")
	data, _ = sjson.SetBytes(data, "ssl.cert", cert)
	data, _ = sjson.SetBytes(data, "ssl.key", key)
	data, _ = sjson.SetBytes(data, "ssl.sni", domain)
	return Save(data, "")
}

// WriteDomain 写域名
func WriteDomain(domain string) bool {
	data := Load("")
	data, _ = sjson.SetBytes(data, "ssl.sni", domain)
	return Save(data, "")
}

// WritePassword 写密码
func WritePassword(pass []string) bool {
	data := Load("")
	data, _ = sjson.SetBytes(data, "password", pass)
	return Save(data, "")
}

// WritePort 写trojan端口
func WritePort(port int) bool {
	data := Load("")
	data, _ = sjson.SetBytes(data, "local_port", port)
	return Save(data, "")
}

// WriteLogLevel 写日志等级
func WriteLogLevel(level int) bool {
	data := Load("")
	data, _ = sjson.SetBytes(data, "log_level", level)
	return Save(data, "")
}

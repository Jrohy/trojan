package core

import (
	"encoding/json"
	"fmt"
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
func Load(path string) *ServerConfig {
	if path == "" {
		path = configPath
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	config := ServerConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println(err)
		return nil
	}
	return &config
}

// Save 保存服务端配置文件
func Save(config *ServerConfig, path string) bool {
	if path == "" {
		path = configPath
	}
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println(err)
		return false
	}
	if err = ioutil.WriteFile(path, data, 0644); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// GetMysql 获取mysql连接
func GetMysql() *Mysql {
	config := Load("")
	return &config.Mysql
}

// WriterMysql 写mysql配置
func WriterMysql(mysql *Mysql) bool {
	mysql.Enabled = true
	mysql.Database = "trojan"
	config := Load("")
	config.Mysql = *mysql
	return Save(config, "")
}

// WriterTls 写tls配置
func WriterTls(cert string, key string) bool {
	config := Load("")
	config.SSl.Cert = cert
	config.SSl.Key = key
	return Save(config, "")
}

// WriterPassword 写密码
func WriterPassword(pass []string) bool {
	config := Load("")
	config.Password = pass
	return Save(config, "")
}

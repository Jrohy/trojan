package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var configPath = "/usr/local/etc/trojan/config.json"

type ServerConfig struct {
	Config
	SSl          ServerSSL       `json:"ssl"`
	Tcp          ServerTCP       `json:"tcp"`
	Mysql        Mysql           `json:"mysql"`
}

type ServerSSL struct {
	SSL
	Key                   string   `json:"key"`
	KeyPassword           string   `json:"key_password"`
	PreferServerCipher    bool     `json:"prefer_server_cipher"`
	SessionTimeout        int      `json:"session_timeout"`
	PlainHttpResponse     string   `json:"plain_http_response"`
	Dhparam               string   `json:"dhparam"`
}

type ServerTCP struct {
	TCP
	PreferIPv4        bool   `json:"prefer_ipv4"`
}

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

func GetMysql() *Mysql {
	config := Load("")
	return &config.Mysql
}

func WriterMysql(mysql *Mysql) bool {
	mysql.Enabled = true
	mysql.Database = "trojan"
	config := Load("")
	config.Mysql = *mysql
	return  Save(config, "")
}

func WriterTls(cert string, key string) bool {
	config := Load("")
	config.SSl.Cert = cert
	config.SSl.Key = key
	return  Save(config, "")
}
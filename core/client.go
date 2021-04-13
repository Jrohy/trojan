package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"trojan/asset"
)

// ClientConfig 结构体
type ClientConfig struct {
	Config
	SSl ClientSSL `json:"ssl"`
	Tcp ClientTCP `json:"tcp"`
}

// ClientSSL 结构体
type ClientSSL struct {
	SSL
	Verify         bool `json:"verify"`
	VerifyHostname bool `json:"verify_hostname"`
}

// ClientTCP 结构体
type ClientTCP struct {
	TCP
}

// WriteClient 生成客户端json
func WriteClient(port int, password, domain, writePath string) bool {
	data := asset.GetAsset("client.json")
	config := ClientConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println(err)
		return false
	}
	config.RemoteAddr = domain
	config.RemotePort = port
	config.Password = []string{password}
	outData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println(err)
		return false
	}
	if err = ioutil.WriteFile(writePath, outData, 0644); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

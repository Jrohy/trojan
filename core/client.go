package core

import (
	"encoding/json"
	"fmt"
	"os"
	"trojan/asset"
)

// ClientConfig 结构体
type ClientConfig struct {
	Config
	SSl       ClientSSL `json:"ssl"`
	Tcp       ClientTCP `json:"tcp"`
	Mux       Mux       `json:"mux"`
	Websocket Websocket `json:"websocket,omitempty"`
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
	config.SSl.Sni = domain
	if len(config.SSl.Alpn) == 0 {
		config.SSl.Alpn = []string{"h2", "http/1.1"}
	}
	if config.Mux.Concurrency == 0 {
		config.Mux = Mux{
			Enabled:     false,
			Concurrency: 8,
			IdleTimeout: 60,
		}
	}
	outData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println(err)
		return false
	}
	if err = os.WriteFile(writePath, outData, 0644); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

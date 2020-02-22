package core

import (
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"io/ioutil"
)

type ClientConfig struct {
	Config
	SSl          ClientSSL       `json:"ssl"`
	Tcp          ClientTCP       `json:"tcp"`
}

type ClientSSL struct {
	SSL
	Verify                bool     `json:"verify"`
	VerifyHostname        bool     `json:"verify_hostname"`
	Sni                   string   `json:"sni"`
}

type ClientTCP struct {
	TCP
}

func WriteClient(password string, ip string, domain string, writePath string) bool {
	box := packr.New("client.json", "../asset")
	data, err := box.Find("client.json")
	if err != nil {
		fmt.Println(err)
		return false
	}
	config := ClientConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println(err)
		return false
	}
	config.RemoteAddr = ip
	config.Password = []string{password}
	config.SSl.Sni = domain
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
package asset

import "embed"

//go:embed trojan-install.sh client.json clash-rules.yaml
var f embed.FS

// GetAsset 获取资源[]byte
func GetAsset(name string) []byte {
	data, _ := f.ReadFile(name)
	return data
}

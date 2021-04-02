package asset

import "embed"

//go:embed trojan-install.sh client.json
var f embed.FS

// GetAsset 获取资源字符串
func GetAsset(name string) []byte {
	data, _ := f.ReadFile(name)
	return data
}

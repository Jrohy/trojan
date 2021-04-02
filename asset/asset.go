package asset

import _ "embed"

//go:embed trojan-install.sh
var installF string

//go:embed client.json
var clientF string

// GetAssetStr 获取资源字符串
func GetAssetStr(ftype int) string {
	if ftype == 0 {
		return installF
	} else if ftype == 1 {
		return clientF
	}
	return ""
}

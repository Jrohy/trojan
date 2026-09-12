package controller

import (
	"os"
	"strconv"
	"strings"
	"time"
	"trojan/asset"
	"trojan/core"
	"trojan/trojan"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

// ResponseBody 结构体
type ResponseBody struct {
	Duration string
	Data     interface{}
	Msg      string
}

// TimeCost web函数执行用时统计方法
func TimeCost(start time.Time, body *ResponseBody) {
	body.Duration = time.Since(start).String()
}

func clashRules() string {
	rules, _ := core.GetValue("clash-rules")
	if rules == "" {
		rules = string(asset.GetAsset("clash-rules.yaml"))
	}
	return rules
}

// Version 获取版本信息
func Version() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	responseBody.Data = map[string]string{
		"version":       trojan.MVersion,
		"buildDate":     trojan.BuildDate,
		"goVersion":     trojan.GoVersion,
		"gitVersion":    trojan.GitVersion,
		"trojanVersion": trojan.Version(),
		"trojanUptime":  trojan.UpTime(),
		"trojanType":    trojan.Type(),
		"trojanState":   trojan.ActiveState(),
	}
	return &responseBody
}

// SetLoginInfo 设置登录页信息
func SetLoginInfo(title string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	err := core.SetValue("login_title", title)
	if err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// SetDomain 设置域名
func SetDomain(domain string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.SetDomain(domain)
	return &responseBody
}

// SetClashRules 设置clash规则
func SetClashRules(rules string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	core.SetValue("clash-rules", rules)
	return &responseBody
}

// ResetClashRules 重置clash规则
func ResetClashRules() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	core.DelValue("clash-rules")
	responseBody.Data = clashRules()
	return &responseBody
}

// GetClashRules 获取clash规则
func GetClashRules() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	responseBody.Data = clashRules()
	return &responseBody
}

// SetTrojanType 设置trojan类型
func SetTrojanType(tType string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	err := trojan.SwitchType(tType)
	if err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func readSockstat() map[string]int {
	result := map[string]int{"tcp": 0, "udp": 0, "timeWait": 0}
	data, err := os.ReadFile("/proc/net/sockstat")
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		protocol := strings.TrimSuffix(fields[0], ":")
		values := map[string]int{}
		for index := 1; index+1 < len(fields); index += 2 {
			values[fields[index]], _ = strconv.Atoi(fields[index+1])
		}
		switch protocol {
		case "TCP":
			result["tcp"] = values["inuse"]
			result["timeWait"] = values["tw"]
		case "UDP":
			result["udp"] = values["inuse"]
		}
	}
	return result
}

// ServerInfo 获取服务器信息
func ServerInfo() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	cpuPercent, _ := cpu.Percent(0, false)
	vmInfo, _ := mem.VirtualMemory()
	smInfo, _ := mem.SwapMemory()
	diskInfo, _ := disk.Usage("/")
	loadInfo, _ := load.Avg()
	netCount := readSockstat()
	network := NetworkTrafficSnapshot()
	responseBody.Data = map[string]interface{}{
		"cpu":    cpuPercent,
		"memory": vmInfo,
		"swap":   smInfo,
		"disk":   diskInfo,
		"load":   loadInfo,
		"speed": map[string]uint64{
			"Up":   network.UploadSpeed,
			"Down": network.DownloadSpeed,
		},
		"network":  network,
		"netCount": netCount,
		"conntrack": map[string]int{
			"count": readIntFile("/proc/sys/net/netfilter/nf_conntrack_count"),
			"limit": readIntFile("/proc/sys/net/netfilter/nf_conntrack_max"),
		},
	}
	return &responseBody
}

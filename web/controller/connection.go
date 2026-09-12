package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"trojan/core"
	"trojan/trojan"

	gnet "github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

type apiUser struct {
	Password string `json:"password"`
	Hash     string `json:"hash"`
}

type apiTraffic struct {
	UploadTraffic   uint64 `json:"upload_traffic"`
	DownloadTraffic uint64 `json:"download_traffic"`
}

type apiSpeed struct {
	UploadSpeed   uint64 `json:"upload_speed"`
	DownloadSpeed uint64 `json:"download_speed"`
}

type apiStatus struct {
	User         apiUser    `json:"user"`
	TrafficTotal apiTraffic `json:"traffic_total"`
	SpeedCurrent apiSpeed   `json:"speed_current"`
	SpeedLimit   apiSpeed   `json:"speed_limit"`
	IPCurrent    int        `json:"ip_current"`
	IPLimit      int        `json:"ip_limit"`
}

type apiListItem struct {
	User   apiUser   `json:"user"`
	Status apiStatus `json:"status"`
}

type userConnectionMetric struct {
	ID              uint   `json:"id"`
	Username        string `json:"username"`
	Hash            string `json:"hash"`
	Online          bool   `json:"online"`
	IPCurrent       int    `json:"ipCurrent"`
	IPLimit         int    `json:"ipLimit"`
	UploadSpeed     uint64 `json:"uploadSpeed"`
	DownloadSpeed   uint64 `json:"downloadSpeed"`
	UploadTraffic   uint64 `json:"uploadTraffic"`
	DownloadTraffic uint64 `json:"downloadTraffic"`
	TCPConnections  *int   `json:"tcpConnections"`
	FDCount         *int   `json:"fdCount"`
}

type trojanProcessMetric struct {
	PID                 int32  `json:"pid"`
	TCPConnections      int    `json:"tcpConnections"`
	FDCount             int    `json:"fdCount"`
	FDLimit             int    `json:"fdLimit"`
	TCPLimit            int    `json:"tcpLimit"`
	TCPLimitSource      string `json:"tcpLimitSource"`
	SYNBacklogLimit     int    `json:"synBacklogLimit"`
	TimeWaitBucketLimit int    `json:"timeWaitBucketLimit"`
	ConntrackLimit      int    `json:"conntrackLimit"`
}

func configuredAPIAddr() string {
	config := core.GetConfig()
	if config == nil || !config.API.Enabled {
		return ""
	}
	host := config.API.APIAddr
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.API.APIPort
	if port == 0 {
		port = 10000
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func trimJSONList(out string) string {
	start := strings.Index(out, "[")
	end := strings.LastIndex(out, "]")
	if start < 0 || end < start {
		return ""
	}
	return out[start : end+1]
}

func listAPIUsers(apiAddr string) (map[string]apiStatus, error) {
	if apiAddr == "" {
		return nil, errors.New("trojan-go api is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/trojan/trojan", "-api-addr", apiAddr, "-api", "list")
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errors.New("trojan-go api request timeout")
	}
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	payload := trimJSONList(string(out))
	if payload == "" {
		return nil, errors.New("trojan-go api returned invalid json")
	}
	var items []apiListItem
	if err := json.Unmarshal([]byte(payload), &items); err != nil {
		return nil, err
	}
	statusByHash := make(map[string]apiStatus, len(items))
	for _, item := range items {
		status := item.Status
		hash := item.User.Hash
		if hash == "" {
			hash = status.User.Hash
		}
		if hash != "" {
			statusByHash[hash] = status
		}
	}
	return statusByHash, nil
}

func processFDLimit(pid int32) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/limits", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "Max open files") {
			continue
		}
		fields := strings.Fields(line)
		for _, field := range fields {
			if value, err := strconv.Atoi(field); err == nil {
				return value
			}
		}
	}
	return 0
}

func readIntFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return value
}

func tcpLimit(fdLimit int) (int, string) {
	fdBasedLimit := 0
	if fdLimit > 0 {
		fdBasedLimit = fdLimit / 2
	}
	limits := []struct {
		value  int
		source string
	}{
		{fdBasedLimit, "process fd soft limit / 2"},
		{readIntFile("/proc/sys/net/netfilter/nf_conntrack_max"), "nf_conntrack_max"},
	}
	result := 0
	source := ""
	for _, limit := range limits {
		if limit.value <= 0 {
			continue
		}
		if result == 0 || limit.value < result {
			result = limit.value
			source = limit.source
		}
	}
	return result, source
}

func trojanProcessStats() []trojanProcessMetric {
	config := core.GetConfig()
	if config == nil {
		return nil
	}
	connections, _ := gnet.Connections("tcp")
	tcpByPID := map[int32]int{}
	pidSet := map[int32]struct{}{}
	for _, conn := range connections {
		if conn.Pid <= 0 {
			continue
		}
		if int(conn.Laddr.Port) == config.LocalPort || int(conn.Raddr.Port) == config.LocalPort {
			pidSet[conn.Pid] = struct{}{}
			tcpByPID[conn.Pid]++
		}
	}
	result := make([]trojanProcessMetric, 0, len(pidSet))
	for pid := range pidSet {
		fdCount := int32(0)
		if proc, err := process.NewProcess(pid); err == nil {
			if name, err := proc.Name(); err == nil && !strings.Contains(name, "trojan") {
				continue
			}
			fdCount, _ = proc.NumFDs()
		}
		fdLimit := processFDLimit(pid)
		tcpLimitValue, tcpLimitSource := tcpLimit(fdLimit)
		result = append(result, trojanProcessMetric{
			PID:                 pid,
			TCPConnections:      tcpByPID[pid],
			FDCount:             int(fdCount),
			FDLimit:             fdLimit,
			TCPLimit:            tcpLimitValue,
			TCPLimitSource:      tcpLimitSource,
			SYNBacklogLimit:     readIntFile("/proc/sys/net/ipv4/tcp_max_syn_backlog"),
			TimeWaitBucketLimit: readIntFile("/proc/sys/net/ipv4/tcp_max_tw_buckets"),
			ConntrackLimit:      readIntFile("/proc/sys/net/netfilter/nf_conntrack_max"),
		})
	}
	return result
}

// ActiveUsers 获取当前在线用户。进程级扫描由系统监控独立按低频调用。
func ActiveUsers() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	mysql := core.GetMysql()
	userList, err := mysql.GetData()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}

	apiAddr := configuredAPIAddr()
	statusByHash, apiErr := listAPIUsers(apiAddr)
	metrics := make([]userConnectionMetric, 0, len(userList))
	onlineMetrics := make([]userConnectionMetric, 0)
	for _, user := range userList {
		if isOpsProbeHash(user.EncryptPass) {
			continue
		}
		status := statusByHash[user.EncryptPass]
		metric := userConnectionMetric{
			ID:              user.ID,
			Username:        user.Username,
			Hash:            user.EncryptPass,
			Online:          status.IPCurrent > 0 || status.SpeedCurrent.UploadSpeed > 0 || status.SpeedCurrent.DownloadSpeed > 0,
			IPCurrent:       status.IPCurrent,
			IPLimit:         status.IPLimit,
			UploadSpeed:     status.SpeedCurrent.UploadSpeed,
			DownloadSpeed:   status.SpeedCurrent.DownloadSpeed,
			UploadTraffic:   status.TrafficTotal.UploadTraffic,
			DownloadTraffic: status.TrafficTotal.DownloadTraffic,
		}
		if metric.Online {
			onlineMetrics = append(onlineMetrics, metric)
		}
		metrics = append(metrics, metric)
	}

	apiError := ""
	if apiErr != nil {
		apiError = apiErr.Error()
	}
	responseBody.Data = map[string]interface{}{
		"apiEnabled":  apiAddr != "",
		"apiAddr":     apiAddr,
		"apiError":    apiError,
		"users":       metrics,
		"onlineUsers": onlineMetrics,
		"note":        "Trojan-Go API exposes per-user online IP count, speed and traffic.",
	}
	return &responseBody
}

// ProcessMetrics returns the heavier process-level TCP/FD scan for System Monitor only.
func ProcessMetrics() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	responseBody.Data = trojanProcessStats()
	return &responseBody
}

// H2Profile 获取H2链路配置状态
func H2Profile() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	config := core.GetConfig()
	if config == nil {
		responseBody.Msg = "load config failed"
		return &responseBody
	}
	hasH2 := false
	for _, alpn := range config.SSl.Alpn {
		if alpn == "h2" {
			hasH2 = true
			break
		}
	}
	h2Port := config.SSl.AlpnPortOverride["h2"]
	h2BackendReady := false
	h2BackendError := ""
	if h2Port > 0 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", h2Port), 500*time.Millisecond)
		if err == nil {
			h2BackendReady = true
			_ = conn.Close()
		} else {
			h2BackendError = err.Error()
		}
	}
	responseBody.Data = map[string]interface{}{
		"trojanType":       trojan.Type(),
		"trojanState":      trojan.ActiveState(),
		"trojanRunning":    trojan.IsRunning(),
		"trojanUptime":     trojan.UpTime(),
		"h2Alpn":           hasH2,
		"h2Port":           h2Port,
		"h2BackendReady":   h2BackendReady,
		"h2BackendError":   h2BackendError,
		"h2FullyEnabled":   hasH2 && h2Port > 0 && h2BackendReady,
		"alpn":             config.SSl.Alpn,
		"alpnPortOverride": config.SSl.AlpnPortOverride,
		"mux":              config.Mux,
		"api":              config.API,
		"tcp":              config.Tcp,
	}
	return &responseBody
}

// ApplyH2Profile 应用推荐H2链路配置
func ApplyH2Profile() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if trojan.Type() != "trojan-go" {
		responseBody.Msg = "H2 profile requires trojan-go"
		return &responseBody
	}
	if !core.WriteH2Profile() {
		responseBody.Msg = "write h2 profile failed"
		return &responseBody
	}
	trojan.Restart()
	return &responseBody
}

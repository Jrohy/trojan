package controller

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"trojan/core"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"golang.org/x/net/websocket"
)

const (
	opsSchemaVersion      = 1
	opsSampleInterval     = time.Minute
	opsRetention          = 30 * 24 * time.Hour
	opsMaxStorageBytes    = int64(64 * 1024 * 1024)
	opsMaxInMemorySamples = 30*24*60 + 1
	opsOverviewCacheTTL   = 30 * time.Second
	opsSyntheticDeadline  = 5 * time.Second
	opsSyntheticBodyLimit = int64(16 * 1024)
	opsMinimumDiskFree    = uint64(512 * 1024 * 1024)
	opsMinimumMemoryFree  = uint64(128 * 1024 * 1024)
	opsMaximumLoadPerCPU  = 1.0
	opsSLOTarget          = 0.999
	opsDefaultStorageDir  = "/var/lib/trojan-web/ops"
	opsStorageDirEnv      = "TROJAN_OPS_DATA_DIR"
	opsEnabledEnv         = "TROJAN_OPS_ENABLED"
	opsCredentialHashEnv  = "TROJAN_OPS_CREDENTIAL_HASH"
)

type opsTargetResult struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Scope         string   `json:"scope"`
	Status        string   `json:"status"`
	LatencyMS     *float64 `json:"latencyMs"`
	HTTPStatus    *int     `json:"httpStatus"`
	Error         string   `json:"error,omitempty"`
	LastCheckedAt *string  `json:"lastCheckedAt"`
}

type opsResourceSnapshot struct {
	SampledAt       *string  `json:"sampledAt"`
	CPUPercent      *float64 `json:"cpuPercent"`
	MemoryPercent   *float64 `json:"memoryPercent"`
	SwapPercent     *float64 `json:"swapPercent"`
	DiskPercent     *float64 `json:"diskPercent"`
	DiskFreeBytes   *uint64  `json:"diskFreeBytes"`
	Load1           *float64 `json:"load1"`
	NetworkUpBPS    *uint64  `json:"networkUpBps"`
	NetworkDownBPS  *uint64  `json:"networkDownBps"`
	ServiceState    string   `json:"serviceState"`
	PressureSkipped bool     `json:"pressureSkipped"`
	Error           string   `json:"error,omitempty"`
}

type opsSample struct {
	At            time.Time           `json:"at"`
	TrackingSince time.Time           `json:"trackingSince,omitempty"`
	Status        string              `json:"status"`
	Resources     opsResourceSnapshot `json:"resources"`
	Targets       []opsTargetResult   `json:"targets"`
}

type opsProbeConfig struct {
	domain        string
	port          int
	websocketPath string
}

type opsOverviewSummary struct {
	Success              int      `json:"success"`
	Failed               int      `json:"failed"`
	Unknown              int      `json:"unknown"`
	Expected             int      `json:"expected"`
	Coverage             float64  `json:"coverage"`
	Availability         *float64 `json:"availability"`
	P50MS                *float64 `json:"p50Ms"`
	P95MS                *float64 `json:"p95Ms"`
	P99MS                *float64 `json:"p99Ms"`
	ErrorBudgetRemaining *float64 `json:"errorBudgetRemaining"`
	SLOTarget            float64  `json:"sloTarget"`
}

type opsFreshness struct {
	LastSampleAt          *string `json:"lastSampleAt"`
	NextExpectedAt        *string `json:"nextExpectedAt"`
	Stale                 bool    `json:"stale"`
	SampleIntervalSeconds int     `json:"sampleIntervalSeconds"`
}

type opsHistoryPoint struct {
	At             string   `json:"at"`
	Status         string   `json:"status"`
	Success        int      `json:"success"`
	Failed         int      `json:"failed"`
	Unknown        int      `json:"unknown"`
	Availability   *float64 `json:"availability"`
	P95MS          *float64 `json:"p95Ms"`
	CPUPercent     *float64 `json:"cpuPercent"`
	MemoryPercent  *float64 `json:"memoryPercent"`
	DiskPercent    *float64 `json:"diskPercent"`
	NetworkUpBPS   *uint64  `json:"networkUpBps"`
	NetworkDownBPS *uint64  `json:"networkDownBps"`
}

type opsLatencyBucket struct {
	LEMS  *int `json:"leMs"`
	Count int  `json:"count"`
}

type opsEvent struct {
	At       string `json:"at"`
	Severity string `json:"severity"`
	Target   string `json:"target"`
	Message  string `json:"message"`
}

type opsSource struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Scope     string `json:"scope"`
	Available bool   `json:"available"`
	Message   string `json:"message,omitempty"`
}

type opsOverviewData struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	UpdatedAt      string              `json:"updatedAt"`
	TrackingSince  string              `json:"trackingSince"`
	Range          string              `json:"range"`
	Scope          string              `json:"scope"`
	Status         string              `json:"status"`
	Overview       opsOverviewSummary  `json:"overview"`
	Freshness      opsFreshness        `json:"freshness"`
	Resources      opsResourceSnapshot `json:"resources"`
	Targets        []opsTargetResult   `json:"targets"`
	History        []opsHistoryPoint   `json:"history"`
	LatencyBuckets []opsLatencyBucket  `json:"latencyBuckets"`
	Events         []opsEvent          `json:"events"`
	Sources        []opsSource         `json:"sources"`
}

var opsMonitor = struct {
	sync.RWMutex
	started           bool
	trackingSince     time.Time
	storageLoadError  bool
	storageWriteError bool
	storageDropped    int
	samples           []opsSample
	caches            map[string]opsOverviewCacheEntry
	probe             func(context.Context, opsProbeConfig, string, string) opsTargetResult
	resource          func(context.Context) opsResourceSnapshot
}{
	caches:   make(map[string]opsOverviewCacheEntry),
	probe:    probeTrojanTarget,
	resource: collectOpsResources,
}

type opsOverviewCacheEntry struct {
	data      opsOverviewData
	expiresAt time.Time
}

var opsOverviewBuild sync.Mutex

var opsCPUState = struct {
	sync.Mutex
	primed bool
}{}

var opsTargets = []struct {
	id   string
	name string
	host string
}{
	{id: "google", name: "Google", host: "www.google.com"},
	{id: "youtube", name: "YouTube", host: "www.youtube.com"},
}

func opsFloatPointer(value float64) *float64 { return &value }
func opsIntPointer(value int) *int           { return &value }
func opsUint64Pointer(value uint64) *uint64  { return &value }
func opsStringPointer(value string) *string  { return &value }

func opsStorageDir() string {
	if value := strings.TrimSpace(os.Getenv(opsStorageDirEnv)); value != "" {
		return value
	}
	return opsDefaultStorageDir
}

func opsEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(opsEnabledEnv)))
	return value != "0" && value != "false" && value != "off"
}

// StartOpsMonitor starts one shared minute-level sampler. Requests only read its cache.
func StartOpsMonitor() {
	opsMonitor.Lock()
	if opsMonitor.started || !opsEnabled() {
		opsMonitor.Unlock()
		return
	}
	opsMonitor.started = true
	loadedSamples, trackingHint, dropped, loadErr := loadOpsSamples(time.Now())
	opsMonitor.samples = loadedSamples
	opsMonitor.storageLoadError = loadErr != nil || dropped > 0
	opsMonitor.storageDropped = dropped
	if len(opsMonitor.samples) > 0 {
		opsMonitor.trackingSince = opsMonitor.samples[0].TrackingSince
		if opsMonitor.trackingSince.IsZero() {
			opsMonitor.trackingSince = opsMonitor.samples[0].At
		}
	} else {
		opsMonitor.trackingSince = trackingHint
		if opsMonitor.trackingSince.IsZero() {
			opsMonitor.trackingSince = time.Now().UTC()
		}
	}
	opsMonitor.Unlock()

	go func() {
		ticker := time.NewTicker(opsSampleInterval)
		defer ticker.Stop()
		collectAndStoreOpsSample(time.Now())
		for now := range ticker.C {
			collectAndStoreOpsSample(now)
		}
	}()
}

func collectAndStoreOpsSample(now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resources := opsMonitor.resource(ctx)
	cancel()

	opsMonitor.RLock()
	trackingSince := opsMonitor.trackingSince
	opsMonitor.RUnlock()
	sample := opsSample{At: now.UTC(), TrackingSince: trackingSince, Resources: resources}
	if resources.PressureSkipped {
		for _, target := range opsTargets {
			sample.Targets = append(sample.Targets, unknownOpsTarget(target.id, target.name, target.host, "服务器资源压力较高，本轮主动跳过"))
		}
	} else {
		probeConfig, credentialHash, err := opsProbeCredential()
		if err != nil {
			for _, target := range opsTargets {
				sample.Targets = append(sample.Targets, unknownOpsTarget(target.id, target.name, target.host, err.Error()))
			}
		} else {
			for _, target := range opsTargets {
				ctx, cancel := context.WithTimeout(context.Background(), opsSyntheticDeadline)
				result := opsMonitor.probe(ctx, probeConfig, credentialHash, target.host)
				cancel()
				result.ID, result.Name, result.Host = target.id, target.name, target.host
				result.Scope = "server-side-synthetic"
				sample.Targets = append(sample.Targets, result)
			}
		}
	}
	sample.Status = deriveOpsStatus(resources, sample.Targets)

	opsMonitor.Lock()
	opsMonitor.samples = append(opsMonitor.samples, sample)
	trimOpsSamplesLocked(now)
	opsMonitor.caches = make(map[string]opsOverviewCacheEntry)
	opsMonitor.Unlock()
	storageErr := appendOpsSample(sample)
	opsMonitor.Lock()
	opsMonitor.storageWriteError = storageErr != nil
	opsMonitor.caches = make(map[string]opsOverviewCacheEntry)
	opsMonitor.Unlock()
}

func unknownOpsTarget(id, name, host, message string) opsTargetResult {
	return opsTargetResult{ID: id, Name: name, Host: host, Scope: "server-side-synthetic", Status: "unknown", Error: message}
}

func opsProbeCredential() (opsProbeConfig, string, error) {
	config := core.GetConfig()
	if config == nil {
		return opsProbeConfig{}, "", errors.New("Trojan 配置不可用，无法执行合成探测")
	}
	domain := strings.TrimSpace(config.SSl.Sni)
	if !validOpsHost(domain) || config.LocalPort <= 0 || config.LocalPort > 65535 {
		return opsProbeConfig{}, "", errors.New("Trojan 域名或端口未配置")
	}
	probeConfig := opsProbeConfig{domain: domain, port: config.LocalPort}
	if config.Websocket.Enabled {
		path, err := normalizeOpsWebsocketPath(config.Websocket.Path)
		if err != nil {
			return opsProbeConfig{}, "", err
		}
		configuredHost := strings.TrimSpace(config.Websocket.Host)
		if configuredHost != "" && (!validOpsHost(configuredHost) || !strings.EqualFold(configuredHost, domain)) {
			return opsProbeConfig{}, "", errors.New("WebSocket Host 与 Trojan TLS 域名不一致")
		}
		probeConfig.websocketPath = path
	}
	if credentialHash := strings.TrimSpace(os.Getenv(opsCredentialHashEnv)); credentialHash != "" {
		if len(credentialHash) != 56 {
			return opsProbeConfig{}, "", errors.New("专用探测凭据哈希格式无效")
		}
		if _, err := hex.DecodeString(credentialHash); err != nil {
			return opsProbeConfig{}, "", errors.New("专用探测凭据哈希格式无效")
		}
		return probeConfig, strings.ToLower(credentialHash), nil
	}
	return opsProbeConfig{}, "", errors.New("未配置专用合成探测凭据")
}

func validOpsHost(host string) bool {
	return host != "" && !strings.ContainsAny(host, " /\\@?#")
}

func normalizeOpsWebsocketPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "", errors.New("WebSocket 探测路径配置无效")
	}
	parsed, err := url.ParseRequestURI(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" {
		return "", errors.New("WebSocket 探测路径配置无效")
	}
	return parsed.RequestURI(), nil
}

func collectOpsResources(ctx context.Context) opsResourceSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	result := opsResourceSnapshot{SampledAt: &now, ServiceState: boundedServiceState(ctx)}
	var failures []string
	opsCPUState.Lock()
	values, cpuErr := cpu.Percent(0, false)
	cpuPrimed := opsCPUState.primed
	opsCPUState.primed = true
	opsCPUState.Unlock()
	if cpuErr != nil || len(values) == 0 {
		failures = append(failures, "CPU 指标不可用")
	} else if !cpuPrimed {
		failures = append(failures, "CPU 指标正在建立采样基线")
	} else {
		result.CPUPercent = opsFloatPointer(values[0])
	}
	vm, vmErr := mem.VirtualMemory()
	if vmErr != nil {
		failures = append(failures, "内存指标不可用")
	} else {
		result.MemoryPercent = opsFloatPointer(vm.UsedPercent)
	}
	if swapInfo, err := mem.SwapMemory(); err != nil {
		failures = append(failures, "Swap 指标不可用")
	} else {
		result.SwapPercent = opsFloatPointer(swapInfo.UsedPercent)
	}
	diskInfo, diskErr := disk.Usage("/")
	if diskErr != nil {
		failures = append(failures, "磁盘指标不可用")
	} else {
		result.DiskPercent = opsFloatPointer(diskInfo.UsedPercent)
		result.DiskFreeBytes = opsUint64Pointer(diskInfo.Free)
	}
	loadInfo, loadErr := load.Avg()
	if loadErr != nil {
		failures = append(failures, "负载指标不可用")
	} else {
		result.Load1 = opsFloatPointer(loadInfo.Load1)
	}
	network := NetworkTrafficSnapshot()
	if network.Ready && network.Error == "" {
		result.NetworkUpBPS = opsUint64Pointer(network.UploadSpeed)
		result.NetworkDownBPS = opsUint64Pointer(network.DownloadSpeed)
	} else if network.Error != "" {
		failures = append(failures, "网络指标不可用")
	}
	result.PressureSkipped = opsUnderPressure(result, vm, vmErr, diskInfo, diskErr)
	result.Error = strings.Join(failures, "；")
	return result
}

func boundedServiceState(parent context.Context) string {
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "systemctl", "is-active", "trojan.service").Output()
	state := strings.TrimSpace(string(output))
	if err != nil && state == "" {
		return "unknown"
	}
	if state == "" {
		return "unknown"
	}
	return state
}

func opsUnderPressure(resources opsResourceSnapshot, vm *mem.VirtualMemoryStat, vmErr error, diskInfo *disk.UsageStat, diskErr error) bool {
	if vmErr == nil && vm.Available < opsMinimumMemoryFree {
		return true
	}
	if diskErr == nil && diskInfo.Free < opsMinimumDiskFree {
		return true
	}
	if resources.CPUPercent != nil && resources.Load1 != nil {
		cpuCount, err := cpu.Counts(true)
		if err == nil && cpuCount > 0 && *resources.CPUPercent > 80 && *resources.Load1/float64(cpuCount) > opsMaximumLoadPerCPU {
			return true
		}
	}
	return false
}

func probeTrojanTarget(ctx context.Context, probeConfig opsProbeConfig, credentialHash, target string) opsTargetResult {
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	result := opsTargetResult{Scope: "server-side-synthetic", Status: "failed", LastCheckedAt: &checkedAt}
	started := time.Now()
	dialer := &net.Dialer{}
	raw, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(probeConfig.port)))
	if err != nil {
		result.Error = "Trojan 端口连接失败: " + compactOpsError(err)
		return result
	}
	defer raw.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = raw.SetDeadline(deadline)
	}
	outer := tls.Client(raw, &tls.Config{ServerName: probeConfig.domain, MinVersion: tls.VersionTLS12})
	if err := outer.HandshakeContext(ctx); err != nil {
		result.Error = "Trojan TLS 校验失败: " + compactOpsError(err)
		return result
	}
	tunnel, err := openOpsTrojanTransport(outer, probeConfig)
	if err != nil {
		result.Error = "Trojan WebSocket 握手失败: " + compactOpsError(err)
		return result
	}
	request, err := trojanConnectRequest(credentialHash, target, 443)
	if err != nil {
		result.Error = compactOpsError(err)
		return result
	}
	if _, err := tunnel.Write(request); err != nil {
		result.Error = "Trojan CONNECT 写入失败: " + compactOpsError(err)
		return result
	}
	inner := tls.Client(tunnel, &tls.Config{ServerName: target, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}})
	if err := inner.HandshakeContext(ctx); err != nil {
		result.Error = "Trojan 转发/认证或目标 TLS 阶段失败: " + compactOpsError(err)
		return result
	}
	requestText := "GET /generate_204 HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: trojan-web-sla/1\r\nAccept: */*\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(inner, requestText); err != nil {
		result.Error = "HTTP 探测写入失败: " + compactOpsError(err)
		return result
	}
	response, err := http.ReadResponse(bufio.NewReader(io.LimitReader(inner, opsSyntheticBodyLimit)), &http.Request{Method: http.MethodGet})
	if err != nil {
		result.Error = "HTTP 探测响应无效: " + compactOpsError(err)
		return result
	}
	defer response.Body.Close()
	result.HTTPStatus = opsIntPointer(response.StatusCode)
	latency := float64(time.Since(started).Microseconds()) / 1000
	result.LatencyMS = &latency
	if response.StatusCode != http.StatusNoContent {
		result.Error = fmt.Sprintf("期望 HTTP 204，实际为 %d", response.StatusCode)
		return result
	}
	result.Status = "healthy"
	return result
}

func openOpsTrojanTransport(outer *tls.Conn, probeConfig opsProbeConfig) (net.Conn, error) {
	if probeConfig.websocketPath == "" {
		return outer, nil
	}
	location := url.URL{Scheme: "wss", Host: probeConfig.domain}
	path, err := url.ParseRequestURI(probeConfig.websocketPath)
	if err != nil {
		return nil, errors.New("WebSocket 探测路径配置无效")
	}
	location.Path = path.Path
	location.RawPath = path.RawPath
	location.RawQuery = path.RawQuery
	origin := url.URL{Scheme: "https", Host: probeConfig.domain}
	websocketConfig, err := websocket.NewConfig(location.String(), origin.String())
	if err != nil {
		return nil, err
	}
	connection, err := websocket.NewClient(websocketConfig, outer)
	if err != nil {
		return nil, err
	}
	connection.PayloadType = websocket.BinaryFrame
	return connection, nil
}

func trojanConnectRequest(credentialHash, target string, port uint16) ([]byte, error) {
	if len(target) == 0 || len(target) > 255 {
		return nil, errors.New("探测目标域名长度无效")
	}
	if len(credentialHash) != 56 {
		return nil, errors.New("Trojan 探测凭据哈希无效")
	}
	if _, err := hex.DecodeString(credentialHash); err != nil {
		return nil, errors.New("Trojan 探测凭据哈希无效")
	}
	buffer := make([]byte, 0, 56+2+1+1+1+len(target)+2+2)
	buffer = append(buffer, credentialHash...)
	buffer = append(buffer, '\r', '\n', 0x01, 0x03, byte(len(target)))
	buffer = append(buffer, target...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	buffer = append(buffer, portBytes...)
	buffer = append(buffer, '\r', '\n')
	return buffer, nil
}

func compactOpsError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return "操作超时"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "certificate") || strings.Contains(message, "x509"):
		return "证书校验失败"
	case strings.Contains(message, "connection refused"):
		return "连接被拒绝"
	case strings.Contains(message, "connection reset") || strings.Contains(message, "broken pipe"):
		return "连接被远端重置"
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		return "连接提前关闭"
	default:
		return "协议或网络错误"
	}
}

func deriveOpsStatus(resources opsResourceSnapshot, targets []opsTargetResult) string {
	if resources.ServiceState != "active" && resources.ServiceState != "unknown" {
		return "critical"
	}
	healthy, failed := 0, 0
	for _, target := range targets {
		switch target.Status {
		case "healthy":
			healthy++
		case "failed":
			failed++
		}
	}
	if failed == len(targets) && failed > 0 {
		return "critical"
	}
	if failed > 0 || resources.PressureSkipped || opsResourceWarning(resources) {
		return "degraded"
	}
	if healthy == len(targets) && healthy > 0 && resources.ServiceState == "active" {
		return "healthy"
	}
	return "unknown"
}

func opsResourceWarning(resources opsResourceSnapshot) bool {
	return (resources.CPUPercent != nil && *resources.CPUPercent >= 80) ||
		(resources.MemoryPercent != nil && *resources.MemoryPercent >= 85) ||
		(resources.DiskPercent != nil && *resources.DiskPercent >= 90)
}

func appendOpsSample(sample opsSample) error {
	directory := opsStorageDir()
	if err := os.MkdirAll(directory, 0750); err != nil {
		return err
	}
	if err := enforceOpsRetention(directory, sample.At); err != nil {
		return err
	}
	path := filepath.Join(directory, "ops-"+sample.At.In(chinaLocation()).Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(sample); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := enforceOpsRetention(directory, sample.At); err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("运维监控持久化文件超过容量上限")
	}
	return nil
}

func enforceOpsRetention(directory string, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	type storedFile struct {
		path string
		day  time.Time
		size int64
	}
	var files []storedFile
	var total int64
	localNow := now.In(chinaLocation())
	cutoff := localNow.Add(-opsRetention)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "ops-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		day, parseErr := time.ParseInLocation("2006-01-02", strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "ops-"), ".jsonl"), chinaLocation())
		if parseErr != nil {
			if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
				return err
			}
			continue
		}
		path := filepath.Join(directory, entry.Name())
		cutoffDay := time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, chinaLocation())
		latestDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, chinaLocation()).Add(24 * time.Hour)
		if day.Before(cutoffDay) || day.After(latestDay) {
			if err := os.Remove(path); err != nil {
				return err
			}
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		files = append(files, storedFile{path: path, day: day, size: info.Size()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].day.Before(files[j].day) })
	for _, file := range files {
		if total <= opsMaxStorageBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			return err
		}
		total -= file.size
	}
	if total > opsMaxStorageBytes {
		return errors.New("运维监控持久化目录超过容量上限")
	}
	return nil
}

func loadOpsSamples(now time.Time) ([]opsSample, time.Time, int, error) {
	directory := opsStorageDir()
	loadFailed := false
	if err := enforceOpsRetention(directory, now); err != nil && !os.IsNotExist(err) {
		loadFailed = true
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, 0, nil
		}
		return nil, time.Time{}, 0, errors.New("SLA 历史目录不可读")
	}
	type loadFile struct {
		path string
		size int64
	}
	var files []loadFile
	fileTrackingHint := time.Time{}
	validTrackingHint := time.Time{}
	dropped := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "ops-") && strings.HasSuffix(entry.Name(), ".jsonl") {
			if info, infoErr := entry.Info(); infoErr == nil {
				files = append(files, loadFile{path: filepath.Join(directory, entry.Name()), size: info.Size()})
				if day, parseErr := time.ParseInLocation("2006-01-02", strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "ops-"), ".jsonl"), chinaLocation()); parseErr == nil && (fileTrackingHint.IsZero() || day.Before(fileTrackingHint)) {
					fileTrackingHint = day
				}
			} else {
				dropped++
				loadFailed = true
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path > files[j].path })
	selectedBytes := int64(0)
	selected := make([]loadFile, 0, len(files))
	for _, file := range files {
		if file.size > opsMaxStorageBytes || selectedBytes+file.size > opsMaxStorageBytes {
			continue
		}
		selected = append(selected, file)
		selectedBytes += file.size
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].path < selected[j].path })
	cutoff := now.Add(-opsRetention)
	samples := make([]opsSample, 0, min(len(selected)*1440, opsMaxInMemorySamples))
	ringPosition := 0
	for _, stored := range selected {
		file, openErr := os.Open(stored.path)
		if openErr != nil {
			dropped++
			loadFailed = true
			continue
		}
		scanner := bufio.NewScanner(io.LimitReader(file, opsMaxStorageBytes))
		scanner.Buffer(make([]byte, 4096), 256*1024)
		for scanner.Scan() {
			var sample opsSample
			if json.Unmarshal(scanner.Bytes(), &sample) != nil || sample.At.IsZero() || sample.At.After(now.Add(opsSampleInterval)) {
				dropped++
				loadFailed = true
				continue
			}
			if sample.At.Before(cutoff) {
				continue
			}
			normalizeOpsSample(&sample)
			candidateTracking := sample.TrackingSince
			if candidateTracking.IsZero() {
				candidateTracking = sample.At
			}
			if validTrackingHint.IsZero() || candidateTracking.Before(validTrackingHint) {
				validTrackingHint = candidateTracking
			}
			if len(samples) < opsMaxInMemorySamples {
				samples = append(samples, sample)
			} else {
				samples[ringPosition] = sample
				ringPosition = (ringPosition + 1) % opsMaxInMemorySamples
			}
		}
		if scanner.Err() != nil {
			dropped++
			loadFailed = true
		}
		file.Close()
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].At.Before(samples[j].At) })
	deduplicated := samples[:0]
	for _, sample := range samples {
		if len(deduplicated) > 0 && sample.At.Equal(deduplicated[len(deduplicated)-1].At) {
			deduplicated[len(deduplicated)-1] = sample
			continue
		}
		deduplicated = append(deduplicated, sample)
	}
	samples = deduplicated
	if len(samples) > opsMaxInMemorySamples {
		samples = append([]opsSample(nil), samples[len(samples)-opsMaxInMemorySamples:]...)
	}
	trackingHint := validTrackingHint
	if trackingHint.IsZero() {
		trackingHint = fileTrackingHint
	}
	if loadFailed {
		return samples, trackingHint, dropped, errors.New("SLA 历史数据读取不完整")
	}
	return samples, trackingHint, dropped, nil
}

func normalizeOpsSample(sample *opsSample) {
	sample.Status = canonicalOpsStatus(sample.Status)
	sample.Resources.ServiceState = canonicalServiceState(sample.Resources.ServiceState)
	for index := range sample.Targets {
		target := &sample.Targets[index]
		target.Scope = "server-side-synthetic"
		target.Status = canonicalOpsTargetStatus(target.Status)
		for _, configured := range opsTargets {
			if target.ID == configured.id {
				target.ID, target.Name, target.Host = configured.id, configured.name, configured.host
				break
			}
		}
	}
}

func canonicalOpsTargetStatus(status string) string {
	switch status {
	case "healthy":
		return "healthy"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func canonicalOpsStatus(status string) string {
	switch status {
	case "healthy":
		return "healthy"
	case "degraded":
		return "degraded"
	case "critical":
		return "critical"
	default:
		return "unknown"
	}
}

func canonicalServiceState(state string) string {
	switch state {
	case "active":
		return "active"
	case "inactive":
		return "inactive"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func trimOpsSamplesLocked(now time.Time) {
	cutoff := now.Add(-opsRetention)
	first := sort.Search(len(opsMonitor.samples), func(i int) bool { return !opsMonitor.samples[i].At.Before(cutoff) })
	if first > 0 {
		opsMonitor.samples = append([]opsSample(nil), opsMonitor.samples[first:]...)
	}
	if len(opsMonitor.samples) > opsMaxInMemorySamples {
		opsMonitor.samples = append([]opsSample(nil), opsMonitor.samples[len(opsMonitor.samples)-opsMaxInMemorySamples:]...)
	}
}

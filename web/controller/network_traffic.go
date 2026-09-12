package controller

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"trojan/core"

	gnet "github.com/shirou/gopsutil/net"
)

const (
	networkSampleInterval  = 2 * time.Second
	networkPersistInterval = 5 * time.Minute
)

type networkTrafficSnapshot struct {
	Interface     string `json:"interface"`
	MonthStart    string `json:"monthStart"`
	MonthEnd      string `json:"monthEnd"`
	Upload        uint64 `json:"upload"`
	Download      uint64 `json:"download"`
	Total         uint64 `json:"total"`
	UploadSpeed   uint64 `json:"uploadSpeed"`
	DownloadSpeed uint64 `json:"downloadSpeed"`
	TrackingSince string `json:"trackingSince"`
	CoverageMode  string `json:"coverageMode"`
	UpdatedAt     string `json:"updatedAt"`
	Ready         bool   `json:"ready"`
	Error         string `json:"error,omitempty"`
}

type networkTrafficRecord struct {
	MonthStart    string
	Interface     string
	BootID        string
	Upload        uint64
	Download      uint64
	LastBytesSent uint64
	LastBytesRecv uint64
	TrackingSince string
	CoverageMode  string
}

var networkTracker = struct {
	sync.RWMutex
	Snapshot      networkTrafficSnapshot
	BootID        string
	LastBytesSent uint64
	LastBytesRecv uint64
	LastSample    time.Time
}{}

var networkInitializer = struct {
	sync.Mutex
	NextAttempt time.Time
}{}

var networkSchema = struct {
	sync.Mutex
	Ready bool
}{}

func chinaLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func monthBounds(now time.Time) (string, string) {
	local := now.In(chinaLocation())
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func parseDefaultRoute(reader io.Reader) string {
	scanner := bufio.NewScanner(reader)
	bestInterface := ""
	bestMetric := uint64(^uint64(0))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || fields[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 64)
		if err != nil || flags&1 == 0 || !isExternalInterface(fields[0]) {
			continue
		}
		metric, err := strconv.ParseUint(fields[6], 10, 64)
		if err != nil {
			metric = bestMetric
		}
		if bestInterface == "" || metric < bestMetric {
			bestInterface = fields[0]
			bestMetric = metric
		}
	}
	return bestInterface
}

func isExternalInterface(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == "lo" {
		return false
	}
	internalPrefixes := []string{"docker", "veth", "br-", "cni", "flannel", "virbr", "tun", "tap", "wg", "tailscale", "zt"}
	for _, prefix := range internalPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

func defaultExternalInterface() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err == nil {
		defer file.Close()
		if name := parseDefaultRoute(file); name != "" {
			return name, nil
		}
	}
	counters, counterErr := gnet.IOCounters(true)
	if counterErr != nil {
		return "", counterErr
	}
	bestName := ""
	bestTotal := uint64(0)
	for _, counter := range counters {
		if !isExternalInterface(counter.Name) {
			continue
		}
		total := addTraffic(counter.BytesSent, counter.BytesRecv)
		if bestName == "" || total > bestTotal {
			bestName = counter.Name
			bestTotal = total
		}
	}
	if bestName == "" {
		return "", errors.New("未找到VPS默认外网网卡")
	}
	return bestName, nil
}

func interfaceCounters(name string) (uint64, uint64, error) {
	counters, err := gnet.IOCounters(true)
	if err != nil {
		return 0, 0, err
	}
	for _, counter := range counters {
		if counter.Name == name {
			return counter.BytesSent, counter.BytesRecv, nil
		}
	}
	return 0, 0, fmt.Errorf("外网网卡 %s 不存在", name)
}

func currentBootID() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ensureNetworkTrafficTable(db *sql.DB) error {
	networkSchema.Lock()
	defer networkSchema.Unlock()
	if networkSchema.Ready {
		return nil
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS network_traffic_monthly (
		month_start DATE NOT NULL,
		interface_name VARCHAR(64) NOT NULL,
		boot_id VARCHAR(64) NOT NULL,
		upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
		download BIGINT UNSIGNED NOT NULL DEFAULT 0,
		last_bytes_sent BIGINT UNSIGNED NOT NULL DEFAULT 0,
		last_bytes_recv BIGINT UNSIGNED NOT NULL DEFAULT 0,
		tracking_since VARCHAR(40) NOT NULL,
		baseline_mode VARCHAR(24) NOT NULL DEFAULT 'partial',
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (month_start)
	) DEFAULT CHARSET=utf8mb4;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE network_traffic_monthly
		ADD COLUMN IF NOT EXISTS baseline_mode VARCHAR(24) NOT NULL DEFAULT 'partial' AFTER tracking_since`); err != nil {
		return err
	}
	networkSchema.Ready = true
	return nil
}

func networkTrafficDB() (*sql.DB, error) {
	db := core.GetMysql().GetDB()
	if db == nil {
		return nil, errors.New("连接mysql失败")
	}
	if err := ensureNetworkTrafficTable(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func persistNetworkRecord(record networkTrafficRecord) error {
	if record.MonthStart == "" || record.Interface == "" {
		return nil
	}
	db, err := networkTrafficDB()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO network_traffic_monthly(
		month_start, interface_name, boot_id, upload, download, last_bytes_sent, last_bytes_recv, tracking_since, baseline_mode
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE interface_name=VALUES(interface_name), boot_id=VALUES(boot_id),
		upload=VALUES(upload), download=VALUES(download), last_bytes_sent=VALUES(last_bytes_sent),
		last_bytes_recv=VALUES(last_bytes_recv), tracking_since=VALUES(tracking_since), baseline_mode=VALUES(baseline_mode)`,
		record.MonthStart, record.Interface, record.BootID, record.Upload, record.Download,
		record.LastBytesSent, record.LastBytesRecv, record.TrackingSince, record.CoverageMode)
	return err
}

func currentNetworkRecord() networkTrafficRecord {
	networkTracker.RLock()
	defer networkTracker.RUnlock()
	return networkTrafficRecord{
		MonthStart:    networkTracker.Snapshot.MonthStart,
		Interface:     networkTracker.Snapshot.Interface,
		BootID:        networkTracker.BootID,
		Upload:        networkTracker.Snapshot.Upload,
		Download:      networkTracker.Snapshot.Download,
		LastBytesSent: networkTracker.LastBytesSent,
		LastBytesRecv: networkTracker.LastBytesRecv,
		TrackingSince: networkTracker.Snapshot.TrackingSince,
		CoverageMode:  networkTracker.Snapshot.CoverageMode,
	}
}

func setNetworkTrafficError(err error) {
	networkTracker.Lock()
	networkTracker.Snapshot.Error = err.Error()
	networkTracker.Snapshot.UpdatedAt = time.Now().In(chinaLocation()).Format(time.RFC3339)
	networkTracker.Unlock()
}

func ensureNetworkTrafficInitialized() {
	networkInitializer.Lock()
	if time.Now().Before(networkInitializer.NextAttempt) {
		networkInitializer.Unlock()
		return
	}
	networkInitializer.NextAttempt = time.Now().Add(30 * time.Second)
	networkInitializer.Unlock()
	if err := initializeNetworkTraffic(); err != nil {
		setNetworkTrafficError(err)
	}
}

func initializeNetworkTraffic() error {
	name, err := defaultExternalInterface()
	if err != nil {
		return err
	}
	sent, recv, err := interfaceCounters(name)
	if err != nil {
		return err
	}
	now := time.Now()
	monthStart, monthEnd := monthBounds(now)
	bootID := currentBootID()
	record := networkTrafficRecord{MonthStart: monthStart, Interface: name, BootID: bootID, LastBytesSent: sent, LastBytesRecv: recv, TrackingSince: now.In(chinaLocation()).Format(time.RFC3339), CoverageMode: "partial"}
	db, err := networkTrafficDB()
	if err != nil {
		return err
	}
	var stored networkTrafficRecord
	err = db.QueryRow(`SELECT DATE_FORMAT(month_start, '%Y-%m-%d'), interface_name, boot_id, upload, download,
		last_bytes_sent, last_bytes_recv, tracking_since, baseline_mode FROM network_traffic_monthly WHERE month_start=?`, monthStart).Scan(
		&stored.MonthStart, &stored.Interface, &stored.BootID, &stored.Upload, &stored.Download,
		&stored.LastBytesSent, &stored.LastBytesRecv, &stored.TrackingSince, &stored.CoverageMode)
	db.Close()
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		record.Upload = stored.Upload
		record.Download = stored.Download
		record.TrackingSince = stored.TrackingSince
		record.CoverageMode = stored.CoverageMode
		if stored.Interface == name && stored.BootID == bootID {
			if sent >= stored.LastBytesSent {
				record.Upload = addTraffic(record.Upload, sent-stored.LastBytesSent)
			}
			if recv >= stored.LastBytesRecv {
				record.Download = addTraffic(record.Download, recv-stored.LastBytesRecv)
			}
		}
	}
	networkTracker.Lock()
	networkTracker.Snapshot = networkTrafficSnapshot{
		Interface:     name,
		MonthStart:    monthStart,
		MonthEnd:      monthEnd,
		Upload:        record.Upload,
		Download:      record.Download,
		Total:         addTraffic(record.Upload, record.Download),
		TrackingSince: record.TrackingSince,
		CoverageMode:  record.CoverageMode,
		UpdatedAt:     now.In(chinaLocation()).Format(time.RFC3339),
		Ready:         true,
	}
	networkTracker.BootID = bootID
	networkTracker.LastBytesSent = sent
	networkTracker.LastBytesRecv = recv
	networkTracker.LastSample = now
	networkTracker.Unlock()
	return persistNetworkRecord(record)
}

func sampleNetworkTraffic() {
	networkTracker.RLock()
	name := networkTracker.Snapshot.Interface
	networkTracker.RUnlock()
	if name == "" {
		ensureNetworkTrafficInitialized()
		return
	}
	sent, recv, err := interfaceCounters(name)
	if err != nil {
		setNetworkTrafficError(err)
		return
	}
	now := time.Now()
	monthStart, monthEnd := monthBounds(now)
	var completedMonth *networkTrafficRecord
	networkTracker.Lock()
	if networkTracker.Snapshot.MonthStart != monthStart {
		record := networkTrafficRecord{
			MonthStart: networkTracker.Snapshot.MonthStart, Interface: name, BootID: networkTracker.BootID,
			Upload: networkTracker.Snapshot.Upload, Download: networkTracker.Snapshot.Download,
			LastBytesSent: networkTracker.LastBytesSent, LastBytesRecv: networkTracker.LastBytesRecv,
			TrackingSince: networkTracker.Snapshot.TrackingSince,
			CoverageMode:  networkTracker.Snapshot.CoverageMode,
		}
		completedMonth = &record
		networkTracker.Snapshot.MonthStart = monthStart
		networkTracker.Snapshot.MonthEnd = monthEnd
		networkTracker.Snapshot.Upload = 0
		networkTracker.Snapshot.Download = 0
		networkTracker.Snapshot.Total = 0
		networkTracker.Snapshot.UploadSpeed = 0
		networkTracker.Snapshot.DownloadSpeed = 0
		networkTracker.Snapshot.TrackingSince = now.In(chinaLocation()).Format(time.RFC3339)
		networkTracker.Snapshot.CoverageMode = "full"
	} else {
		elapsed := now.Sub(networkTracker.LastSample).Seconds()
		if elapsed <= 0 {
			elapsed = networkSampleInterval.Seconds()
		}
		var sentDelta, recvDelta uint64
		if sent >= networkTracker.LastBytesSent {
			sentDelta = sent - networkTracker.LastBytesSent
		}
		if recv >= networkTracker.LastBytesRecv {
			recvDelta = recv - networkTracker.LastBytesRecv
		}
		networkTracker.Snapshot.Upload = addTraffic(networkTracker.Snapshot.Upload, sentDelta)
		networkTracker.Snapshot.Download = addTraffic(networkTracker.Snapshot.Download, recvDelta)
		networkTracker.Snapshot.Total = addTraffic(networkTracker.Snapshot.Upload, networkTracker.Snapshot.Download)
		networkTracker.Snapshot.UploadSpeed = uint64(float64(sentDelta) / elapsed)
		networkTracker.Snapshot.DownloadSpeed = uint64(float64(recvDelta) / elapsed)
	}
	networkTracker.LastBytesSent = sent
	networkTracker.LastBytesRecv = recv
	networkTracker.LastSample = now
	networkTracker.Snapshot.UpdatedAt = now.In(chinaLocation()).Format(time.RFC3339)
	networkTracker.Snapshot.Error = ""
	networkTracker.Snapshot.Ready = true
	networkTracker.Unlock()
	if completedMonth != nil {
		if err := persistNetworkRecord(*completedMonth); err != nil {
			setNetworkTrafficError(err)
		}
		if err := persistNetworkRecord(currentNetworkRecord()); err != nil {
			setNetworkTrafficError(err)
		}
	}
}

func NetworkTrafficSnapshot() networkTrafficSnapshot {
	networkTracker.RLock()
	defer networkTracker.RUnlock()
	return networkTracker.Snapshot
}

// CollectTask samples only the default external interface in memory every two seconds and
// persists the cumulative monthly counter every five minutes.
func CollectTask() {
	ensureNetworkTrafficInitialized()
	go func() {
		sampleTicker := time.NewTicker(networkSampleInterval)
		persistTicker := time.NewTicker(networkPersistInterval)
		defer sampleTicker.Stop()
		defer persistTicker.Stop()
		for {
			select {
			case <-sampleTicker.C:
				sampleNetworkTraffic()
			case <-persistTicker.C:
				if err := persistNetworkRecord(currentNetworkRecord()); err != nil {
					setNetworkTrafficError(err)
				}
			}
		}
	}()
}

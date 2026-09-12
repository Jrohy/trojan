package controller

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func healthyTestTarget(id string, latency float64, at time.Time) opsTargetResult {
	checkedAt := at.Format(time.RFC3339)
	status := 204
	return opsTargetResult{
		ID: id, Name: id, Host: id + ".example", Scope: "server-side-synthetic",
		Status: "healthy", LatencyMS: &latency, HTTPStatus: &status, LastCheckedAt: &checkedAt,
	}
}

func TestTrojanConnectRequestUsesProvidedHash(t *testing.T) {
	credentialHash := strings.Repeat("a", 56)
	request, err := trojanConnectRequest(credentialHash, "www.google.com", 443)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(request[:56]); got != credentialHash {
		t.Fatalf("credential hash changed: %q", got)
	}
	if request[56] != '\r' || request[57] != '\n' || request[58] != 0x01 || request[59] != 0x03 {
		t.Fatalf("invalid Trojan request prefix: %x", request[56:60])
	}
	domainLength := int(request[60])
	if got := string(request[61 : 61+domainLength]); got != "www.google.com" {
		t.Fatalf("unexpected target: %q", got)
	}
	if got := binary.BigEndian.Uint16(request[61+domainLength : 63+domainLength]); got != 443 {
		t.Fatalf("unexpected port: %d", got)
	}
	if _, err := trojanConnectRequest("plain-text-password", "www.google.com", 443); err == nil {
		t.Fatal("plain text credential should be rejected")
	}
}

func TestNormalizeOpsWebsocketPath(t *testing.T) {
	for _, valid := range []string{"/websocket", "/websocket?token=fixed"} {
		if got, err := normalizeOpsWebsocketPath(valid); err != nil || got != valid {
			t.Fatalf("valid path %q rejected: got=%q err=%v", valid, got, err)
		}
	}
	for _, invalid := range []string{"", "websocket", "//other.example/path", "https://other.example/path"} {
		if _, err := normalizeOpsWebsocketPath(invalid); err == nil {
			t.Fatalf("unsafe path %q accepted", invalid)
		}
	}
}

func TestOpenOpsTrojanTransportUsesBinaryWebsocketFrames(t *testing.T) {
	payload := []byte{0, 1, 2, 3, 0xff}
	server := httptest.NewTLSServer(websocket.Handler(func(connection *websocket.Conn) {
		connection.PayloadType = websocket.BinaryFrame
		buffer := make([]byte, len(payload))
		if _, err := io.ReadFull(connection, buffer); err == nil {
			_, _ = connection.Write(buffer)
		}
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := net.Dial("tcp", serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	outer := tls.Client(raw, &tls.Config{ServerName: serverURL.Hostname(), InsecureSkipVerify: true}) // test server certificate only
	if err := outer.Handshake(); err != nil {
		t.Fatal(err)
	}
	tunnel, err := openOpsTrojanTransport(outer, opsProbeConfig{domain: serverURL.Host, websocketPath: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tunnel.Write(payload); err != nil {
		t.Fatal(err)
	}
	received := make([]byte, len(payload))
	if _, err := io.ReadFull(tunnel, received); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(received, payload) {
		t.Fatalf("websocket payload changed: got=%x want=%x", received, payload)
	}
}

func TestSummarizeOpsSamplesCountsMissingSlotsAsUnknown(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 3, 0, 0, time.UTC)
	tracking := now.Add(-3 * time.Minute)
	samples := []opsSample{
		{At: tracking, Targets: []opsTargetResult{healthyTestTarget("google", 100, tracking), healthyTestTarget("youtube", 300, tracking)}},
		{At: tracking.Add(time.Minute), Targets: []opsTargetResult{healthyTestTarget("google", 700, tracking), opsTargetResult{Status: "failed"}}},
	}
	summary := summarizeOpsSamples(samples, time.Hour, now, tracking)
	if summary.Expected != 8 || summary.Success != 3 || summary.Failed != 1 || summary.Unknown != 4 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	if summary.Coverage != float64(4)/8 {
		t.Fatalf("unexpected coverage: %v", summary.Coverage)
	}
	if summary.Availability == nil || *summary.Availability != .75 {
		t.Fatalf("unexpected availability: %v", summary.Availability)
	}
	if summary.P50MS == nil || *summary.P50MS != 300 {
		t.Fatalf("unexpected p50: %v", summary.P50MS)
	}
}

func TestBuildOpsHistoryEmitsMissingMinute(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 3, 0, 0, time.UTC)
	tracking := now.Add(-3 * time.Minute)
	samples := []opsSample{
		{At: tracking, Targets: []opsTargetResult{healthyTestTarget("google", 100, tracking), healthyTestTarget("youtube", 100, tracking)}},
		{At: tracking.Add(2 * time.Minute), Targets: []opsTargetResult{healthyTestTarget("google", 100, tracking), healthyTestTarget("youtube", 100, tracking)}},
	}
	history := buildOpsHistory(samples, time.Hour, now, tracking, 240)
	if len(history) != 4 {
		t.Fatalf("history points = %d, want 4", len(history))
	}
	if history[1].Unknown != 2 || history[1].Availability != nil {
		t.Fatalf("missing minute was not represented as unknown: %+v", history[1])
	}
	if history[3].Unknown != 2 {
		t.Fatalf("missing current scheduled minute was not represented: %+v", history[3])
	}
}

func TestMatureRollingHourHasSixtyExpectedSlots(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	tracking := now.Add(-24 * time.Hour)
	samples := make([]opsSample, 0, 60)
	for minute := 59; minute >= 0; minute-- {
		at := now.Add(-time.Duration(minute) * time.Minute)
		samples = append(samples, opsSample{At: at, Targets: []opsTargetResult{
			healthyTestTarget("google", 100, at), healthyTestTarget("youtube", 100, at),
		}})
	}
	summary := summarizeOpsSamples(samples, time.Hour, now, tracking)
	if summary.Expected != 120 || summary.Unknown != 0 || summary.Coverage != 1 {
		t.Fatalf("mature rolling window denominator is wrong: %+v", summary)
	}
	history := buildOpsHistory(samples, time.Hour, now, tracking, 240)
	if len(history) != 60 || history[0].Unknown != 0 || history[59].Unknown != 0 {
		t.Fatalf("mature rolling history has false gaps: first=%+v last=%+v len=%d", history[0], history[len(history)-1], len(history))
	}
}

func TestBuildOpsOverviewBoundsHistoryAndUsesExclusiveBuckets(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	samples := make([]opsSample, 0, 300)
	latencies := []float64{100, 300, 700, 1500, 3500}
	for index := 0; index < 300; index++ {
		at := now.Add(time.Duration(index-299) * time.Minute)
		latency := latencies[index%len(latencies)]
		samples = append(samples, opsSample{
			At: at, Status: "healthy", Resources: opsResourceSnapshot{SampledAt: opsStringPointer(at.Format(time.RFC3339)), ServiceState: "active"},
			Targets: []opsTargetResult{healthyTestTarget("google", latency, at), healthyTestTarget("youtube", latency, at)},
		})
	}
	overview := buildOpsOverview(samples, "24h", 24*time.Hour, now, samples[0].At, true, "")
	if len(overview.History) > 240 {
		t.Fatalf("history is not bounded: %d", len(overview.History))
	}
	total := 0
	for _, bucket := range overview.LatencyBuckets {
		total += bucket.Count
	}
	if total != 600 {
		t.Fatalf("exclusive bucket total = %d, want 600", total)
	}
	if overview.Scope != "server-side-synthetic" || overview.Sources[0].Available != true {
		t.Fatalf("unexpected source metadata: %+v", overview.Sources)
	}
}

func TestOpsSamplePersistenceDoesNotContainCredentialAndLoadsBoundedData(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(opsStorageDirEnv, directory)
	now := time.Now().UTC().Truncate(time.Second)
	sample := opsSample{
		At: now, Status: "healthy", Resources: opsResourceSnapshot{ServiceState: "active"},
		Targets: []opsTargetResult{healthyTestTarget("google", 123, now), opsTargetResult{ID: "youtube", Status: "failed", Error: "固定错误类别"}},
	}
	if err := appendOpsSample(sample); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "ops-"+now.In(chinaLocation()).Format("2006-01-02")+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "password") || strings.Contains(string(data), strings.Repeat("a", 56)) {
		t.Fatalf("persistent sample contains credential material: %s", data)
	}
	var decoded opsSample
	if err := json.Unmarshal(data[:len(data)-1], &decoded); err != nil {
		t.Fatal(err)
	}
	loaded, _, dropped, loadErr := loadOpsSamples(now.Add(time.Second))
	if loadErr != nil || dropped != 0 {
		t.Fatalf("clean history reported data loss: dropped=%d err=%v", dropped, loadErr)
	}
	if len(loaded) != 1 || loaded[0].Status != "healthy" || loaded[0].Targets[1].Status != "failed" {
		t.Fatalf("unexpected loaded samples: %+v", loaded)
	}
}

func TestLoadOpsSamplesReportsCorruptHistory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(opsStorageDirEnv, directory)
	now := time.Now().UTC().Truncate(time.Second)
	path := filepath.Join(directory, "ops-"+now.In(chinaLocation()).Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(path, []byte("{not-json}\n"), 0640); err != nil {
		t.Fatal(err)
	}
	loaded, trackingHint, dropped, err := loadOpsSamples(now)
	if err == nil || dropped != 1 || len(loaded) != 0 || trackingHint.IsZero() {
		t.Fatalf("corruption was hidden: loaded=%d hint=%v dropped=%d err=%v", len(loaded), trackingHint, dropped, err)
	}
	expectedHint, parseErr := time.ParseInLocation("2006-01-02", now.In(chinaLocation()).Format("2006-01-02"), chinaLocation())
	if parseErr != nil || !trackingHint.Equal(expectedHint) {
		t.Fatalf("tracking hint timezone mismatch: got=%v want=%v", trackingHint, expectedHint)
	}
	healthy := opsSample{
		At: now, Status: "healthy", Resources: opsResourceSnapshot{SampledAt: opsStringPointer(now.Format(time.RFC3339)), ServiceState: "active"},
		Targets: []opsTargetResult{healthyTestTarget("google", 100, now), healthyTestTarget("youtube", 100, now)},
	}
	overview := buildOpsOverview([]opsSample{healthy}, "1h", time.Hour, now, now, false, "历史数据读取不完整，已丢弃 1 条损坏或不可读记录")
	if overview.Status != "degraded" || len(overview.Sources) < 3 || overview.Sources[2].Available || overview.Sources[2].Message == "" {
		t.Fatalf("corrupt persistence did not degrade a healthy view: status=%s source=%+v", overview.Status, overview.Sources)
	}
}

func TestLoadOpsSamplesAllowsMissingInitialDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "not-created")
	t.Setenv(opsStorageDirEnv, directory)
	loaded, trackingHint, dropped, err := loadOpsSamples(time.Now().UTC())
	if err != nil || dropped != 0 || len(loaded) != 0 || !trackingHint.IsZero() {
		t.Fatalf("missing initial directory should be clean: loaded=%d hint=%v dropped=%d err=%v", len(loaded), trackingHint, dropped, err)
	}
}

func TestDeriveOpsStatusDoesNotTreatUnknownAsHealthy(t *testing.T) {
	resources := opsResourceSnapshot{ServiceState: "active"}
	unknown := []opsTargetResult{{Status: "unknown"}, {Status: "unknown"}}
	if got := deriveOpsStatus(resources, unknown); got != "unknown" {
		t.Fatalf("unknown probes produced %q", got)
	}
	failed := []opsTargetResult{{Status: "failed"}, {Status: "failed"}}
	if got := deriveOpsStatus(resources, failed); got != "critical" {
		t.Fatalf("failed probes produced %q", got)
	}
}

func TestBuildOpsEventsEmitsTransitionsWithoutDuplicateOutageRows(t *testing.T) {
	start := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	samples := []opsSample{
		{At: start, Targets: []opsTargetResult{healthyTestTarget("google", 100, start)}},
		{At: start.Add(time.Minute), Targets: []opsTargetResult{{ID: "google", Name: "Google", Status: "failed", Error: "操作超时"}}},
		{At: start.Add(2 * time.Minute), Targets: []opsTargetResult{{ID: "google", Name: "Google", Status: "failed", Error: "操作超时"}}},
		{At: start.Add(3 * time.Minute), Targets: []opsTargetResult{healthyTestTarget("google", 100, start)}},
	}
	events := buildOpsEvents(samples, 50)
	if len(events) != 2 {
		t.Fatalf("prolonged outage generated %d events, want start+recovery: %+v", len(events), events)
	}
	if events[0].Severity != "critical" || events[1].Severity != "info" || !strings.Contains(events[1].Message, "2m0s") {
		t.Fatalf("unexpected transition events: %+v", events)
	}
}

func TestBuildOpsEventsBreaksDurationAcrossSamplingGap(t *testing.T) {
	start := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	samples := []opsSample{
		{At: start, Targets: []opsTargetResult{{ID: "google", Name: "Google", Status: "failed"}}},
		{At: start.Add(5 * time.Minute), Targets: []opsTargetResult{healthyTestTarget("google", 100, start)}},
	}
	events := buildOpsEvents(samples, 50)
	if len(events) != 3 || events[0].Severity != "critical" || events[1].Severity != "warning" || events[2].Severity != "info" {
		t.Fatalf("gap transitions are incomplete: %+v", events)
	}
	if strings.Contains(events[2].Message, "故障持续") {
		t.Fatalf("recovery after unknown gap claims an exact duration: %+v", events[2])
	}
}

package controller

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestOpsThirtyDayCapacity(t *testing.T) {
	if os.Getenv("OPS_CAPACITY_TEST") != "1" {
		t.Skip("opt-in bounded capacity check")
	}
	now := time.Now().UTC()
	fixture := opsSample{At: now, Status: "healthy", Resources: opsResourceSnapshot{
		SampledAt: opsStringPointer(now.Format(time.RFC3339)), ServiceState: "active",
		CPUPercent: opsFloatPointer(10), MemoryPercent: opsFloatPointer(50), SwapPercent: opsFloatPointer(30), DiskPercent: opsFloatPointer(73), Load1: opsFloatPointer(.2), DiskFreeBytes: opsUint64Pointer(7 * 1024 * 1024 * 1024), NetworkUpBPS: opsUint64Pointer(12345), NetworkDownBPS: opsUint64Pointer(12345),
	}, Targets: []opsTargetResult{healthyTestTarget("google", 100, now), healthyTestTarget("youtube", 200, now)}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	const count = 30 * 24 * 60
	if int64((len(raw)+1)*count) > opsMaxStorageBytes {
		t.Fatal("typical 30-day dataset exceeds disk budget")
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	samples := make([]opsSample, count)
	for i := range samples {
		if err := json.Unmarshal(raw, &samples[i]); err != nil {
			t.Fatal(err)
		}
		samples[i].At = now.Add(-time.Duration(count-1-i) * time.Minute)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	heap := after.HeapAlloc - before.HeapAlloc
	started := time.Now()
	view := buildOpsOverview(samples, "30d", 30*24*time.Hour, now, samples[0].At, true, "")
	elapsed := time.Since(started)
	if heap > 64*1024*1024 {
		t.Fatalf("history live heap exceeds 64MiB: %d", heap)
	}
	if len(view.History) > 240 || view.Overview.Failed != 0 || view.Overview.Coverage != 1 {
		t.Fatalf("capacity fixture aggregation invalid: %+v", view.Overview)
	}
	t.Logf("30d samples=%d serializedMiB=%.2f liveHeapMiB=%.2f uncachedOverview=%s points=%d", count, float64((len(raw)+1)*count)/(1024*1024), float64(heap)/(1024*1024), elapsed, len(view.History))
	runtime.KeepAlive(samples)
}

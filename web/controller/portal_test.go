package controller

import (
	"math"
	"testing"
	"time"
)

func TestPortalStatusPriorityAndBoundaries(t *testing.T) {
	now := time.Date(2026, 8, 2, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	tests := []struct {
		name       string
		expiryDate string
		quota      int64
		used       uint64
		wantStatus string
		wantDays   *int
	}{
		{name: "normal", expiryDate: "2026-08-20", quota: 100, used: 20, wantStatus: "normal", wantDays: intPointer(18)},
		{name: "expiring soon boundary", expiryDate: "2026-08-09", quota: 100, used: 20, wantStatus: "expiring_soon", wantDays: intPointer(7)},
		{name: "expires today", expiryDate: "2026-08-02", quota: 100, used: 20, wantStatus: "expiring_soon", wantDays: intPointer(0)},
		{name: "expired wins over exhausted", expiryDate: "2026-08-01", quota: 0, used: 20, wantStatus: "expired", wantDays: intPointer(-1)},
		{name: "traffic exhausted", expiryDate: "2026-09-01", quota: 20, used: 20, wantStatus: "traffic_exhausted", wantDays: intPointer(30)},
		{name: "unlimited", expiryDate: "", quota: -1, used: math.MaxUint64, wantStatus: "normal", wantDays: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, _, days, err := portalStatus(now, test.expiryDate, test.quota, test.used)
			if err != nil {
				t.Fatalf("portalStatus returned error: %v", err)
			}
			if status != test.wantStatus {
				t.Fatalf("status = %q, want %q", status, test.wantStatus)
			}
			if !sameOptionalInt(days, test.wantDays) {
				t.Fatalf("days = %v, want %v", optionalIntValue(days), optionalIntValue(test.wantDays))
			}
		})
	}
}

func TestPortalStatusRejectsInvalidExpiry(t *testing.T) {
	if _, _, _, err := portalStatus(time.Now(), "2026/08/02", 100, 1); err == nil {
		t.Fatal("expected invalid expiry error")
	}
}

func TestAddTrafficSaturates(t *testing.T) {
	if got := addTraffic(math.MaxUint64-2, 10); got != math.MaxUint64 {
		t.Fatalf("addTraffic overflow result = %d", got)
	}
}

func intPointer(value int) *int { return &value }

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func optionalIntValue(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

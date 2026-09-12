package controller

import (
	"strings"
	"testing"
	"time"
)

func TestParseDefaultRouteUsesLowestMetricExternalInterface(t *testing.T) {
	routes := `Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
docker0 00000000 00000000 0001 0 0 0 00000000 0 0 0
eth1 00000000 01010101 0003 0 0 200 00000000 0 0 0
eth0 00000000 01010101 0003 0 0 100 00000000 0 0 0
eth2 00000000 01010101 0000 0 0 1 00000000 0 0 0`
	if got := parseDefaultRoute(strings.NewReader(routes)); got != "eth0" {
		t.Fatalf("parseDefaultRoute() = %q, want eth0", got)
	}
}

func TestExternalInterfaceFilter(t *testing.T) {
	for _, name := range []string{"lo", "docker0", "veth123", "br-abcd", "tun0", "wg0", "tailscale0"} {
		if isExternalInterface(name) {
			t.Errorf("isExternalInterface(%q) = true", name)
		}
	}
	for _, name := range []string{"eth0", "ens3", "enp1s0"} {
		if !isExternalInterface(name) {
			t.Errorf("isExternalInterface(%q) = false", name)
		}
	}
}

func TestMonthBoundsUseChinaCalendar(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC)
	start, end := monthBounds(now)
	if start != "2026-08-01" || end != "2026-08-31" {
		t.Fatalf("monthBounds() = %s..%s", start, end)
	}
}

package controller

import (
	"strings"
	"testing"
)

func TestOpsIdentityNeverFiltersOrdinaryUsers(t *testing.T) {
	hash := strings.Repeat("a", 56)
	t.Setenv("TROJAN_OPS_CREDENTIAL_HASH", "")
	if isOpsProbeHash("") || isOpsProbeHash(hash) {
		t.Fatal("empty configuration must not filter users")
	}
	t.Setenv("TROJAN_OPS_CREDENTIAL_HASH", hash)
	if !isOpsProbeHash(hash) || !isOpsProbeHash(strings.ToUpper(hash)) {
		t.Fatal("dedicated probe not recognized")
	}
	if isOpsProbeHash(strings.Repeat("b", 56)) {
		t.Fatal("ordinary user filtered")
	}
}

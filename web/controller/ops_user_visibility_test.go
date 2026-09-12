package controller

import (
	"encoding/json"
	"strings"
	"testing"

	"trojan/core"
)

func TestVisibleUsersRemovesOpsProbeWithoutMutatingInput(t *testing.T) {
	probeHash := strings.Repeat("a", 56)
	t.Setenv("TROJAN_OPS_CREDENTIAL_HASH", probeHash)
	users := []*core.User{
		{ID: 1, Username: "ordinary", EncryptPass: strings.Repeat("b", 56), Password: "visible"},
		{ID: 2, Username: "ops-probe", EncryptPass: probeHash, Password: "secret"},
	}

	visible := visibleUsers(users)
	if len(visible) != 1 || visible[0].Username != "ordinary" {
		t.Fatalf("unexpected visible users: %+v", visible)
	}
	if len(users) != 2 {
		t.Fatalf("input was mutated: %+v", users)
	}
	encoded, err := json.Marshal(visible)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "ops-probe") || strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), probeHash) {
		t.Fatalf("probe identity leaked: %s", encoded)
	}
}

func TestVisibleUserPageExcludesProbeFromRowsAndTotals(t *testing.T) {
	probeHash := strings.Repeat("c", 56)
	t.Setenv("TROJAN_OPS_CREDENTIAL_HASH", probeHash)
	users := []*core.User{
		{ID: 1, Username: "first", EncryptPass: strings.Repeat("d", 56)},
		{ID: 2, Username: "ops-probe", EncryptPass: probeHash},
		{ID: 3, Username: "second", EncryptPass: strings.Repeat("e", 56)},
	}

	firstPage := visibleUserPage(users, 1, 1)
	secondPage := visibleUserPage(users, 2, 1)
	if firstPage.Total != 2 || firstPage.PageNum != 2 || len(firstPage.DataList) != 1 || firstPage.DataList[0].Username != "first" {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}
	if secondPage.Total != 2 || len(secondPage.DataList) != 1 || secondPage.DataList[0].Username != "second" {
		t.Fatalf("unexpected second page: %+v", secondPage)
	}
}

func TestRejectOpsProbeMutationUsesSecretFreeError(t *testing.T) {
	probeHash := strings.Repeat("f", 56)
	t.Setenv("TROJAN_OPS_CREDENTIAL_HASH", probeHash)
	if err := rejectOpsProbeMutation(&core.User{EncryptPass: strings.Repeat("e", 56)}); err != nil {
		t.Fatalf("ordinary user rejected: %v", err)
	}
	err := rejectOpsProbeMutation(&core.User{Username: "private-probe", EncryptPass: probeHash, Password: "private-password"})
	if err == nil || err.Error() != opsProbeMutationMessage {
		t.Fatalf("probe mutation result = %v", err)
	}
	for _, secret := range []string{probeHash, "private-probe", "private-password"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("mutation error leaks %q: %s", secret, err)
		}
	}
}

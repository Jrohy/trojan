package controller

import (
	"crypto/subtle"
	"os"
	"strings"
)

// Dedicated synthetic checks must not appear as ordinary user activity.
func isOpsProbeHash(hash string) bool {
	probe := strings.TrimSpace(os.Getenv("TROJAN_OPS_CREDENTIAL_HASH"))
	return len(probe) == 56 && len(hash) == 56 && subtle.ConstantTimeCompare([]byte(strings.ToLower(probe)), []byte(strings.ToLower(hash))) == 1
}

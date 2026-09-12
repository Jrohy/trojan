package web

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4"
)

func TestOpsRouteRoleIsolation(t *testing.T) {
	// core selects its database directory at package initialization. Require an
	// explicit isolated path, never initialize test signing keys in production.
	if !strings.HasPrefix(os.Getenv("TROJAN_MANAGER_DB_PATH"), "/tmp/trojan-ops-test-") {
		t.Skip("requires isolated test manager directory")
	}
	gin.SetMode(gin.TestMode)
	router := newRouter(10, false)
	for _, tc := range []struct {
		role string
		code int
	}{{"", 401}, {"user", 403}, {"admin", 200}} {
		req := httptest.NewRequest("GET", "/common/ops?range=1h", nil)
		if tc.role != "" {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"username": "test", "role": tc.role, "exp": time.Now().Add(time.Minute).Unix(), "orig_iat": time.Now().Unix()})
			signed, err := token.SignedString([]byte(getSecretKey()))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+signed)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("role=%s status=%d want=%d", tc.role, w.Code, tc.code)
		}
		if tc.code == 200 && !strings.Contains(w.Body.String(), "schemaVersion") {
			t.Fatal("missing ops payload")
		}
	}
	req := httptest.NewRequest("GET", "/common/ops", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("invalid token accepted")
	}
}

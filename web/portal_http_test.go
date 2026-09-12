package web

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"trojan/core"
)

func portalTestRequest(router http.Handler, method, path string, values url.Values, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	request := httptest.NewRequest(method, path, body)
	if values != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func portalTestCookies(response *httptest.ResponseRecorder) []*http.Cookie {
	return response.Result().Cookies()
}

func requireHTTPStatus(t *testing.T, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("HTTP status = %d, want %d; body=%s", response.Code, status, response.Body.String())
	}
}

func requireSuccessBody(t *testing.T, response *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	requireHTTPStatus(t, response, http.StatusOK)
	var body map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if message, _ := body["Msg"].(string); message != "success" {
		t.Fatalf("response Msg = %q; body=%s", message, response.Body.String())
	}
	return body
}

func sha224Hex(value string) string {
	digest := sha256.Sum224([]byte(value))
	return hex.EncodeToString(digest[:])
}

func TestPortalHTTPDirectLoginAndRoleIsolation(t *testing.T) {
	if os.Getenv("PORTAL_HTTP_E2E") != "1" {
		t.Skip("PORTAL_HTTP_E2E is not configured")
	}
	mysql := core.GetMysql()
	router := newRouter(120, false)

	adminDigest := strings.Repeat("a", 56)
	register := portalTestRequest(router, http.MethodPost, "/auth/register", url.Values{"password": {adminDigest}}, nil)
	requireHTTPStatus(t, register, http.StatusOK)
	adminLogin := portalTestRequest(router, http.MethodPost, "/auth/login", url.Values{"username": {"admin"}, "password": {adminDigest}}, nil)
	requireHTTPStatus(t, adminLogin, http.StatusOK)
	adminCookies := portalTestCookies(adminLogin)
	if len(adminCookies) == 0 || !adminCookies[0].HttpOnly || adminCookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected auth cookie: %#v", adminCookies)
	}

	connectionPassword := "portal-e2e-connection-password"
	connectionPasswordBase64 := base64.StdEncoding.EncodeToString([]byte(connectionPassword))
	createUser := portalTestRequest(router, http.MethodPost, "/trojan/user", url.Values{
		"username": {"portal-e2e-user"},
		"password": {connectionPasswordBase64},
	}, adminCookies)
	requireSuccessBody(t, createUser)
	user := mysql.GetUserByName("portal-e2e-user")
	if user == nil {
		t.Fatal("created test user was not found")
	}
	db := mysql.GetDB()
	if db == nil {
		t.Fatal("test DB is nil")
	}
	defer db.Close()
	defer db.Exec(`DELETE FROM users WHERE id = ?`, user.ID)

	userID := fmt.Sprint(user.ID)
	requireSuccessBody(t, portalTestRequest(router, http.MethodPost, "/trojan/data", url.Values{"id": {userID}, "quota": {"1000"}}, adminCookies))
	requireSuccessBody(t, portalTestRequest(router, http.MethodPost, "/trojan/user/expire", url.Values{"id": {userID}, "useDays": {"30"}}, adminCookies))
	if _, err := db.Exec(`UPDATE users SET upload = 300, download = 200 WHERE id = ?`, user.ID); err != nil {
		t.Fatalf("set test traffic: %v", err)
	}

	userDigest := sha224Hex(connectionPassword)
	userLogin := portalTestRequest(router, http.MethodPost, "/auth/login", url.Values{"username": {"portal-e2e-user"}, "password": {userDigest}}, nil)
	requireHTTPStatus(t, userLogin, http.StatusOK)
	userCookies := portalTestCookies(userLogin)
	userSession := portalTestRequest(router, http.MethodGet, "/auth/loginUser", nil, userCookies)
	requireHTTPStatus(t, userSession, http.StatusOK)
	if !strings.Contains(userSession.Body.String(), `"role":"user"`) {
		t.Fatalf("user role missing: %s", userSession.Body.String())
	}

	summary := portalTestRequest(router, http.MethodGet, "/portal/me/summary", nil, userCookies)
	summaryBody := requireSuccessBody(t, summary)
	summaryJSON := summary.Body.String()
	for _, forbidden := range []string{"targetHost", "clientIP", "accessCount", "userHash", "password", "passwordShow", "serverAddr"} {
		if strings.Contains(summaryJSON, forbidden) {
			t.Fatalf("personal summary contains forbidden field %q: %s", forbidden, summaryJSON)
		}
	}
	data, _ := summaryBody["Data"].(map[string]interface{})
	traffic, _ := data["traffic"].(map[string]interface{})
	if data["username"] != "portal-e2e-user" || data["status"] != "normal" || traffic["used"] != float64(500) || traffic["remaining"] != float64(500) {
		t.Fatalf("unexpected personal summary: %s", summaryJSON)
	}

	for _, path := range []string{"/trojan/user", "/trojan/access-history", "/common/serverInfo"} {
		response := portalTestRequest(router, http.MethodGet, path, nil, userCookies)
		requireHTTPStatus(t, response, http.StatusForbidden)
	}
	requireHTTPStatus(t, portalTestRequest(router, http.MethodGet, "/portal/me/summary", nil, adminCookies), http.StatusForbidden)
	requireHTTPStatus(t, portalTestRequest(router, http.MethodPost, "/trojan/user/portal", url.Values{"id": {userID}}, adminCookies), http.StatusNotFound)
	adminUsers := portalTestRequest(router, http.MethodGet, "/trojan/user", nil, adminCookies)
	requireSuccessBody(t, adminUsers)
	if strings.Contains(adminUsers.Body.String(), "portalStatus") {
		t.Fatalf("administrator response still exposes portal configuration: %s", adminUsers.Body.String())
	}
	portalPage := portalTestRequest(router, http.MethodGet, "/portal", nil, nil)
	requireHTTPStatus(t, portalPage, http.StatusOK)
	if strings.Contains(portalPage.Body.String(), "访问历史") {
		t.Fatal("personal dashboard exposes access-history terminology")
	}

	newPassword := "portal-e2e-new-connection-password"
	updateUser := portalTestRequest(router, http.MethodPost, "/trojan/user/update", url.Values{
		"id": {userID}, "username": {"portal-e2e-user"},
		"password": {base64.StdEncoding.EncodeToString([]byte(newPassword))},
	}, adminCookies)
	requireSuccessBody(t, updateUser)
	requireHTTPStatus(t, portalTestRequest(router, http.MethodGet, "/portal/me/summary", nil, userCookies), http.StatusUnauthorized)
	requireHTTPStatus(t, portalTestRequest(router, http.MethodPost, "/auth/login", url.Values{"username": {"portal-e2e-user"}, "password": {userDigest}}, nil), http.StatusUnauthorized)
	newUserLogin := portalTestRequest(router, http.MethodPost, "/auth/login", url.Values{"username": {"portal-e2e-user"}, "password": {sha224Hex(newPassword)}}, nil)
	requireHTTPStatus(t, newUserLogin, http.StatusOK)
}

package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
	if cfg.Endpoint != "/metrics" {
		t.Errorf("Endpoint = %q", cfg.Endpoint)
	}
	if !cfg.IncludeSystem || !cfg.IncludeRuntime {
		t.Error("IncludeSystem and IncludeRuntime should default to true")
	}
}

func TestNew_ExposesAppInfo(t *testing.T) {
	m := New(DefaultConfig(), "1.2.3", "abcdef", "2026-01-01")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "casman_app_info") {
		t.Error("missing app_info metric")
	}
	if !strings.Contains(body, `version="1.2.3"`) {
		t.Error("missing version label")
	}
}

func TestHandler_TokenAuth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Token = "secret-token"
	m := New(cfg, "1.0.0", "abc", "2026-01-01")

	// No auth header
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no auth: status = %d, want 401", w.Code)
	}

	// Wrong token
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w = httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", w.Code)
	}

	// Correct token
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	w = httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("correct token: status = %d, want 200", w.Code)
	}
}

func TestRecordAndSetMethods_DoNotPanic(t *testing.T) {
	m := New(DefaultConfig(), "1.0.0", "abc", "2026-01-01")

	m.RecordHTTPRequest("GET", "/api/v1/users/42", 200, 15*time.Millisecond, 128, 512)
	m.IncActiveRequests()
	m.DecActiveRequests()
	m.RecordDBQuery("select", "users", time.Millisecond)
	m.RecordDBError("select", "timeout")
	m.SetDBConnections(5, 2)
	m.RecordCacheHit("pages")
	m.RecordCacheMiss("pages")
	m.RecordCacheEviction("pages")
	m.SetCacheSize("pages", 10, 2048)
	m.RecordAuthAttempt("token", "success")
	m.SetActiveSessions(3)
	m.SetSystemMetrics(12.5, 40.0, 1024, 4096)
	m.SetDiskMetrics("/data", 55.5, 1000, 2000)
}

func TestSetSystemMetrics_NoopWhenDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IncludeSystem = false
	m := New(cfg, "1.0.0", "abc", "2026-01-01")
	// Should not panic even though the gauges were never registered.
	m.SetSystemMetrics(1, 2, 3, 4)
	m.SetDiskMetrics("/data", 1, 2, 3)
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/api/v1/users/42":                            "/api/v1/users/:id",
		"/api/v1/users/550e8400-e29b-41d4-a716-446655440000": "/api/v1/users/:id",
		"/api/v1/users":                                "/api/v1/users",
		"/":                                             "/",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	if !isUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("expected valid UUID to match")
	}
	if isUUID("not-a-uuid") {
		t.Error("expected invalid UUID to not match")
	}
	if isUUID("550e8400-e29b-41d4-a716-44665544000g") {
		t.Error("expected invalid hex char to not match")
	}
	if isUUID("550e8400xe29b-41d4-a716-446655440000") {
		t.Error("expected wrong dash position to not match")
	}
}

func TestIsNumericID(t *testing.T) {
	if !isNumericID("12345") {
		t.Error("expected numeric string to match")
	}
	if isNumericID("") {
		t.Error("expected empty string to not match")
	}
	if isNumericID("12a45") {
		t.Error("expected non-numeric to not match")
	}
	if isNumericID(strings.Repeat("1", 21)) {
		t.Error("expected 21-digit string to exceed max length")
	}
}

func TestIsHexDigit(t *testing.T) {
	for _, c := range []byte("0123456789abcdefABCDEF") {
		if !isHexDigit(c) {
			t.Errorf("isHexDigit(%q) = false, want true", c)
		}
	}
	for _, c := range []byte("gGzZ-_ ") {
		if isHexDigit(c) {
			t.Errorf("isHexDigit(%q) = true, want false", c)
		}
	}
}

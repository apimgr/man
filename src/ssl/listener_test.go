package ssl

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirectHandler_301Https(t *testing.T) {
	h := redirectHandler("example.com")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo?bar=1", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want 301", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://example.com/foo") {
		t.Errorf("Location = %q, want https://example.com/foo...", loc)
	}
}

func TestRedirectHandler_StripsPort(t *testing.T) {
	h := redirectHandler("example.com")
	req := httptest.NewRequest(http.MethodGet, "http://example.com:8080/x", nil)
	req.Host = "example.com:8080"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	loc := w.Header().Get("Location")
	if loc != "https://example.com/x" {
		t.Errorf("Location = %q, want https://example.com/x", loc)
	}
}

func TestHostFromAddr(t *testing.T) {
	if got := hostFromAddr("0.0.0.0:443"); got != "0.0.0.0" {
		t.Errorf("hostFromAddr = %q, want 0.0.0.0", got)
	}
	if got := hostFromAddr("[::1]:443"); got != "::1" {
		t.Errorf("hostFromAddr = %q, want ::1", got)
	}
	if got := hostFromAddr("malformed"); got != "" {
		t.Errorf("hostFromAddr should be empty for bad input, got %q", got)
	}
}

package swagger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	h := New("1.2.3", "https://example.com")
	if h.spec == nil {
		t.Fatal("expected spec to be generated")
	}
	if h.spec.Info.Version != "1.2.3" {
		t.Errorf("Version = %q", h.spec.Info.Version)
	}
	if h.spec.Servers[0].URL != "https://example.com" {
		t.Errorf("Server URL = %q", h.spec.Servers[0].URL)
	}
	if len(h.spec.Paths) == 0 {
		t.Error("expected non-empty Paths")
	}
}

func TestServeSpec(t *testing.T) {
	h := New("1.0.0", "https://example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi", nil)
	w := httptest.NewRecorder()

	h.ServeSpec(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}

	var spec Spec
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if spec.OpenAPI != "3.0.0" {
		t.Errorf("OpenAPI = %q", spec.OpenAPI)
	}
	if _, ok := spec.Paths["/api/v1/healthz"]; !ok {
		t.Error("expected /api/v1/healthz path in spec")
	}
}

func TestServeUI(t *testing.T) {
	h := New("1.0.0", "https://example.com")
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()

	h.ServeUI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.Contains(w.Body.String(), "swagger-ui") {
		t.Error("expected swagger-ui markup in body")
	}
}

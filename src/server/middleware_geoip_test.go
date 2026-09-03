package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeGeoIP struct {
	available bool
	blocked   map[string]bool
	country   map[string]string
}

func (f *fakeGeoIP) IsAvailable() bool          { return f.available }
func (f *fakeGeoIP) IsBlocked(ip string) bool   { return f.blocked[ip] }
func (f *fakeGeoIP) GetCountry(ip string) string { return f.country[ip] }

func handlerThatChecksCountry(t *testing.T, want string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetCountry(r); got != want {
			t.Errorf("GetCountry = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestGeoIPMiddleware_BlocksDeniedCountry(t *testing.T) {
	g := &fakeGeoIP{
		available: true,
		blocked:   map[string]bool{"203.0.113.5": true},
	}
	mw := GeoIPMiddleware(g)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if called {
		t.Error("downstream handler should not run for blocked IP")
	}
}

func TestGeoIPMiddleware_AllowsAndStashesCountry(t *testing.T) {
	g := &fakeGeoIP{
		available: true,
		country:   map[string]string{"198.51.100.10": "US"},
	}
	mw := GeoIPMiddleware(g)
	h := mw(handlerThatChecksCountry(t, "US"))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "198.51.100.10:443"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGeoIPMiddleware_PassesThroughWhenUnavailable(t *testing.T) {
	g := &fakeGeoIP{available: false, blocked: map[string]bool{"1.2.3.4": true}}
	mw := GeoIPMiddleware(g)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:80"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !called {
		t.Error("downstream handler must run when geoip is unavailable")
	}
}

func TestGeoIPMiddleware_SkipsHealthAndStatic(t *testing.T) {
	g := &fakeGeoIP{available: true, blocked: map[string]bool{"1.2.3.4": true}}
	mw := GeoIPMiddleware(g)
	for _, p := range []string{"/healthz", "/metrics", "/static/style.css", "/.well-known/acme-challenge/abc"} {
		called := false
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
		req := httptest.NewRequest("GET", p, nil)
		req.RemoteAddr = "1.2.3.4:1"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if !called {
			t.Errorf("%s should bypass geoip middleware", p)
		}
	}
}

func TestGeoIPMiddleware_PrivateIPSkipped(t *testing.T) {
	g := &fakeGeoIP{available: true, blocked: map[string]bool{"127.0.0.1": true}}
	mw := GeoIPMiddleware(g)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:80"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if !called {
		t.Error("private IPs should bypass enforcement")
	}
}

func TestSkipGeoIPPath(t *testing.T) {
	cases := map[string]bool{
		"/":                           false,
		"/healthz":                    true,
		"/metrics":                    true,
		"/robots.txt":                 true,
		"/static/x":                   true,
		"/.well-known/security.txt":   true,
		"/admin/server/ssl":           false,
	}
	for path, want := range cases {
		if got := skipGeoIPPath(path); got != want {
			t.Errorf("skipGeoIPPath(%q) = %v, want %v", path, got, want)
		}
	}
}

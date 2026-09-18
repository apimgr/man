package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNegotiateFormat_QueryParam(t *testing.T) {
	cases := map[string]ResponseFormat{
		"json": FormatJSON,
		"text": FormatText,
		"txt":  FormatText,
		"html": FormatHTML,
	}
	for q, want := range cases {
		r := httptest.NewRequest("GET", "/?format="+q, nil)
		if got := negotiateFormat(r); got != want {
			t.Errorf("negotiateFormat(format=%s) = %q, want %q", q, got, want)
		}
	}
}

func TestNegotiateFormat_APIAlwaysJSON(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	r.Header.Set("Accept", "text/html")
	if got := negotiateFormat(r); got != FormatJSON {
		t.Errorf("negotiateFormat(/api/...) = %q, want json", got)
	}
}

func TestNegotiateFormat_AcceptHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept", "application/json")
	if got := negotiateFormat(r); got != FormatJSON {
		t.Errorf("negotiateFormat(Accept json) = %q, want json", got)
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept", "text/plain")
	if got := negotiateFormat(r); got != FormatText {
		t.Errorf("negotiateFormat(Accept text/plain) = %q, want text", got)
	}
}

func TestNegotiateFormat_CLIClient(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("User-Agent", "curl/8.0")
	if got := negotiateFormat(r); got != FormatText {
		t.Errorf("negotiateFormat(curl) = %q, want text", got)
	}
}

func TestNegotiateFormat_DefaultHTML(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("User-Agent", "Mozilla/5.0")
	if got := negotiateFormat(r); got != FormatHTML {
		t.Errorf("negotiateFormat(browser) = %q, want html", got)
	}
}

func TestIsCLIClient(t *testing.T) {
	cases := map[string]bool{
		"curl/8.0":                 true,
		"Wget/1.21":                true,
		"python-requests/2.31":     true,
		"Mozilla/5.0 (Windows NT)": false,
		"":                         false,
	}
	for ua, want := range cases {
		if got := isCLIClient(ua); got != want {
			t.Errorf("isCLIClient(%q) = %v, want %v", ua, got, want)
		}
	}
}

func TestGetResponseFormat_Default(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := GetResponseFormat(r); got != FormatHTML {
		t.Errorf("GetResponseFormat(no context) = %q, want html (default)", got)
	}
}

func TestContentNegotiationMiddleware(t *testing.T) {
	var got ResponseFormat
	h := ContentNegotiationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetResponseFormat(r)
	}))
	r := httptest.NewRequest("GET", "/api/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got != FormatJSON {
		t.Errorf("format in context = %q, want json", got)
	}
	if w.Header().Get("Vary") != "Accept" {
		t.Error("expected Vary: Accept header")
	}
}

func TestPathSecurityMiddleware_BlocksTraversal(t *testing.T) {
	h := PathSecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not run for traversal attempt")
	}))
	r := httptest.NewRequest("GET", "/../etc/passwd", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPathSecurityMiddleware_NormalizesPath(t *testing.T) {
	var gotPath string
	h := PathSecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
	}))
	r := httptest.NewRequest("GET", "//foo//bar", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if gotPath != "/foo/bar" {
		t.Errorf("normalized path = %q, want /foo/bar", gotPath)
	}
}

func TestURLNormalizeMiddleware_RedirectsTrailingSlash(t *testing.T) {
	h := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not run before redirect")
	}))
	r := httptest.NewRequest("GET", "/foo/?x=1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/foo?x=1" {
		t.Errorf("Location = %q, want /foo?x=1", loc)
	}
}

func TestURLNormalizeMiddleware_RootPassesThrough(t *testing.T) {
	called := false
	h := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Error("expected handler to run for root path")
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	h := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	for _, hdr := range []string{"X-Content-Type-Options", "X-Frame-Options", "X-XSS-Protection", "Referrer-Policy", "Permissions-Policy", "Content-Security-Policy"} {
		if w.Header().Get(hdr) == "" {
			t.Errorf("missing header %s", hdr)
		}
	}
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should not be set for plain HTTP")
	}
}

func TestSecurityHeadersMiddleware_HSTSOnForwardedHTTPS(t *testing.T) {
	h := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("expected HSTS header when forwarded proto is https")
	}
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	var gotID string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r)
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if gotID == "" {
		t.Error("expected generated request ID")
	}
	if w.Header().Get("X-Request-ID") != gotID {
		t.Error("response header should match context request ID")
	}
}

func TestRequestIDMiddleware_UsesClientProvided(t *testing.T) {
	var gotID string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r)
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Request-ID", "client-supplied-id")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if gotID != "client-supplied-id" {
		t.Errorf("gotID = %q, want client-supplied-id", gotID)
	}
}

func TestGetRequestID_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := GetRequestID(r); got != "" {
		t.Errorf("GetRequestID(no context) = %q, want empty", got)
	}
}

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	called := false
	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("expected origin to be echoed back")
	}
	if !called {
		t.Error("expected downstream handler to run for non-OPTIONS request")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	called := false
	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	r := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if called {
		t.Error("downstream handler should not run for OPTIONS preflight")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	h := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestCacheControlMiddleware(t *testing.T) {
	h := CacheControlMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	r := httptest.NewRequest("GET", "/static/app.css", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Cache-Control") == "" {
		t.Error("expected Cache-Control for /static/ path")
	}

	r2 := httptest.NewRequest("GET", "/foo", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Header().Get("Cache-Control") != "" {
		t.Error("did not expect Cache-Control for non-static path")
	}
}

func TestRateLimiter_AllowAndDeny(t *testing.T) {
	rl := &RateLimiter{
		counters: make(map[string]*rateLimitEntry),
		limits: map[string]rateLimitConfig{
			"login":      {limit: 2, window: time.Minute},
			"api_unauth": {limit: 20, window: time.Minute},
		},
	}

	allowed, remaining, _ := rl.Allow("login", "1.2.3.4")
	if !allowed || remaining != 1 {
		t.Fatalf("first call: allowed=%v remaining=%d, want true 1", allowed, remaining)
	}

	allowed, remaining, _ = rl.Allow("login", "1.2.3.4")
	if !allowed || remaining != 0 {
		t.Fatalf("second call: allowed=%v remaining=%d, want true 0", allowed, remaining)
	}

	allowed, remaining, resetTime := rl.Allow("login", "1.2.3.4")
	if allowed {
		t.Fatal("third call should be denied")
	}
	if resetTime.Before(time.Now()) {
		t.Error("resetTime should be in the future")
	}
}

func TestRateLimiter_UnknownTypeFallsBackToUnauth(t *testing.T) {
	rl := &RateLimiter{
		counters: make(map[string]*rateLimitEntry),
		limits: map[string]rateLimitConfig{
			"api_unauth": {limit: 1, window: time.Minute},
		},
	}
	allowed, _, _ := rl.Allow("nonexistent_type", "id")
	if !allowed {
		t.Error("expected fallback to api_unauth limit to allow first request")
	}
}

func TestRateLimiter_GetKeyType(t *testing.T) {
	rl := &RateLimiter{}
	if got := rl.getKeyType("login:1.2.3.4"); got != "login" {
		t.Errorf("getKeyType = %q, want login", got)
	}
	if got := rl.getKeyType(""); got != "api_unauth" {
		t.Errorf("getKeyType(empty) = %q, want api_unauth", got)
	}
}

func TestGetLimitForType(t *testing.T) {
	rl := NewRateLimiter()
	if got := getLimitForType(rl, "login"); got != 5 {
		t.Errorf("getLimitForType(login) = %d, want 5", got)
	}
	if got := getLimitForType(rl, "nonexistent"); got != 20 {
		t.Errorf("getLimitForType(unknown) = %d, want fallback 20", got)
	}
}

func TestGetRateLimitKey(t *testing.T) {
	cases := []struct {
		path     string
		wantType string
	}{
		{"/auth/login", "login"},
		{"/auth/password/reset", "password_reset"},
		{"/auth/register", "registration"},
		{"/api/v1/search", "api_unauth"},
		{"/", "api_unauth"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", c.path, nil)
		gotType, _ := getRateLimitKey(r)
		if gotType != c.wantType {
			t.Errorf("getRateLimitKey(%q) type = %q, want %q", c.path, gotType, c.wantType)
		}
	}
}

func TestGetRateLimitKey_APIAuthWithSession(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/x", nil)
	r.AddCookie(&http.Cookie{Name: "casman_admin_session", Value: "tok"})
	gotType, _ := getRateLimitKey(r)
	if gotType != "api_auth" {
		t.Errorf("type = %q, want api_auth", gotType)
	}
}

func TestGetRateLimitKey_APIAuthWithHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/x", nil)
	r.Header.Set("Authorization", "Bearer tok")
	gotType, _ := getRateLimitKey(r)
	if gotType != "api_auth" {
		t.Errorf("type = %q, want api_auth", gotType)
	}
}

func TestGetRateLimitKey_UsesForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	_, identifier := getRateLimitKey(r)
	if identifier != "203.0.113.9" {
		t.Errorf("identifier = %q, want 203.0.113.9", identifier)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := &RateLimiter{
		counters: make(map[string]*rateLimitEntry),
		limits: map[string]rateLimitConfig{
			"api_unauth": {limit: 1, window: time.Minute},
		},
	}
	h := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "5.5.5.5:1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", w.Code)
	}

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", w2.Code)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header when rate limited")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:5555"
	if got := clientIP(r); got != "192.0.2.1" {
		t.Errorf("clientIP = %q, want 192.0.2.1", got)
	}
}

func TestIsPrivateAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1": true,
		"10.0.0.5":  true,
		"192.168.1.1": true,
		"8.8.8.8":   false,
		"not-an-ip": false,
	}
	for ip, want := range cases {
		if got := isPrivateAddr(ip); got != want {
			t.Errorf("isPrivateAddr(%q) = %v, want %v", ip, got, want)
		}
	}
}

func TestGetCountry_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := GetCountry(r); got != "" {
		t.Errorf("GetCountry(no context) = %q, want empty", got)
	}
}

func TestDefaultCSRFConfig(t *testing.T) {
	cfg := DefaultCSRFConfig()
	if cfg.TokenLength != 32 || cfg.CookieName != "csrf_token" || !cfg.Secure {
		t.Errorf("unexpected default config: %+v", cfg)
	}
}

func TestNewCSRF_NilConfigUsesDefault(t *testing.T) {
	c := NewCSRF(nil)
	if c.config.CookieName != "csrf_token" {
		t.Errorf("expected default config to be used")
	}
}

func TestCSRF_GenerateToken(t *testing.T) {
	c := NewCSRF(nil)
	tok, err := c.generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if tok == "" {
		t.Error("expected non-empty token")
	}
}

func TestCSRF_IsSafeMethod(t *testing.T) {
	c := NewCSRF(nil)
	for _, m := range []string{"GET", "HEAD", "OPTIONS", "TRACE"} {
		if !c.isSafeMethod(m) {
			t.Errorf("isSafeMethod(%s) = false, want true", m)
		}
	}
	if c.isSafeMethod("POST") {
		t.Error("isSafeMethod(POST) = true, want false")
	}
}

func TestCSRFMiddleware_SkipsAPIRoutes(t *testing.T) {
	called := false
	h := CSRFMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	r := httptest.NewRequest("POST", "/api/v1/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Error("expected API routes to bypass CSRF checks")
	}
}

func TestCSRFMiddleware_SetsTokenCookieOnGET(t *testing.T) {
	h := CSRFMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetCSRFToken(r) == "" {
			t.Error("expected token in context")
		}
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "csrf_token" {
			found = true
		}
	}
	if !found {
		t.Error("expected csrf_token cookie to be set")
	}
}

func TestCSRFMiddleware_RejectsMissingTokenOnPOST(t *testing.T) {
	h := CSRFMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not run without valid CSRF token")
	}))
	r := httptest.NewRequest("POST", "/submit", strings.NewReader(""))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestCSRFMiddleware_AcceptsValidHeaderToken(t *testing.T) {
	c := NewCSRF(nil)
	tok, _ := c.generateToken()

	called := false
	h := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	r := httptest.NewRequest("POST", "/submit", nil)
	r.AddCookie(&http.Cookie{Name: c.config.CookieName, Value: tok})
	r.Header.Set(c.config.HeaderName, tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to run with valid CSRF token")
	}
	if w.Code != http.StatusOK && w.Code != 0 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCSRFMiddleware_AcceptsValidFormToken(t *testing.T) {
	c := NewCSRF(nil)
	tok, _ := c.generateToken()

	called := false
	h := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	form := c.config.FormField + "=" + tok
	r := httptest.NewRequest("POST", "/submit", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: c.config.CookieName, Value: tok})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to run with valid form CSRF token")
	}
}

func TestGetCSRFToken_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := GetCSRFToken(r); got != "" {
		t.Errorf("GetCSRFToken(no context) = %q, want empty", got)
	}
}

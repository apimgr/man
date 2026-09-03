// Package server provides the HTTP server for casman.
package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// ResponseFormat indicates the preferred response format.
type ResponseFormat string

const (
	FormatHTML ResponseFormat = "html"
	FormatJSON ResponseFormat = "json"
	FormatText ResponseFormat = "text"
)

// contextKey is a private type for context keys.
type contextKey string

const responseFormatKey contextKey = "responseFormat"

// GetResponseFormat retrieves the preferred response format from context.
func GetResponseFormat(r *http.Request) ResponseFormat {
	if format, ok := r.Context().Value(responseFormatKey).(ResponseFormat); ok {
		return format
	}
	return FormatHTML
}

// ContentNegotiationMiddleware determines response format from Accept header.
// Per PART 14: HTML for browsers, text for CLI, JSON for API clients.
func ContentNegotiationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Vary header for content negotiation caching per AI.md PART 14
		w.Header().Set("Vary", "Accept")

		format := negotiateFormat(r)
		ctx := context.WithValue(r.Context(), responseFormatKey, format)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// negotiateFormat determines the preferred response format.
func negotiateFormat(r *http.Request) ResponseFormat {
	accept := r.Header.Get("Accept")
	userAgent := r.Header.Get("User-Agent")

	// Check for explicit format query parameter
	if q := r.URL.Query().Get("format"); q != "" {
		switch strings.ToLower(q) {
		case "json":
			return FormatJSON
		case "text", "txt", "plain":
			return FormatText
		case "html":
			return FormatHTML
		}
	}

	// API paths always return JSON
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return FormatJSON
	}

	// Check Accept header
	accept = strings.ToLower(accept)

	// Explicit JSON request
	if strings.Contains(accept, "application/json") {
		return FormatJSON
	}

	// Explicit text request
	if strings.Contains(accept, "text/plain") {
		return FormatText
	}

	// CLI tools (curl, wget, httpie) without browser user-agent
	if isCLIClient(userAgent) && !strings.Contains(accept, "text/html") {
		return FormatText
	}

	// Default to HTML for browsers
	return FormatHTML
}

// isCLIClient checks if the User-Agent indicates a CLI tool.
func isCLIClient(ua string) bool {
	ua = strings.ToLower(ua)
	cliTools := []string{
		"curl", "wget", "httpie", "fetch", "aria2",
		"libwww-perl", "python-requests", "go-http-client",
		"java/", "node-fetch", "axios",
	}
	for _, tool := range cliTools {
		if strings.Contains(ua, tool) {
			return true
		}
	}
	return false
}

// PathSecurityMiddleware normalizes paths and blocks traversal attempts.
// This middleware MUST be first in the chain - before auth, before routing.
// See AI.md PART 5 for details.
func PathSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		original := r.URL.Path

		// Check both raw path and URL-decoded for traversal
		rawPath := r.URL.RawPath
		if rawPath == "" {
			rawPath = r.URL.Path
		}

		// Block path traversal attempts (encoded and decoded)
		if strings.Contains(original, "..") ||
			strings.Contains(rawPath, "..") ||
			strings.Contains(strings.ToLower(rawPath), "%2e") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Normalize the path
		cleaned := path.Clean(original)

		// Ensure leading slash
		if !strings.HasPrefix(cleaned, "/") {
			cleaned = "/" + cleaned
		}

		// Update request
		r.URL.Path = cleaned

		next.ServeHTTP(w, r)
	})
}

// URLNormalizeMiddleware redirects trailing slashes to canonical URLs.
// Per PART 14: Routes MUST NOT end with `/` (except root).
// Redirect with 301 to canonical URL without trailing slash.
func URLNormalizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path

		// Skip root path
		if p == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Redirect trailing slashes with 301
		if len(p) > 1 && strings.HasSuffix(p, "/") {
			// Build canonical URL without trailing slash
			canonical := strings.TrimSuffix(p, "/")
			if r.URL.RawQuery != "" {
				canonical += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, canonical, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses.
// See AI.md PART 11 for details.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers per AI.md PART 11
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy per AI.md
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content Security Policy (relaxed for development)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' unpkg.com; style-src 'self' 'unsafe-inline' unpkg.com; img-src 'self' data:; font-src 'self' data:; connect-src 'self'")

		// HSTS header when using HTTPS per AI.md PART 11
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware handles X-Request-ID headers per AI.md PART 11.
// Accepts client-provided request ID or generates one.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing request ID (priority order per AI.md)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = r.Header.Get("X-Correlation-ID")
		}
		if requestID == "" {
			requestID = r.Header.Get("X-Trace-ID")
		}

		// Generate if not provided
		if requestID == "" {
			// Generate a simple request ID
			b := make([]byte, 16)
			rand.Read(b)
			requestID = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		}

		// Set in response header
		w.Header().Set("X-Request-ID", requestID)

		// Store in context for logging
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestIDKey is the context key for request ID.
const requestIDKey contextKey = "requestID"

// GetRequestID retrieves the request ID from context.
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// CORSMiddleware handles CORS for API requests.
// See AI.md PART 14 for details.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow all origins for API
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Requested-With")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and returns 500.
// See AI.md PART 6 for details.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CacheControlMiddleware sets cache headers for static assets.
func CacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers for static assets
		if strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimiter implements in-memory rate limiting per AI.md PART 11.
type RateLimiter struct {
	mu       sync.RWMutex
	counters map[string]*rateLimitEntry
	limits   map[string]rateLimitConfig
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type rateLimitConfig struct {
	limit  int
	window time.Duration
}

// NewRateLimiter creates a new rate limiter with default limits per AI.md.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		counters: make(map[string]*rateLimitEntry),
		limits: map[string]rateLimitConfig{
			// Per AI.md PART 11 defaults
			"login":          {limit: 5, window: 15 * time.Minute},
			"password_reset": {limit: 3, window: time.Hour},
			"api_auth":       {limit: 100, window: time.Minute},
			"api_unauth":     {limit: 20, window: time.Minute},
			"registration":   {limit: 5, window: time.Hour},
			"upload":         {limit: 10, window: time.Hour},
		},
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup periodically removes expired rate limit entries.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.counters {
			// Find the limit for this key type
			keyType := rl.getKeyType(key)
			cfg, ok := rl.limits[keyType]
			if !ok {
				cfg = rl.limits["api_unauth"]
			}
			if now.Sub(entry.windowStart) > cfg.window {
				delete(rl.counters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getKeyType(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return "api_unauth"
}

// Allow checks if a request should be allowed.
// Returns (allowed, remaining, resetTime).
func (rl *RateLimiter) Allow(limitType, identifier string) (bool, int, time.Time) {
	key := fmt.Sprintf("%s:%s", limitType, identifier)

	cfg, ok := rl.limits[limitType]
	if !ok {
		cfg = rl.limits["api_unauth"]
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.counters[key]

	if !exists || now.Sub(entry.windowStart) > cfg.window {
		// New window
		rl.counters[key] = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true, cfg.limit - 1, now.Add(cfg.window)
	}

	if entry.count >= cfg.limit {
		resetTime := entry.windowStart.Add(cfg.window)
		return false, 0, resetTime
	}

	entry.count++
	remaining := cfg.limit - entry.count
	return true, remaining, entry.windowStart.Add(cfg.window)
}

// geoCountryKey carries the resolved ISO 3166-1 alpha-2 country code through
// the request context so downstream handlers (audit log, admin views) can
// read it without re-running the lookup.
const geoCountryKey contextKey = "geoCountry"

// GetCountry returns the country code resolved by GeoIPMiddleware, or "" if
// the middleware did not run for this request (geoip disabled, healthz path,
// private IP, or lookup miss).
func GetCountry(r *http.Request) string {
	if v, ok := r.Context().Value(geoCountryKey).(string); ok {
		return v
	}
	return ""
}

// geoIPSource is the subset of *geoip.GeoIP the middleware needs. Defined
// as an interface so tests can stub it without importing the geoip package.
type geoIPSource interface {
	IsAvailable() bool
	GetCountry(ip string) string
	IsBlocked(ip string) bool
}

// GeoIPMiddleware enforces deny_countries (AI.md PART 20) and stashes the
// resolved country code in the request context for audit logging. The
// lookup is skipped for the /healthz, /metrics, /static, and well-known
// paths so monitoring and ACME challenges are never blocked, and for
// private/loopback addresses where geoip cannot return a country anyway.
func GeoIPMiddleware(g geoIPSource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if g == nil || !g.IsAvailable() {
				next.ServeHTTP(w, r)
				return
			}
			if skipGeoIPPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r)
			if ip == "" || isPrivateAddr(ip) {
				next.ServeHTTP(w, r)
				return
			}
			if g.IsBlocked(ip) {
				http.Error(w, "Forbidden: access denied for your country", http.StatusForbidden)
				return
			}
			country := g.GetCountry(ip)
			if country != "" {
				ctx := context.WithValue(r.Context(), geoCountryKey, country)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func skipGeoIPPath(p string) bool {
	if p == "/healthz" || p == "/metrics" || p == "/robots.txt" {
		return true
	}
	if strings.HasPrefix(p, "/static/") {
		return true
	}
	if strings.HasPrefix(p, "/.well-known/") {
		return true
	}
	return false
}

func clientIP(r *http.Request) string {
	// chi's middleware.RealIP rewrote RemoteAddr to the trusted forwarded
	// IP earlier in the chain when X-Forwarded-For was present.
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host
}

func isPrivateAddr(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// RateLimitMiddleware applies rate limiting to requests.
// Per AI.md PART 11: Rate limiting on sensitive endpoints.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine limit type and identifier
			limitType, identifier := getRateLimitKey(r)

			// Check rate limit
			allowed, remaining, resetTime := rl.Allow(limitType, identifier)

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", getLimitForType(rl, limitType)))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getLimitForType(rl *RateLimiter, limitType string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	if cfg, ok := rl.limits[limitType]; ok {
		return cfg.limit
	}
	return 20
}

// getRateLimitKey determines the rate limit type and identifier for a request.
func getRateLimitKey(r *http.Request) (limitType, identifier string) {
	path := r.URL.Path

	// Get client IP
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}

	// Determine limit type based on path
	switch {
	case strings.HasPrefix(path, "/auth/login"):
		return "login", ip
	case strings.HasPrefix(path, "/auth/password"):
		return "password_reset", ip
	case strings.HasPrefix(path, "/auth/register"):
		return "registration", ip
	case strings.HasPrefix(path, "/api/"):
		// Check if authenticated (has valid session cookie or API key)
		if _, err := r.Cookie("casman_admin_session"); err == nil {
			return "api_auth", ip
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			return "api_auth", ip
		}
		return "api_unauth", ip
	default:
		return "api_unauth", ip
	}
}

// CSRF protection per AI.md PART 11

// CSRFConfig holds CSRF configuration.
type CSRFConfig struct {
	TokenLength int
	CookieName  string
	HeaderName  string
	FormField   string
	Secure      bool
}

// DefaultCSRFConfig returns the default CSRF configuration.
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength: 32,
		CookieName:  "csrf_token",
		HeaderName:  "X-CSRF-Token",
		FormField:   "_csrf",
		Secure:      true,
	}
}

// CSRF provides CSRF protection middleware.
type CSRF struct {
	config *CSRFConfig
}

// NewCSRF creates a new CSRF protection instance.
func NewCSRF(cfg *CSRFConfig) *CSRF {
	if cfg == nil {
		cfg = DefaultCSRFConfig()
	}
	return &CSRF{config: cfg}
}

// generateCSRFToken creates a new CSRF token.
func (c *CSRF) generateToken() (string, error) {
	bytes := make([]byte, c.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Middleware returns the CSRF middleware handler.
// Per AI.md: All forms must have CSRF tokens, all non-GET requests validate.
func (c *CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for API routes (they use tokens/sessions)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Get or create CSRF token
		token := c.getTokenFromCookie(r)
		if token == "" {
			var err error
			token, err = c.generateToken()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			c.setTokenCookie(w, r, token)
		}

		// Store token in context for templates
		ctx := context.WithValue(r.Context(), csrfTokenKey, token)

		// For non-safe methods, validate CSRF token
		if !c.isSafeMethod(r.Method) {
			if !c.validateToken(r, token) {
				http.Error(w, "Forbidden - Invalid CSRF Token", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c *CSRF) isSafeMethod(method string) bool {
	return method == "GET" || method == "HEAD" || method == "OPTIONS" || method == "TRACE"
}

func (c *CSRF) getTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(c.config.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (c *CSRF) setTokenCookie(w http.ResponseWriter, r *http.Request, token string) {
	// Auto-detect secure flag per AI.md PART 11
	// Secure=true for HTTPS, false for HTTP
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     c.config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Accessible from JavaScript for AJAX
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   86400, // 24 hours
	})
}

func (c *CSRF) validateToken(r *http.Request, expectedToken string) bool {
	// Check header first
	headerToken := r.Header.Get(c.config.HeaderName)
	if headerToken != "" && headerToken == expectedToken {
		return true
	}

	// Check form field
	if r.Method == "POST" {
		if err := r.ParseForm(); err == nil {
			formToken := r.FormValue(c.config.FormField)
			if formToken != "" && formToken == expectedToken {
				return true
			}
		}
	}

	return false
}

// Context key for CSRF token
const csrfTokenKey contextKey = "csrfToken"

// GetCSRFToken retrieves the CSRF token from the request context.
func GetCSRFToken(r *http.Request) string {
	if token, ok := r.Context().Value(csrfTokenKey).(string); ok {
		return token
	}
	return ""
}

// CSRFMiddleware creates a CSRF middleware with default config.
func CSRFMiddleware() func(http.Handler) http.Handler {
	csrf := NewCSRF(nil)
	return csrf.Middleware
}

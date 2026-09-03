// Package ssl provides SSL/TLS and Let's Encrypt support for casman.
// See AI.md PART 15 for details.
package ssl

import (
	"crypto/tls"
	"net"
	"os"
	"strings"
)

// ChallengeType represents an ACME challenge type.
type ChallengeType string

const (
	// HTTP01 is HTTP-based challenge (default, requires port 80).
	HTTP01 ChallengeType = "http-01"
	// TLSALPN01 is TLS-based challenge (requires port 443).
	TLSALPN01 ChallengeType = "tls-alpn-01"
	// DNS01 is DNS TXT record challenge (wildcard certs, no port requirements).
	DNS01 ChallengeType = "dns-01"
)

// Config holds SSL/TLS configuration.
type Config struct {
	Enabled       bool          `yaml:"enabled"`
	CertFile      string        `yaml:"cert_file"`
	KeyFile       string        `yaml:"key_file"`
	LetsEncrypt   bool          `yaml:"letsencrypt"`
	ChallengeType ChallengeType `yaml:"challenge_type"`
	Email         string        `yaml:"email"`
	DNSProvider   string        `yaml:"dns_provider"`
}

// GetHostFromRequest resolves host from HTTP request headers.
// Use this for request-time host resolution (preferred).
// See AI.md PART 15 for details.
func GetHostFromRequest(headers map[string]string, projectName string) string {
	// 1. Reverse proxy headers (highest priority - we prefer to be behind a proxy)
	for _, header := range []string{"X-Forwarded-Host", "X-Real-Host", "X-Original-Host"} {
		if host, ok := headers[header]; ok && host != "" {
			// Strip port if present (we handle port separately)
			if h, _, err := net.SplitHostPort(host); err == nil {
				return h
			}
			return host
		}
	}

	// 2. Fall back to static resolution
	return GetFQDN(projectName)
}

// GetFQDN resolves the FQDN when no request context is available.
// Returns first domain from DOMAIN env var (comma-separated list supported).
// See AI.md PART 15 for details.
func GetFQDN(projectName string) string {
	// 1. DOMAIN env var (explicit user override, comma-separated)
	if domain := os.Getenv("DOMAIN"); domain != "" {
		// Return first domain as primary
		if idx := strings.Index(domain, ","); idx > 0 {
			return strings.TrimSpace(domain[:idx])
		}
		return domain
	}

	// 2. os.Hostname() - cross-platform (Linux, macOS, Windows, BSD)
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		if !isLoopback(hostname) {
			return hostname
		}
	}

	// 3. $HOSTNAME env var (skip loopback)
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		if !isLoopback(hostname) {
			return hostname
		}
	}

	// 4. Global IPv6 (preferred for modern networks)
	if ipv6 := getGlobalIPv6(); ipv6 != "" {
		return ipv6
	}

	// 5. Global IPv4
	if ipv4 := getGlobalIPv4(); ipv4 != "" {
		return ipv4
	}

	// Last resort (not recommended)
	return "localhost"
}

// isLoopback checks if a host is loopback.
func isLoopback(host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// getGlobalIPv6 returns first public IPv6 address.
// Excludes: loopback (::1), link-local (fe80::/10), unique local (fc00::/7).
func getGlobalIPv6() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP
			// Must be IPv6 (not IPv4), globally routable, and not private
			if ip.To4() == nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return ""
}

// getGlobalIPv4 returns first public IPv4 address.
// Excludes: loopback (127.0.0.0/8), private (10/8, 172.16/12, 192.168/16), link-local (169.254/16).
func getGlobalIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP
			// Must be IPv4, globally routable, and not private
			if ip4 := ip.To4(); ip4 != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
				return ip4.String()
			}
		}
	}
	return ""
}

// IsPublicIP checks if an IP is publicly routable.
func IsPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

// GetAllDomains returns all domains from DOMAIN env var.
// Used for CORS configuration and SSL certificates.
func GetAllDomains() []string {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		return nil
	}
	parts := strings.Split(domain, ",")
	domains := make([]string, 0, len(parts))
	for _, p := range parts {
		if d := strings.TrimSpace(p); d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}

// DefaultTLSConfig returns a secure TLS configuration.
func DefaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}
}

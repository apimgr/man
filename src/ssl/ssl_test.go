package ssl

import (
	"net"
	"os"
	"testing"
)

func TestGetHostFromRequest_ForwardedHost(t *testing.T) {
	headers := map[string]string{"X-Forwarded-Host": "example.com:8443"}
	if got := GetHostFromRequest(headers, "casman"); got != "example.com" {
		t.Errorf("got %q, want example.com", got)
	}
}

func TestGetHostFromRequest_NoPort(t *testing.T) {
	headers := map[string]string{"X-Real-Host": "example.com"}
	if got := GetHostFromRequest(headers, "casman"); got != "example.com" {
		t.Errorf("got %q, want example.com", got)
	}
}

func TestGetHostFromRequest_FallsBackToFQDN(t *testing.T) {
	t.Setenv("DOMAIN", "fallback.example.com")
	got := GetHostFromRequest(map[string]string{}, "casman")
	if got != "fallback.example.com" {
		t.Errorf("got %q, want fallback.example.com", got)
	}
}

func TestGetFQDN_DomainEnv(t *testing.T) {
	t.Setenv("DOMAIN", "a.example.com,b.example.com")
	if got := GetFQDN("casman"); got != "a.example.com" {
		t.Errorf("got %q, want a.example.com", got)
	}
}

func TestGetFQDN_DomainEnvSingle(t *testing.T) {
	t.Setenv("DOMAIN", "single.example.com")
	if got := GetFQDN("casman"); got != "single.example.com" {
		t.Errorf("got %q, want single.example.com", got)
	}
}

func TestGetFQDN_NoDomainStillResolves(t *testing.T) {
	os.Unsetenv("DOMAIN")
	got := GetFQDN("casman")
	if got == "" {
		t.Error("GetFQDN returned empty string")
	}
}

func TestIsLoopback(t *testing.T) {
	cases := map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
		"example.com": false,
		"8.8.8.8":   false,
	}
	for host, want := range cases {
		if got := isLoopback(host); got != want {
			t.Errorf("isLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"192.168.1.1", false},
		{"169.254.1.1", false},
		{"::1", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if got := IsPublicIP(ip); got != c.want {
			t.Errorf("IsPublicIP(%q) = %v, want %v", c.ip, got, c.want)
		}
	}
}

func TestGetAllDomains_Empty(t *testing.T) {
	os.Unsetenv("DOMAIN")
	if got := GetAllDomains(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestGetAllDomains_Multiple(t *testing.T) {
	t.Setenv("DOMAIN", "a.example.com, b.example.com,c.example.com")
	got := GetAllDomains()
	want := []string{"a.example.com", "b.example.com", "c.example.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultTLSConfig(t *testing.T) {
	cfg := DefaultTLSConfig()
	if cfg == nil {
		t.Fatal("DefaultTLSConfig returned nil")
	}
	if len(cfg.CipherSuites) == 0 {
		t.Error("expected non-empty CipherSuites")
	}
	if len(cfg.CurvePreferences) == 0 {
		t.Error("expected non-empty CurvePreferences")
	}
}

func TestGetGlobalIPv6AndIPv4_NoPanic(t *testing.T) {
	// These depend on the host's network interfaces; just ensure they
	// run without panicking and return a valid (possibly empty) string.
	_ = getGlobalIPv6()
	_ = getGlobalIPv4()
}

package smtp

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 587 {
		t.Errorf("Port = %d, want 587", cfg.Port)
	}
	if cfg.TLS != "auto" {
		t.Errorf("TLS = %q, want auto", cfg.TLS)
	}
}

func TestNew(t *testing.T) {
	cfg := Config{Host: "mail.example.com"}
	c := New(cfg)
	if c.GetConfig().Host != "mail.example.com" {
		t.Errorf("Host = %q", c.GetConfig().Host)
	}
	if c.IsAvailable() {
		t.Error("new client should not be available")
	}
}

func TestSetConfig(t *testing.T) {
	c := New(DefaultConfig())
	c.detected = true
	c.SetConfig(Config{Host: "smtp.example.com", Port: 25})
	if c.GetConfig().Host != "smtp.example.com" {
		t.Errorf("Host not updated: %q", c.GetConfig().Host)
	}
	if c.detected {
		t.Error("SetConfig should reset detected flag")
	}
}

func TestTestConnection_NoHost(t *testing.T) {
	c := New(DefaultConfig())
	if c.TestConnection() {
		t.Error("expected false when host is empty")
	}
}

func TestTestConnection_UnreachableHost(t *testing.T) {
	c := New(Config{Host: "127.0.0.1", Port: 1})
	if c.TestConnection() {
		t.Error("expected false for unreachable host:port")
	}
	if c.IsAvailable() {
		t.Error("client should not be available after failed test")
	}
}

func TestAutoDetect_DoesNotPanicAndSetsConfigOnSuccess(t *testing.T) {
	c := New(DefaultConfig())
	host, port, ok := c.AutoDetect("")
	if ok {
		// Some environments (e.g. a Docker bridge gateway with a mail relay)
		// may have a reachable SMTP server; just check the client state is
		// consistent with a successful detection.
		if c.GetConfig().Host != host || c.GetConfig().Port != port {
			t.Errorf("client config %s:%d does not match detected %s:%d", c.GetConfig().Host, c.GetConfig().Port, host, port)
		}
		if !c.IsAvailable() {
			t.Error("expected available true after successful auto-detect")
		}
	}
}

func TestSend_NotAvailable(t *testing.T) {
	c := New(DefaultConfig())
	if err := c.Send("to@example.com", "subject", "body"); err == nil {
		t.Error("expected error when not available")
	}
}

func TestSend_NotConfigured(t *testing.T) {
	c := New(DefaultConfig())
	c.available = true
	if err := c.Send("to@example.com", "subject", "body"); err == nil {
		t.Error("expected error when host is empty")
	}
}

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("from@example.com", "to@example.com", "Hello", "Body text")
	s := string(msg)
	if !strings.Contains(s, "From: from@example.com") {
		t.Error("missing From header")
	}
	if !strings.Contains(s, "To: to@example.com") {
		t.Error("missing To header")
	}
	if !strings.Contains(s, "Subject: Hello") {
		t.Error("missing Subject header")
	}
	if !strings.Contains(s, "\r\n\r\nBody text") {
		t.Error("missing body after headers")
	}
}

func TestGetDefaultGateway(t *testing.T) {
	if getDefaultGateway() != "" {
		t.Error("expected empty string stub")
	}
}

func TestGetGlobalIPv4_NoPanic(t *testing.T) {
	_ = getGlobalIPv4()
}

func TestIsPrivateIP(t *testing.T) {
	cases := map[string]bool{
		"10.0.0.1":     true,
		"172.16.0.1":   true,
		"172.31.255.1": true,
		"192.168.1.1":  true,
		"8.8.8.8":       false,
		"1.1.1.1":       false,
		"172.32.0.1":    false,
	}
	for ip, want := range cases {
		if got := isPrivateIP(ip); got != want {
			t.Errorf("isPrivateIP(%q) = %v, want %v", ip, got, want)
		}
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SMTP_HOST", "mail.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USERNAME", "user")
	t.Setenv("SMTP_PASSWORD", "pass")
	t.Setenv("SMTP_TLS", "starttls")
	t.Setenv("SMTP_FROM_NAME", "Example")
	t.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")

	cfg := DefaultConfig()
	LoadFromEnv(&cfg)

	if cfg.Host != "mail.example.com" {
		t.Errorf("Host = %q", cfg.Host)
	}
	if cfg.Port != 2525 {
		t.Errorf("Port = %d", cfg.Port)
	}
	if cfg.Username != "user" || cfg.Password != "pass" {
		t.Error("username/password not loaded")
	}
	if cfg.TLS != "starttls" {
		t.Errorf("TLS = %q", cfg.TLS)
	}
	if cfg.FromName != "Example" || cfg.FromAddr != "noreply@example.com" {
		t.Error("from name/email not loaded")
	}
}

func TestLoadFromEnv_InvalidPortIgnored(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "not-a-number")
	cfg := DefaultConfig()
	LoadFromEnv(&cfg)
	if cfg.Port != 587 {
		t.Errorf("Port should remain default when invalid, got %d", cfg.Port)
	}
}

func TestLoadFromEnv_EmptyLeavesDefaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	cfg := DefaultConfig()
	LoadFromEnv(&cfg)
	if cfg != DefaultConfig() {
		t.Errorf("expected unchanged default config, got %+v", cfg)
	}
}

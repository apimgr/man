package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir, dir)
	if err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	cfg.Server.Port = "8080"
	cfg.Server.HTTPSPort = "8443"
	cfg.Server.SSL.Enabled = true
	cfg.Server.SSL.LetsEncrypt.Email = "ops@example.com"
	cfg.Server.SSL.LetsEncrypt.Challenge = "dns-01"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "server.yml"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("perm = %o, want 0600", perm)
	}

	got, err := Load(dir, dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Server.Port != "8080" {
		t.Errorf("Port = %q, want 8080", got.Server.Port)
	}
	if got.Server.HTTPSPort != "8443" {
		t.Errorf("HTTPSPort = %q, want 8443", got.Server.HTTPSPort)
	}
	if !got.Server.SSL.Enabled {
		t.Error("SSL.Enabled should round-trip true")
	}
	if got.Server.SSL.LetsEncrypt.Email != "ops@example.com" {
		t.Errorf("LE email = %q", got.Server.SSL.LetsEncrypt.Email)
	}
	if got.Server.SSL.LetsEncrypt.Challenge != "dns-01" {
		t.Errorf("LE challenge = %q", got.Server.SSL.LetsEncrypt.Challenge)
	}
}

func TestSave_RejectsEmptyConfigDir(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Save(); err == nil {
		t.Error("expected error when ConfigDir is empty")
	}
}

func TestDefaultConfig_HasHTTPSPorts(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir, dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.HTTPSPort != "443" {
		t.Errorf("default HTTPSPort = %q, want 443", cfg.Server.HTTPSPort)
	}
	if cfg.Server.HTTPRedirectPort != "80" {
		t.Errorf("default HTTPRedirectPort = %q, want 80", cfg.Server.HTTPRedirectPort)
	}
}

package server

import (
	"testing"
	"time"

	"github.com/casapps/casman/src/config"
)

func TestParsePort(t *testing.T) {
	cases := map[string]int{
		"":      0,
		"80":    80,
		"8443":  8443,
		"0":     0,
		"abc":   0,
		"12a3":  0,
		"65535": 65535,
	}
	for in, want := range cases {
		if got := parsePort(in); got != want {
			t.Errorf("parsePort(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestTorAvailable_NoTorService(t *testing.T) {
	s := &Server{}
	if s.TorAvailable() {
		t.Error("TorAvailable should be false when tor is nil")
	}
}

func TestTorRunning_NoTorService(t *testing.T) {
	s := &Server{}
	if s.TorRunning() {
		t.Error("TorRunning should be false when tor is nil")
	}
}

func TestTorOnionAddress_NoTorService(t *testing.T) {
	s := &Server{}
	if s.TorOnionAddress() != "" {
		t.Error("TorOnionAddress should be empty when tor is nil")
	}
}

func TestOutboundHTTPClient_NoTor(t *testing.T) {
	s := &Server{}
	c := s.OutboundHTTPClient(5 * time.Second)
	if c == nil {
		t.Fatal("OutboundHTTPClient returned nil")
	}
	if c.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", c.Timeout)
	}
}

func TestOutboundHTTPClient_TorConfiguredButNotRunning(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Tor = &config.TorConfig{UseNetwork: true}
	s := &Server{cfg: cfg}
	c := s.OutboundHTTPClient(0)
	if c == nil {
		t.Fatal("OutboundHTTPClient returned nil")
	}
}

func TestSSLDNSProvider_NoVault(t *testing.T) {
	s := &Server{}
	if got := s.sslDNSProvider(); got != "" {
		t.Errorf("sslDNSProvider = %q, want empty", got)
	}
}

func TestGetRandomPort(t *testing.T) {
	s := &Server{}
	port := s.getRandomPort()
	if port == "" {
		t.Error("getRandomPort returned empty string")
	}
}

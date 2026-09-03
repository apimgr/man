package tor

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.VirtualPort != 80 {
		t.Errorf("VirtualPort = %d, want 80", c.VirtualPort)
	}
	if c.BootstrapTimeout != 3*time.Minute {
		t.Errorf("BootstrapTimeout = %v", c.BootstrapTimeout)
	}
	if !c.SafeLogging {
		t.Error("SafeLogging should default true per AI.md PART 32")
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	s := New(Config{})
	if s.cfg.VirtualPort != 80 {
		t.Errorf("VirtualPort default not applied")
	}
	if s.cfg.BootstrapTimeout == 0 {
		t.Errorf("BootstrapTimeout default not applied")
	}
}

func TestStart_NoOpWhenBinaryMissing(t *testing.T) {
	// With Binary unset and no `tor` on PATH inside the test container,
	// Start must be a successful no-op per PART 32.
	if DetectBinary() != "" {
		t.Skip("tor binary present on PATH; cannot exercise the no-op path")
	}
	s := New(Config{LocalPort: 12345})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start should be a no-op when binary is missing, got %v", err)
	}
	if s.IsRunning() {
		t.Error("IsRunning should be false when no binary")
	}
	if s.OnionAddress() != "" {
		t.Error("OnionAddress should be empty when no binary")
	}
}

func TestAvailable_FalseWhenBinaryMissing(t *testing.T) {
	if DetectBinary() != "" {
		t.Skip("tor binary present on PATH; cannot exercise the unavailable path")
	}
	s := New(Config{})
	if s.Available() {
		t.Error("Available should be false when binary is missing from PATH")
	}
}

func TestStart_RequiresLocalPort(t *testing.T) {
	s := New(Config{})
	if err := s.Start(context.Background()); err == nil {
		t.Error("Start should reject missing LocalPort")
	}
}

func TestStop_NoOpWhenNotRunning(t *testing.T) {
	s := New(DefaultConfig())
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("Stop on idle service should be a no-op, got %v", err)
	}
}

func TestNilReceivers(t *testing.T) {
	var s *Service
	if s.Available() {
		t.Error("nil Service should report unavailable")
	}
	if s.IsRunning() {
		t.Error("nil Service should report not running")
	}
	if s.OnionAddress() != "" {
		t.Error("nil Service should return empty onion")
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("Stop on nil service should be safe, got %v", err)
	}
}

func TestOutbound_RequiresRunningService(t *testing.T) {
	s := New(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := s.Dial(ctx, "tcp", "example.com:80"); err == nil {
		t.Error("Dial should error when service is not running")
	}
	if _, err := s.HTTPTransport(); err == nil {
		t.Error("HTTPTransport should error when service is not running")
	}
	if _, err := s.HTTPClient(0); err == nil {
		t.Error("HTTPClient should error when service is not running")
	}
}

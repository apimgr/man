// Package tor wraps github.com/cretz/bine to expose the running server as a
// Tor hidden service. See AI.md PART 32.
//
// The package is deliberately minimal in this pass: it covers the
// non-negotiable "always enabled if a tor binary is found" requirement —
// auto-detection, v3 onion creation, lifecycle management, and onion-address
// reporting. Outbound routing through the Tor network, per-user preferences,
// bandwidth accounting, and the admin configuration form are tracked
// separately as follow-up work in TODO.AI.md.
package tor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/cretz/bine/tor"
)

// Config holds runtime knobs for the hidden service.
type Config struct {
	// Binary is the absolute path to a tor executable. Empty means
	// "auto-detect via PATH".
	Binary string
	// DataDir is where bine writes the Tor data directory. Empty means
	// "let bine choose" (a temp dir is the default).
	DataDir string
	// VirtualPort is the public port advertised on the .onion (default 80).
	VirtualPort int
	// LocalPort is the actual server listen port; bine forwards onion:80 to
	// localhost:LocalPort.
	LocalPort int
	// BootstrapTimeout caps how long Start() waits for Tor to bootstrap.
	BootstrapTimeout time.Duration
	// SafeLogging mirrors PART 32 — when true, Tor scrubs sensitive info.
	SafeLogging bool
}

// DefaultConfig returns the PART 32 default values.
func DefaultConfig() Config {
	return Config{
		VirtualPort:      80,
		BootstrapTimeout: 3 * time.Minute,
		SafeLogging:      true,
	}
}

// Service controls a hidden-service lifecycle.
type Service struct {
	cfg Config

	mu    sync.RWMutex
	t     *tor.Tor
	onion *tor.OnionService
	addr  string
	up    bool
}

// New builds a Service. The returned value is always non-nil; callers check
// Available()/IsRunning() to gate behavior.
func New(cfg Config) *Service {
	if cfg.VirtualPort == 0 {
		cfg.VirtualPort = 80
	}
	if cfg.BootstrapTimeout == 0 {
		cfg.BootstrapTimeout = 3 * time.Minute
	}
	return &Service{cfg: cfg}
}

// SetLocalPort updates the local port the hidden service forwards to. Must
// be called before Start. The server uses this to defer the decision until
// after random-port allocation in Run().
func (s *Service) SetLocalPort(port int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.LocalPort = port
}

// DetectBinary returns the resolved tor binary path or "" if it is not on
// PATH. Callers can use the empty return as the "skip silently" signal per
// PART 32 ("auto-enabled if tor binary is installed").
func DetectBinary() string {
	p, err := exec.LookPath("tor")
	if err != nil {
		return ""
	}
	return p
}

// Available reports whether a tor binary is reachable at the configured
// path. Used by /healthz and the admin UI.
func (s *Service) Available() bool {
	if s == nil {
		return false
	}
	bin := s.cfg.Binary
	if bin == "" {
		bin = DetectBinary()
	}
	return bin != ""
}

// Start launches Tor, waits for bootstrap, and registers the hidden service.
// If no tor binary is available, Start is a successful no-op so callers can
// invoke it unconditionally during server startup.
func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("tor: nil service")
	}
	if s.cfg.LocalPort == 0 {
		return errors.New("tor: LocalPort is required")
	}
	bin := s.cfg.Binary
	if bin == "" {
		bin = DetectBinary()
	}
	if bin == "" {
		log.Println("tor: binary not found on PATH — hidden service disabled")
		return nil
	}

	startConf := &tor.StartConf{
		ExePath:   bin,
		DataDir:   s.cfg.DataDir,
		NoHush:    !s.cfg.SafeLogging,
		DebugWriter: nil,
	}
	t, err := tor.Start(ctx, startConf)
	if err != nil {
		return fmt.Errorf("tor: start: %w", err)
	}
	t.DeleteDataDirOnClose = s.cfg.DataDir == ""

	bootCtx, cancel := context.WithTimeout(ctx, s.cfg.BootstrapTimeout)
	defer cancel()

	listenConf := &tor.ListenConf{
		Version3:    true,
		RemotePorts: []int{s.cfg.VirtualPort},
		LocalPort:   s.cfg.LocalPort,
	}
	onion, err := t.Listen(bootCtx, listenConf)
	if err != nil {
		_ = t.Close()
		return fmt.Errorf("tor: hidden service listen: %w", err)
	}

	s.mu.Lock()
	s.t = t
	s.onion = onion
	s.addr = onion.ID + ".onion"
	s.up = true
	s.mu.Unlock()

	log.Printf("tor: hidden service running at http://%s:%d (forwarded to localhost:%d)",
		s.addr, s.cfg.VirtualPort, s.cfg.LocalPort)
	return nil
}

// Stop tears down the hidden service and the embedded Tor process. Safe to
// call when Start was a no-op.
func (s *Service) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.up {
		return nil
	}
	var firstErr error
	if s.onion != nil {
		if err := s.onion.Close(); err != nil {
			firstErr = err
		}
	}
	if s.t != nil {
		if err := s.t.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.onion = nil
	s.t = nil
	s.up = false
	s.addr = ""
	return firstErr
}

// IsRunning reports whether the hidden service is actively forwarding.
func (s *Service) IsRunning() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.up
}

// OnionAddress returns the .onion hostname (e.g. abc...xyz.onion) when the
// service is running, or "" otherwise.
func (s *Service) OnionAddress() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

// Dial opens a connection to addr through the bundled Tor SOCKS5 proxy.
// Returns an error if the service is not running, since outbound routing
// requires a live Tor instance. Per AI.md PART 32 — server uses its OWN
// Tor process for outbound, never the system Tor.
func (s *Service) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	d, err := s.dialer(ctx)
	if err != nil {
		return nil, err
	}
	return d.DialContext(ctx, network, addr)
}

// dialer returns the bine *tor.Dialer for outbound use, after checking the
// service is actually running.
func (s *Service) dialer(ctx context.Context) (*tor.Dialer, error) {
	if s == nil || !s.IsRunning() {
		return nil, errors.New("tor: outbound requires a running hidden service")
	}
	s.mu.RLock()
	t := s.t
	s.mu.RUnlock()
	if t == nil {
		return nil, errors.New("tor: nil tor client")
	}
	d, err := t.Dialer(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("tor: dialer: %w", err)
	}
	return d, nil
}

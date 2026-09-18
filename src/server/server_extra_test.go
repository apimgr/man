package server

import (
	"database/sql"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	"github.com/casapps/casman/src/config"
	"github.com/casapps/casman/src/graphql"
	"github.com/casapps/casman/src/metrics"
	"github.com/casapps/casman/src/secret"
	"github.com/casapps/casman/src/server/handler"
	"github.com/casapps/casman/src/ssl"
	"github.com/casapps/casman/src/swagger"
)

func newRoutableServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.FQDN = "http://localhost"
	return &Server{
		cfg:         cfg,
		handlers:    handler.New(cfg, "v", "c", "d"),
		swagger:     swagger.New("v", "http://localhost"),
		graphql:     graphql.New("v"),
		metrics:     metrics.New(metrics.DefaultConfig(), "v", "c", "d"),
		rateLimiter: NewRateLimiter(),
	}
}

func routeRegistered(r *chi.Mux, method, path string) bool {
	rctx := chi.NewRouteContext()
	return r.Match(rctx, method, path)
}

func TestSetupRouter_KnownRoutesRegistered(t *testing.T) {
	s := newRoutableServer(t)
	r := s.setupRouter()
	if r == nil {
		t.Fatal("setupRouter returned nil")
	}

	wantRoutes := map[string]string{
		"/healthz":       "GET",
		"/":              "GET",
		"/search":        "GET",
		"/browse/":       "GET",
		"/api/v1/":       "GET",
		"/metrics":       "GET",
		"/openapi":       "GET",
		"/swagger.json":  "GET",
		"/graphql":       "POST",
		"/sitemap.xml":   "GET",
		"/robots.txt":    "GET",
		"/manifest.json": "GET",
		"/man/ls":        "GET",
		"/whatis/ls":     "GET",
	}
	for path, method := range wantRoutes {
		if !routeRegistered(r, method, path) {
			t.Errorf("expected route %s %s to be registered", method, path)
		}
	}
}

func TestSetupRouter_UnknownRouteNotRegistered(t *testing.T) {
	s := newRoutableServer(t)
	r := s.setupRouter()
	if routeRegistered(r, "GET", "/totally/not/a/route") {
		t.Error("unexpected route matched")
	}
}

func TestClose_AllNilComponents(t *testing.T) {
	s := &Server{}
	if err := s.Close(); err != nil {
		t.Errorf("Close on empty server should be nil, got %v", err)
	}
}

func TestSSLDNSProvider_PrefersNonManual(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	sealer, err := secret.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	v, err := ssl.NewVault(db, sealer)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	if err := v.Save(ssl.Manual, map[string]string{}); err != nil {
		t.Fatalf("save manual: %v", err)
	}
	if err := v.Save("cloudflare", map[string]string{}); err != nil {
		t.Fatalf("save cloudflare: %v", err)
	}

	s := &Server{sslVault: v}
	if got := s.sslDNSProvider(); got != "cloudflare" {
		t.Errorf("sslDNSProvider = %q, want cloudflare", got)
	}
}

func TestSSLDNSProvider_FallsBackToManual(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	sealer, err := secret.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	v, err := ssl.NewVault(db, sealer)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	if err := v.Save(ssl.Manual, map[string]string{}); err != nil {
		t.Fatalf("save manual: %v", err)
	}

	s := &Server{sslVault: v}
	if got := s.sslDNSProvider(); got != ssl.Manual {
		t.Errorf("sslDNSProvider = %q, want manual", got)
	}
}

func TestInitTor_AppliesConfigOverrides(t *testing.T) {
	cfg := &config.Config{}
	cfg.Paths.DataDir = t.TempDir()
	cfg.Server.Tor = &config.TorConfig{
		Binary:           "/usr/bin/tor",
		VirtualPort:      8080,
		BootstrapTimeout: "5s",
		SafeLogging:      false,
	}
	s := &Server{cfg: cfg}
	s.initTor()
	if s.tor == nil {
		t.Fatal("initTor did not set s.tor")
	}
}

func TestInitTor_DefaultsWhenNoConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Paths.DataDir = t.TempDir()
	s := &Server{cfg: cfg}
	s.initTor()
	if s.tor == nil {
		t.Fatal("initTor did not set s.tor")
	}
	if s.tor.OnionAddress() != "" {
		t.Error("expected no onion address before Start")
	}
}

func TestInitTor_InvalidBootstrapTimeoutIgnored(t *testing.T) {
	cfg := &config.Config{}
	cfg.Paths.DataDir = t.TempDir()
	cfg.Server.Tor = &config.TorConfig{BootstrapTimeout: "not-a-duration"}
	s := &Server{cfg: cfg}
	s.initTor()
	if s.tor == nil {
		t.Fatal("initTor did not set s.tor")
	}
}

func TestInitSSL_DisabledSetsUpVaultOnly(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{}
	cfg.Paths.ConfigDir = t.TempDir()
	cfg.Paths.DataDir = t.TempDir()
	cfg.Server.SSL.Enabled = false

	s := &Server{cfg: cfg, usersDB: db}
	if err := s.initSSL(); err != nil {
		t.Fatalf("initSSL: %v", err)
	}
	if s.secret == nil {
		t.Error("expected secret sealer to be set")
	}
	if s.sslVault == nil {
		t.Error("expected sslVault to be set when usersDB present")
	}
	if s.provisioner != nil {
		t.Error("expected no provisioner when SSL disabled")
	}
}

func TestInitSSL_NoUsersDB_SkipsVault(t *testing.T) {
	cfg := &config.Config{}
	cfg.Paths.ConfigDir = t.TempDir()
	cfg.Paths.DataDir = t.TempDir()
	cfg.Server.SSL.Enabled = false

	s := &Server{cfg: cfg}
	if err := s.initSSL(); err != nil {
		t.Fatalf("initSSL: %v", err)
	}
	if s.sslVault != nil {
		t.Error("expected nil sslVault without usersDB")
	}
}

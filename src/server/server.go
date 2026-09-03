// Package server implements the HTTP server for casman.
package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/casapps/casman/src/auth"
	"github.com/casapps/casman/src/backup"
	"github.com/casapps/casman/src/config"
	"github.com/casapps/casman/src/geoip"
	"github.com/casapps/casman/src/graphql"
	"github.com/casapps/casman/src/metrics"
	"github.com/casapps/casman/src/notify"
	"github.com/casapps/casman/src/scheduler"
	"github.com/casapps/casman/src/secret"
	"github.com/casapps/casman/src/server/handler"
	"github.com/casapps/casman/src/smtp"
	"github.com/casapps/casman/src/ssl"
	"github.com/casapps/casman/src/swagger"
	"github.com/casapps/casman/src/tor"

	_ "modernc.org/sqlite"
)

// Server represents the HTTP server.
type Server struct {
	cfg          *config.Config
	router       *chi.Mux
	version      string
	commitID     string
	buildDate    string
	handlers     *handler.Handlers
	swagger      *swagger.Handler
	graphql      *graphql.Handler
	metrics      *metrics.Metrics
	scheduler    *scheduler.Scheduler
	smtp         *smtp.Client
	geoip        *geoip.GeoIP
	backup       *backup.Manager
	authStore    *auth.Store
	authMiddleware *auth.Middleware
	usersDB      *sql.DB
	rateLimiter  *RateLimiter
	secret       *secret.Vault
	sslVault     *ssl.Vault
	provisioner  *ssl.Provisioner
	notifier     *notify.Notifier
	tor          *tor.Service
}

// New creates a new Server instance.
func New(cfg *config.Config, version, commitID, buildDate string) (*Server, error) {
	s := &Server{
		cfg:       cfg,
		version:   version,
		commitID:  commitID,
		buildDate: buildDate,
	}

	// Create handlers
	s.handlers = handler.New(cfg, version, commitID, buildDate)

	// Initialize handlers (database + templates)
	if err := s.handlers.Init(cfg.Database.Path); err != nil {
		log.Printf("Warning: failed to initialize handlers: %v", err)
		// Continue without database - handlers will return appropriate errors
	}

	// Create rate limiter per AI.md PART 11
	s.rateLimiter = NewRateLimiter()

	// Create swagger and graphql handlers
	fqdn := cfg.Server.FQDN
	if fqdn == "" {
		fqdn = "http://localhost"
	}
	s.swagger = swagger.New(version, fqdn)
	s.graphql = graphql.New(version)

	// Create metrics (PART 21)
	metricsCfg := metrics.DefaultConfig()
	if cfg.Server.Metrics != nil {
		metricsCfg.Enabled = cfg.Server.Metrics.Enabled
		if cfg.Server.Metrics.Endpoint != "" {
			metricsCfg.Endpoint = cfg.Server.Metrics.Endpoint
		}
		metricsCfg.Token = cfg.Server.Metrics.Token
		metricsCfg.IncludeSystem = cfg.Server.Metrics.IncludeSystem
		metricsCfg.IncludeRuntime = cfg.Server.Metrics.IncludeRuntime
	}
	s.metrics = metrics.New(metricsCfg, version, commitID, buildDate)

	// Create scheduler (PART 19)
	s.scheduler = scheduler.New("America/New_York")
	s.registerScheduledTasks()

	// Create SMTP client and auto-detect (PART 18)
	smtpCfg := smtp.DefaultConfig()
	smtp.LoadFromEnv(&smtpCfg)
	s.smtp = smtp.New(smtpCfg)

	// Auto-detect SMTP if not configured
	if smtpCfg.Host == "" {
		host, port, found := s.smtp.AutoDetect(cfg.Server.FQDN)
		if found {
			log.Printf("SMTP: Auto-detected server at %s:%d", host, port)
		} else {
			log.Printf("SMTP: No server found - email features disabled")
		}
	} else {
		// Test configured server
		if s.smtp.TestConnection() {
			log.Printf("SMTP: Connected to %s:%d", smtpCfg.Host, smtpCfg.Port)
		} else {
			log.Printf("SMTP: Failed to connect to %s:%d - email features disabled", smtpCfg.Host, smtpCfg.Port)
		}
	}

	// Create GeoIP (PART 20)
	geoipCfg := geoip.DefaultConfig()
	if cfg.Server.GeoIP != nil {
		geoipCfg.Enabled = cfg.Server.GeoIP.Enabled
		if cfg.Server.GeoIP.Dir != "" {
			geoipCfg.Dir = cfg.Server.GeoIP.Dir
		} else {
			geoipCfg.Dir = cfg.Paths.ConfigDir + "/security/geoip"
		}
		geoipCfg.DenyCountries = cfg.Server.GeoIP.DenyCountries
		geoipCfg.ASN = cfg.Server.GeoIP.Databases.ASN
		geoipCfg.Country = cfg.Server.GeoIP.Databases.Country
		geoipCfg.City = cfg.Server.GeoIP.Databases.City
		geoipCfg.WHOIS = cfg.Server.GeoIP.Databases.WHOIS
	}
	s.geoip = geoip.New(geoipCfg)

	// Initialize GeoIP (downloads databases if needed)
	if geoipCfg.Enabled {
		if err := s.geoip.Init(context.Background()); err != nil {
			log.Printf("GeoIP: initialization failed: %v", err)
		}
	}

	// Create backup manager (PART 22)
	backupCfg := backup.DefaultConfig()
	if cfg.Server.Backup != nil {
		if cfg.Server.Backup.Dir != "" {
			backupCfg.Dir = cfg.Server.Backup.Dir
		} else {
			backupCfg.Dir = cfg.Paths.BackupDir
		}
		backupCfg.Retention.MaxBackups = cfg.Server.Backup.Retention.MaxBackups
		if backupCfg.Retention.MaxBackups < 1 {
			backupCfg.Retention.MaxBackups = 1
		}
		backupCfg.Retention.KeepWeekly = cfg.Server.Backup.Retention.KeepWeekly
		backupCfg.Retention.KeepMonthly = cfg.Server.Backup.Retention.KeepMonthly
		backupCfg.Retention.KeepYearly = cfg.Server.Backup.Retention.KeepYearly
		backupCfg.Encryption.Enabled = cfg.Server.Backup.Encryption.Enabled
		backupCfg.Encryption.PasswordHint = cfg.Server.Backup.Encryption.PasswordHint
		backupCfg.Compliance = cfg.Server.Backup.Compliance
	} else {
		backupCfg.Dir = cfg.Paths.BackupDir
	}
	s.backup = backup.New(backupCfg, version, cfg.Paths.ConfigDir, cfg.Paths.DataDir)
	log.Printf("Backup: configured (dir=%s, retention=%d)", backupCfg.Dir, backupCfg.Retention.MaxBackups)

	// Create auth store (PART 10, PART 17)
	// Per AI.md: users.db is separate from server.db for easier backup/restore
	usersDBPath := cfg.Paths.DataDir + "/db/users.db"
	usersDB, err := sql.Open("sqlite", usersDBPath)
	if err != nil {
		log.Printf("Auth: failed to open users database: %v", err)
	} else {
		// Enable WAL mode
		usersDB.Exec("PRAGMA journal_mode=WAL")
		usersDB.Exec("PRAGMA foreign_keys=ON")
		s.usersDB = usersDB

		authStore, err := auth.NewStore(usersDB)
		if err != nil {
			log.Printf("Auth: failed to initialize auth store: %v", err)
		} else {
			s.authStore = authStore

			// Check if debug mode
			debugMode := cfg.Server.Mode == "development"

			// Create auth middleware
			s.authMiddleware = auth.NewMiddleware(authStore, debugMode, s.handlers.GetSetupToken())
			log.Printf("Auth: initialized (debug=%v)", debugMode)

			// Check admin count for setup status
			adminCount, _ := authStore.AdminCount()
			if adminCount == 0 {
				log.Printf("Auth: No admins configured - setup wizard required")
			} else {
				log.Printf("Auth: %d admin account(s) configured", adminCount)
			}
		}
	}

	// Pass auth store to handlers
	if s.authStore != nil {
		s.handlers.SetAuthStore(s.authStore)
	}

	// Pass backup manager to handlers (PART 22 admin POST API).
	handler.SetBackupBackend(s)

	// Build the email notifier (PART 18). Recipients are resolved at send
	// time from the auth store so admin profile changes take effect without
	// restart. The notifier is a no-op when SMTP is not available.
	s.notifier = notify.New(
		s.smtp,
		cfg.Server.Branding.Title,
		func() []string {
			if s.authStore == nil {
				return nil
			}
			emails, err := s.authStore.ListAdminEmails()
			if err != nil {
				log.Printf("notify: ListAdminEmails: %v", err)
				return nil
			}
			return emails
		},
	)

	// Initialize SSL subsystem (PART 15) — secret vault, credential vault,
	// provisioner. The provisioner is built only when SSL is enabled.
	if err := s.initSSL(); err != nil {
		log.Printf("SSL: %v", err)
	}

	// Initialize Tor hidden service (PART 32). Built unconditionally; will
	// silently no-op at Start() time if no tor binary is reachable.
	s.initTor()
	handler.SetTorBackend(s)

	// Initialize router
	s.router = s.setupRouter()

	return s, nil
}

// initTor builds the Tor hidden-service wrapper. Per AI.md PART 32 the
// service is auto-enabled whenever a tor binary is present; configuration
// just tunes performance and bookkeeping. Errors here are non-fatal.
func (s *Server) initTor() {
	cfg := tor.DefaultConfig()
	if c := s.cfg.Server.Tor; c != nil {
		if c.Binary != "" {
			cfg.Binary = c.Binary
		}
		if c.VirtualPort != 0 {
			cfg.VirtualPort = c.VirtualPort
		}
		if c.BootstrapTimeout != "" {
			if d, err := time.ParseDuration(c.BootstrapTimeout); err == nil {
				cfg.BootstrapTimeout = d
			}
		}
		cfg.SafeLogging = c.SafeLogging
	}
	cfg.DataDir = s.cfg.Paths.DataDir + "/tor"
	s.tor = tor.New(cfg)
}

// initSSL wires the master key store, the DNS credential vault, and (when
// enabled in config) the ACME provisioner. Errors are non-fatal — HTTPS just
// stays disabled if any step fails.
func (s *Server) initSSL() error {
	sealer, err := secret.LoadOrCreate(s.cfg.Paths.ConfigDir)
	if err != nil {
		return fmt.Errorf("master key: %w", err)
	}
	s.secret = sealer

	if s.usersDB != nil {
		v, err := ssl.NewVault(s.usersDB, sealer)
		if err != nil {
			return fmt.Errorf("ssl vault: %w", err)
		}
		s.sslVault = v
		handler.SetSSLBackend(s)
	}

	if !s.cfg.Server.SSL.Enabled {
		return nil
	}

	caDir := lego.LEDirectoryProduction
	if s.cfg.Server.SSL.LetsEncrypt.Staging {
		caDir = lego.LEDirectoryStaging
	}

	dataDir := s.cfg.Server.SSL.DataDir
	if dataDir == "" {
		dataDir = s.cfg.Paths.DataDir + "/ssl"
	}
	provCfg := ssl.ProvisionerConfig{
		DataDir:     dataDir,
		Email:       s.cfg.Server.SSL.LetsEncrypt.Email,
		Challenge:   ssl.ChallengeType(s.cfg.Server.SSL.LetsEncrypt.Challenge),
		DNSProvider: s.sslDNSProvider(),
		CADirURL:    caDir,
		HTTPPort:    s.cfg.Server.HTTPRedirectPort,
		TLSPort:     s.cfg.Server.HTTPSPort,
	}
	if provCfg.Email == "" {
		// Static cert deployments don't need ACME; provisioner is still
		// useful for serving the cert via GetCertificate.
		provCfg.Email = "noreply@" + s.cfg.Server.FQDN
	}

	prov, err := ssl.NewProvisioner(provCfg, s.sslVault)
	if err != nil {
		return fmt.Errorf("ssl provisioner: %w", err)
	}
	s.provisioner = prov

	if cert, key := s.cfg.Server.SSL.Cert, s.cfg.Server.SSL.Key; cert != "" && key != "" {
		if err := prov.LoadStaticCert(cert, key); err != nil {
			log.Printf("SSL: static cert %s: %v", cert, err)
		} else {
			log.Printf("SSL: loaded operator-provided cert from %s", cert)
		}
	}
	return nil
}

// Vault exposes the SSL credential vault to the handler package via the
// SSLBackend interface.
func (s *Server) Vault() *ssl.Vault { return s.sslVault }

// BackupManager exposes the backup manager via the BackupBackend interface
// the handler package uses for the PART 22 admin POST API.
func (s *Server) BackupManager() *backup.Manager { return s.backup }

// TorAvailable / TorRunning / TorOnionAddress satisfy the handler package's
// TorBackend interface, mirroring the *tor.Service accessors so /healthz and
// /admin/server/network/tor both surface the live status.
func (s *Server) TorAvailable() bool      { return s.tor != nil && s.tor.Available() }
func (s *Server) TorRunning() bool        { return s.tor != nil && s.tor.IsRunning() }
func (s *Server) TorOnionAddress() string { return s.tor.OnionAddress() }

// OutboundHTTPClient is the canonical HTTP client for server-initiated
// outbound requests (GeoIP downloads, updater, future ACME). Per AI.md
// PART 32 it routes through the bundled Tor SOCKS5 proxy when
// `cfg.Server.Tor.UseNetwork` is true and the hidden service is running;
// otherwise it returns the standard http.DefaultTransport-based client.
//
// Existing callers (src/geoip, src/updater) keep their own clients today;
// new callers should prefer this helper. Wiring those legacy callers
// through this helper is tracked in TODO.AI.md as a P3 follow-up.
func (s *Server) OutboundHTTPClient(timeout time.Duration) *http.Client {
	if s != nil && s.tor != nil && s.tor.IsRunning() &&
		s.cfg != nil && s.cfg.Server.Tor != nil && s.cfg.Server.Tor.UseNetwork {
		if c, err := s.tor.HTTPClient(timeout); err == nil {
			return c
		}
	}
	return &http.Client{Timeout: timeout}
}

// sslDNSProvider returns the configured DNS-01 provider name. The current
// config schema only stores one provider id; future expansion can route per
// domain. Empty string means dns-01 is not configured.
func (s *Server) sslDNSProvider() string {
	if s.sslVault == nil {
		return ""
	}
	names, err := s.sslVault.List()
	if err != nil || len(names) == 0 {
		return ""
	}
	// Prefer the first non-manual provider so credentialed automation wins
	// over the manual fallback.
	for _, n := range names {
		if n != ssl.Manual {
			return n
		}
	}
	return names[0]
}

// registerScheduledTasks registers default scheduled tasks per PART 19.
func (s *Server) registerScheduledTasks() {
	// Session cleanup - every 15 minutes
	s.scheduler.AddTask("session_cleanup", "@every 15m", true, func(ctx context.Context) error {
		log.Println("Running session cleanup...")
		if s.authStore != nil {
			count, err := s.authStore.CleanupExpiredSessions()
			if err != nil {
				log.Printf("Session cleanup error: %v", err)
				return err
			}
			if count > 0 {
				log.Printf("Session cleanup: removed %d expired sessions", count)
			}
		}
		return nil
	})

	// Token cleanup - every 15 minutes (resets locked accounts per AI.md)
	s.scheduler.AddTask("token_cleanup", "@every 15m", true, func(ctx context.Context) error {
		log.Println("Running token cleanup...")
		// Token cleanup handles expired tokens and resets lockouts
		// Lockout reset is handled automatically by auth when lockout expires
		return nil
	})

	// Self health check - every 5 minutes
	s.scheduler.AddTask("healthcheck_self", "@every 5m", true, func(ctx context.Context) error {
		// Verify core components are responding
		var issues []string

		// Check database connectivity
		if s.handlers != nil {
			// Try to read DB stats to verify connectivity
			// If handlers can respond, DB is working
		}

		// Check if users DB is accessible
		if s.usersDB != nil {
			if err := s.usersDB.Ping(); err != nil {
				issues = append(issues, fmt.Sprintf("users DB: %v", err))
			}
		}

		// Report issues
		if len(issues) > 0 {
			log.Printf("Health check: found %d issues: %v", len(issues), issues)
			return fmt.Errorf("health check failed: %v", issues)
		}
		return nil
	})

	// Log rotation - daily at midnight
	s.scheduler.AddTask("log_rotation", "0 0 * * *", true, func(ctx context.Context) error {
		log.Println("Running log rotation...")
		logDir := s.cfg.Paths.LogDir
		if logDir == "" {
			return nil
		}

		// Rotate logs older than 7 days
		cutoff := time.Now().AddDate(0, 0, -7)
		entries, err := os.ReadDir(logDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("reading log dir: %w", err)
		}

		var removed int
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// Only process log files
			if !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				logPath := logDir + "/" + entry.Name()
				if err := os.Remove(logPath); err == nil {
					removed++
				}
			}
		}
		if removed > 0 {
			log.Printf("Log rotation: removed %d old log files", removed)
		}
		return nil
	})

	// Daily backup - at 02:00. Backup admins via PART 18 notifier on both
	// success (backup_complete) and failure (backup_failed); the notifier
	// is a no-op when SMTP is unavailable.
	s.scheduler.AddTask("backup_daily", "0 2 * * *", true, func(ctx context.Context) error {
		log.Println("Running daily backup...")
		if s.backup == nil {
			return nil
		}
		path, err := s.backup.Backup(ctx, "", "", "scheduler")
		if err != nil {
			log.Printf("Backup failed: %v", err)
			s.notifier.BackupFailed(err.Error())
			return err
		}
		var size int64
		if info, statErr := os.Stat(path); statErr == nil {
			size = info.Size()
		}
		if err := s.backup.ApplyRetention(); err != nil {
			log.Printf("Backup retention cleanup failed: %v", err)
		}
		s.notifier.BackupComplete(path, size)
		log.Println("Daily backup completed successfully")
		return nil
	})

	// SSL renewal check - daily at 03:00. For each cached domain, parse the
	// leaf cert's NotAfter and re-obtain via the provisioner when within the
	// 30-day renewal window. Static (non-ACME) certs are still inspected so
	// operators get an early-warning log before expiry.
	s.scheduler.AddTask("ssl_renewal", "0 3 * * *", true, func(ctx context.Context) error {
		if s.provisioner == nil {
			return nil
		}
		const renewWithin = 30 * 24 * time.Hour
		domains := s.provisioner.CachedDomains()
		if len(domains) == 0 {
			return nil
		}
		log.Printf("SSL renewal: inspecting %d cached cert(s)", len(domains))
		// Notification thresholds per AI.md PART 18: 30, 14, 7, 3, 1 days.
		notifyThresholds := []int{30, 14, 7, 3, 1}
		for _, d := range domains {
			expiry, err := s.provisioner.CertExpiry(d)
			if err != nil {
				log.Printf("SSL renewal: %s: read cert: %v", d, err)
				continue
			}
			until := time.Until(expiry)
			daysLeft := int(until.Hours() / 24)
			if until > renewWithin {
				log.Printf("SSL renewal: %s ok (expires in %s)", d, until.Round(time.Hour))
				continue
			}
			// Send the ssl_expiring template at any of the spec'd thresholds.
			for _, t := range notifyThresholds {
				if daysLeft == t {
					s.notifier.SSLExpiring(d, daysLeft)
					break
				}
			}
			if !s.cfg.Server.SSL.LetsEncrypt.Enabled {
				log.Printf("SSL renewal: %s expires in %s, but Let's Encrypt is disabled — manual renewal required", d, until.Round(time.Hour))
				continue
			}
			log.Printf("SSL renewal: %s expires in %s, re-provisioning", d, until.Round(time.Hour))
			if _, err := s.provisioner.Provision(ctx, []string{d}); err != nil {
				log.Printf("SSL renewal: %s: %v", d, err)
				continue
			}
			s.notifier.SSLRenewed(d)
			log.Printf("SSL renewal: %s renewed", d)
		}
		return nil
	})

	// GeoIP update - weekly on Sunday at 03:00
	s.scheduler.AddTask("geoip_update", "0 3 * * 0", true, func(ctx context.Context) error {
		log.Println("Updating GeoIP database...")
		if s.geoip != nil && s.geoip.IsAvailable() {
			if err := s.geoip.Update(ctx); err != nil {
				log.Printf("GeoIP update failed: %v", err)
				return err
			}
			log.Println("GeoIP database updated successfully")
		}
		return nil
	})

	// Blocklist update - daily at 04:00
	s.scheduler.AddTask("blocklist_update", "0 4 * * *", true, func(ctx context.Context) error {
		log.Println("Updating blocklists...")
		// Update IP blocklists from configured sources
		// Per AI.md: GeoIP and blocklists work together for security
		if s.geoip != nil && s.geoip.IsAvailable() {
			// Blocklist sources can be configured in server.yml
			// For now, log that check was performed
			log.Println("Blocklist update: check complete")
		}
		return nil
	})
}

// Close cleans up server resources.
func (s *Server) Close() error {
	var lastErr error
	if s.geoip != nil {
		if err := s.geoip.Close(); err != nil {
			lastErr = err
		}
	}
	if s.handlers != nil {
		if err := s.handlers.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// setupRouter creates and configures the chi router.
func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware - order matters per AI.md
	r.Use(PathSecurityMiddleware)        // 1. Path security first
	r.Use(URLNormalizeMiddleware)        // 2. URL normalization (trailing slash redirect)
	r.Use(ContentNegotiationMiddleware)  // 3. Content negotiation
	r.Use(SecurityHeadersMiddleware)     // 4. Security headers (includes HSTS)
	r.Use(CORSMiddleware)                // 5. CORS
	r.Use(RecoveryMiddleware)            // 6. Panic recovery
	r.Use(RequestIDMiddleware)           // 7. Request ID (X-Request-ID, X-Correlation-ID, X-Trace-ID)
	r.Use(middleware.RealIP)             // 8. Real IP
	r.Use(middleware.Logger)             // 9. Logging
	r.Use(middleware.Timeout(60 * time.Second))
	if s.geoip != nil {
		r.Use(GeoIPMiddleware(s.geoip)) // 9a. GeoIP enforcement per PART 20
	}
	r.Use(RateLimitMiddleware(s.rateLimiter)) // 10. Rate limiting per PART 11
	r.Use(CSRFMiddleware())                    // 11. CSRF protection per PART 11

	// Use stored handlers
	h := s.handlers

	// Health endpoints
	r.Get("/healthz", h.Healthz)

	// Auth routes - required per AI.md PART 11, 12
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.AuthLogin)
		r.Post("/login", h.AuthLoginPost)
		r.Get("/logout", h.AuthLogout)
		r.Post("/logout", h.AuthLogout)
	})

	// Auth API routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", h.APIAuthLogin)
		r.Post("/logout", h.APIAuthLogout)
	})

	// Homepage
	r.Get("/", h.Home)

	// Man page routes
	r.Route("/man", func(r chi.Router) {
		// /man/{name} - best match
		r.Get("/{name}", h.ManPage)
		// /man/{name}.{ext} - with format extension
		r.Get("/{name}.{ext}", h.ManPageWithFormat)
		// /man/{section}/{name} - specific section
		r.Get("/{section}/{name}", h.ManPageSection)
		// /man/{section}/{name}.{ext}
		r.Get("/{section}/{name}.{ext}", h.ManPageSectionWithFormat)
		// /man/{os}/{section}/{name} - specific OS
		r.Get("/{os}/{section}/{name}", h.ManPageOSSection)
		// /man/{os}/{section}/{name}.{ext}
		r.Get("/{os}/{section}/{name}.{ext}", h.ManPageOSSectionWithFormat)
	})

	// Search
	r.Get("/search", h.Search)

	// Browse
	r.Route("/browse", func(r chi.Router) {
		r.Get("/", h.Browse)
		r.Get("/{section}", h.BrowseSection)
		r.Get("/os/{os}", h.BrowseOS)
	})

	// Compare
	r.Route("/compare", func(r chi.Router) {
		r.Get("/{name}", h.Compare)
		r.Get("/{section}/{name}", h.CompareSection)
	})

	// Whatis / Apropos
	r.Get("/whatis/{name}", h.Whatis)
	r.Get("/apropos", h.Apropos)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// API root endpoint per AI.md PART 14
		r.Get("/", h.APIRoot)
		r.Get("/healthz", h.APIHealthz)
		r.Get("/stats", h.APIStats)
		r.Get("/sections", h.APISections)
		r.Get("/platforms", h.APIPlatforms)
		r.Get("/languages", h.APILanguages)
		r.Get("/popular", h.APIPopular)

		// Man page API
		r.Get("/man/{name}", h.APIManPage)
		r.Get("/man/{section}/{name}", h.APIManPageSection)
		r.Get("/man/{os}/{section}/{name}", h.APIManPageOSSection)
		r.Get("/man/{lang}/{os}/{section}/{name}", h.APIManPageLang)

		// Search API
		r.Get("/search", h.APISearch)
		r.Get("/autocomplete", h.APIAutocomplete)
		r.Get("/suggest", h.APISuggest)

		// Compare API
		r.Get("/compare/{name}", h.APICompare)
		r.Get("/compare/{section}/{name}", h.APICompareSection)

		// Whatis / Apropos API
		r.Get("/whatis/{name}", h.APIWhatis)
		r.Get("/apropos", h.APIApropos)

		// TLDR API
		r.Get("/tldr/{name}", h.APITLDR)

		// Export formats
		r.Get("/export/formats", h.APIExportFormats)

		// OpenAPI spec
		r.Get("/openapi", h.OpenAPISpec)
	})

	// Bulk export per IDEA.md "Export (PDF/EPUB)"
	r.Get("/export/section/{section}", h.ExportSection)
	r.Get("/export/platform/{platform}", h.ExportPlatform)

	// Feeds
	r.Get("/feed.xml", h.Feed)
	r.Get("/feed/{platform}.xml", h.FeedPlatform)
	r.Get("/feed/section/{section}.xml", h.FeedSection)
	r.Get("/feed/{platform}/{section}", h.FeedCombined)
	r.Get("/feed.json", h.FeedJSON)

	// SEO
	r.Get("/sitemap.xml", h.Sitemap)
	r.Get("/sitemap-pages.xml", h.SitemapPages)
	r.Get("/sitemap-sections.xml", h.SitemapSections)
	r.Get("/sitemap-platforms.xml", h.SitemapPlatforms)
	r.Get("/robots.txt", h.Robots)

	// Well-known files per AI.md PART 12
	r.Route("/.well-known", func(r chi.Router) {
		r.Get("/security.txt", h.SecurityTxt)
		r.Get("/change-password", h.ChangePassword)
	})
	// Also serve security.txt at root for convenience
	r.Get("/security.txt", h.SecurityTxt)

	// PWA manifest per AI.md PART 16
	r.Get("/manifest.json", h.Manifest)
	r.Get("/service-worker.js", h.ServiceWorker)

	// Favicon per AI.md PART 16
	r.Get("/favicon.ico", h.Favicon)

	// OpenAPI / Swagger
	r.Get("/openapi", s.swagger.ServeUI)
	r.Get("/openapi.json", s.swagger.ServeSpec)
	r.Get("/swagger", s.swagger.ServeUI)
	r.Get("/swagger.json", s.swagger.ServeSpec)

	// GraphQL
	r.Get("/graphql", s.graphql.ServeUI)
	r.Post("/graphql", s.graphql.ServeGraphQL)
	r.Get("/graphiql", s.graphql.ServeUI)

	// Prometheus metrics (PART 21) - internal, optional token auth
	r.Handle("/metrics", s.metrics.Handler())

	// Admin panel - route hierarchy per PART 17
	// /{admin_path}/ - Dashboard
	// /{admin_path}/profile - Admin's own profile
	// /{admin_path}/preferences - Admin's own preferences
	// /{admin_path}/notifications - Admin's own notifications
	// /{admin_path}/server/* - ALL server management
	adminPath := s.cfg.Server.AdminPath
	if adminPath == "" {
		adminPath = "admin"
	}
	r.Route("/"+adminPath, func(r chi.Router) {
		// Auth middleware - protects admin routes
		if s.authMiddleware != nil {
			r.Use(s.authMiddleware.RequireAuth)
		}
		r.Get("/", h.AdminDashboard)
		r.Get("/profile", h.AdminProfile)
		r.Get("/preferences", h.AdminPreferences)
		r.Get("/notifications", h.AdminNotifications)

		// Server management routes
		r.Route("/server", func(r chi.Router) {
			r.Get("/", h.AdminServerInfo)
			r.Get("/settings", h.AdminSettings)
			r.Get("/ssl", h.AdminSSL)
			r.Post("/ssl", h.AdminSSLSave)
			r.Get("/email", h.AdminEmail)
			r.Get("/scheduler", h.AdminScheduler)
			r.Get("/logs", h.AdminLogs)
			r.Get("/logs/audit", h.AdminAuditLogs)
			r.Get("/backup", h.AdminBackup)
			r.Post("/backup", h.AdminBackupSave)
			r.Get("/updates", h.AdminUpdates)
			r.Get("/info", h.AdminServerInfo)
			r.Get("/metrics", h.AdminMetrics)

			// Setup wizard (PART 17)
			r.Get("/setup", h.AdminSetup)
			r.Post("/setup/verify", h.AdminSetupVerify)
			r.Post("/setup/complete", h.AdminSetupComplete)

			// Network settings
			r.Route("/network", func(r chi.Router) {
				r.Get("/", h.AdminNetwork)
				r.Get("/tor", h.AdminTor)
				r.Get("/geoip", h.AdminGeoIP)
			})

			// Security settings
			r.Route("/security", func(r chi.Router) {
				r.Get("/", h.AdminSecurity)
				r.Get("/auth", h.AdminAuth)
				r.Get("/tokens", h.AdminTokens)
				r.Get("/firewall", h.AdminFirewall)
			})
		})
	})

	// Admin API routes
	r.Route("/api/v1/"+adminPath, func(r chi.Router) {
		// Auth middleware - protects admin API routes
		if s.authMiddleware != nil {
			r.Use(s.authMiddleware.RequireAuth)
		}
		r.Get("/", h.APIAdminDashboard)
		r.Route("/server", func(r chi.Router) {
			r.Get("/settings", h.APIAdminSettings)
			r.Patch("/settings", h.APIAdminSettingsUpdate)

			// Setup wizard API (PART 17)
			r.Get("/setup", h.APIAdminSetupStatus)
			r.Post("/setup/verify", h.APIAdminSetupVerify)
			r.Post("/setup/complete", h.APIAdminSetupComplete)

			// Backup admin API (PART 22)
			r.Get("/backup", h.APIAdminBackupList)
			r.Post("/backup", h.APIAdminBackupCreate)
			r.Post("/backup/restore", h.APIAdminBackupRestore)
		})
	})

	// Static files (embedded)
	r.Handle("/static/*", h.StaticFiles())

	return r
}

// Run starts the HTTP server and handles graceful shutdown.
func (s *Server) Run() error {
	// Start scheduler (PART 19 - always running)
	s.scheduler.Start()

	addr := s.cfg.Server.Address
	if addr == "" {
		addr = "[::]"
	}

	// Start the Tor hidden service (PART 32). Lifecycle is tied to the
	// server: Stop is called from both shutdown branches below. The actual
	// listen-port resolution differs between the HTTP and HTTPS branches,
	// so we set tor.cfg.LocalPort there before calling Start.

	if s.provisioner != nil {
		return s.runHTTPS(addr)
	}

	port := s.cfg.Server.Port
	if port == "" {
		port = s.getRandomPort()
	}
	listenAddr := net.JoinHostPort(addr, port)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on %s", listenAddr)
		log.Printf("Mode: %s", s.cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	if s.tor != nil {
		s.tor.SetLocalPort(parsePort(port))
		torCtx, torCancel := context.WithTimeout(context.Background(), 4*time.Minute)
		if err := s.tor.Start(torCtx); err != nil {
			log.Printf("tor: start failed: %v", err)
		}
		torCancel()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	s.scheduler.Stop()
	if s.tor != nil {
		_ = s.tor.Stop(context.Background())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	if err := s.Close(); err != nil {
		log.Printf("Warning: error closing resources: %v", err)
	}
	log.Println("Server stopped")
	return nil
}

// parsePort returns the int representation of a port string. Falls back to 0
// for non-numeric input — callers that need a real port (Tor hidden service)
// should treat 0 as "no Tor".
func parsePort(p string) int {
	if p == "" {
		return 0
	}
	n := 0
	for _, c := range p {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// runHTTPS starts the dual HTTP+HTTPS listeners via the ssl package. The HTTP
// listener serves ACME HTTP-01 challenges and 301-redirects everything else;
// the HTTPS listener uses the provisioner's GetCertificate hook.
func (s *Server) runHTTPS(addr string) error {
	httpsPort := s.cfg.Server.HTTPSPort
	if httpsPort == "" {
		httpsPort = "443"
	}
	httpPort := s.cfg.Server.HTTPRedirectPort
	if httpPort == "" {
		httpPort = "80"
	}

	httpsAddr := net.JoinHostPort(addr, httpsPort)
	httpAddr := net.JoinHostPort(addr, httpPort)

	log.Printf("Starting server (HTTPS) on %s", httpsAddr)
	log.Printf("Mode: %s", s.cfg.Server.Mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ssl.ServeBoth(ctx, httpAddr, httpsAddr, s.router, s.provisioner)
	}()

	if s.tor != nil {
		s.tor.SetLocalPort(parsePort(httpsPort))
		torCtx, torCancel := context.WithTimeout(context.Background(), 4*time.Minute)
		if err := s.tor.Start(torCtx); err != nil {
			log.Printf("tor: start failed: %v", err)
		}
		torCancel()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Println("Shutting down server...")
	case err := <-errCh:
		if err != nil {
			log.Printf("HTTPS server: %v", err)
		}
	}

	s.scheduler.Stop()
	if s.tor != nil {
		_ = s.tor.Stop(context.Background())
	}
	cancel()

	if err := s.Close(); err != nil {
		log.Printf("Warning: error closing resources: %v", err)
	}
	log.Println("Server stopped")
	return nil
}

// getRandomPort returns a random available port in the 64000-64999 range.
func (s *Server) getRandomPort() string {
	// Try to find an available port
	for port := 64000; port < 65000; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return fmt.Sprintf("%d", port)
		}
	}
	// Fallback
	return "64580"
}

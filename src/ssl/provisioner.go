// Provisioner wraps go-acme/lego to obtain Let's Encrypt certificates and
// serves them to a TLS listener via GetCertificate. Certificates and the ACME
// account key are cached on disk under {data_dir}/ssl/. See AI.md PART 15.

package ssl

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
)

// ProvisionerConfig holds runtime knobs for the provisioner.
type ProvisionerConfig struct {
	// DataDir is the on-disk cache root, e.g. {data_dir}/ssl.
	DataDir string
	// Email is the ACME registration email.
	Email string
	// Challenge selects HTTP-01, TLS-ALPN-01, or DNS-01.
	Challenge ChallengeType
	// DNSProvider is the lego provider ID when Challenge == DNS01.
	DNSProvider string
	// CADirURL overrides the ACME directory; empty defaults to Let's Encrypt
	// production. Use lego.LEDirectoryStaging for staging.
	CADirURL string
	// HTTPPort is the port lego binds for HTTP-01 challenges (default "80").
	HTTPPort string
	// TLSPort is the port lego binds for TLS-ALPN-01 challenges (default "443").
	TLSPort string
}

// Provisioner orchestrates ACME issuance and on-disk cert caching.
type Provisioner struct {
	cfg           ProvisionerConfig
	vault         *Vault
	user          *acmeUser
	client        *lego.Client
	httpChallenge *httpChallenge

	mu    sync.RWMutex
	certs map[string]*tls.Certificate
}

// NewProvisioner builds a provisioner ready to serve GetCertificate. The vault
// is consulted at provision time when challenge is DNS-01 to load credentials.
func NewProvisioner(cfg ProvisionerConfig, vault *Vault) (*Provisioner, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("ssl: provisioner DataDir required")
	}
	if cfg.Email == "" {
		return nil, errors.New("ssl: provisioner Email required")
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "80"
	}
	if cfg.TLSPort == "" {
		cfg.TLSPort = "443"
	}
	if cfg.Challenge == "" {
		cfg.Challenge = HTTP01
	}

	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", cfg.DataDir, err)
	}

	user, err := loadOrCreateAccount(cfg.DataDir, cfg.Email)
	if err != nil {
		return nil, err
	}

	legoCfg := lego.NewConfig(user)
	if cfg.CADirURL != "" {
		legoCfg.CADirURL = cfg.CADirURL
	}
	legoCfg.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return nil, fmt.Errorf("lego client: %w", err)
	}

	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("acme registration: %w", err)
		}
		user.Registration = reg
		if err := saveAccount(cfg.DataDir, user); err != nil {
			return nil, err
		}
	}

	p := &Provisioner{
		cfg:           cfg,
		vault:         vault,
		user:          user,
		client:        client,
		httpChallenge: newHTTPChallenge(),
		certs:         map[string]*tls.Certificate{},
	}
	if err := p.loadDiskCerts(); err != nil {
		log.Printf("ssl: warming cert cache: %v", err)
	}
	return p, nil
}

// GetCertificate is the tls.Config.GetCertificate hook.
func (p *Provisioner) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := strings.ToLower(strings.TrimSpace(hello.ServerName))
	if host == "" {
		return nil, errors.New("ssl: missing SNI ServerName")
	}

	p.mu.RLock()
	if c, ok := p.certs[host]; ok {
		p.mu.RUnlock()
		return c, nil
	}
	p.mu.RUnlock()

	cert, err := p.Provision(context.Background(), []string{host})
	if err != nil {
		return nil, fmt.Errorf("ssl: provision %s: %w", host, err)
	}
	return cert, nil
}

// Provision obtains (or re-obtains) a certificate for the given domain set
// using the provisioner's challenge type. The first domain is treated as the
// primary; the remainder become Subject Alternative Names.
func (p *Provisioner) Provision(ctx context.Context, domains []string) (*tls.Certificate, error) {
	if len(domains) == 0 {
		return nil, errors.New("ssl: provision requires at least one domain")
	}
	if err := p.applyChallenge(ctx); err != nil {
		return nil, err
	}

	res, err := p.client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("acme obtain: %w", err)
	}

	if err := p.persistResource(res); err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(res.Certificate, res.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse issued cert: %w", err)
	}
	p.cache(domains[0], &cert)
	return &cert, nil
}

// LoadStaticCert installs an operator-supplied PEM cert/key pair into the
// cache so the TLS listener can serve it without ACME involvement.
func (p *Provisioner) LoadStaticCert(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("static cert: %w", err)
	}
	if len(cert.Certificate) == 0 {
		return errors.New("static cert: empty chain")
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("static cert parse: %w", err)
	}
	for _, name := range leaf.DNSNames {
		p.cache(strings.ToLower(name), &cert)
	}
	if leaf.Subject.CommonName != "" {
		p.cache(strings.ToLower(leaf.Subject.CommonName), &cert)
	}
	return nil
}

// CertPaths returns the cert/key file paths for a primary domain.
func (p *Provisioner) CertPaths(domain string) (cert, key string) {
	d := safeDomain(domain)
	return filepath.Join(p.cfg.DataDir, d+".crt"),
		filepath.Join(p.cfg.DataDir, d+".key")
}

// CachedDomains lists domains currently in the in-memory cache.
func (p *Provisioner) CachedDomains() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.certs))
	for d := range p.certs {
		out = append(out, d)
	}
	return out
}

func (p *Provisioner) cache(domain string, cert *tls.Certificate) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.certs[strings.ToLower(domain)] = cert
}

func (p *Provisioner) loadDiskCerts() error {
	entries, err := os.ReadDir(p.cfg.DataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".crt") {
			continue
		}
		domain := strings.TrimSuffix(name, ".crt")
		certFile, keyFile := p.CertPaths(domain)
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Printf("ssl: skip cert %s: %v", domain, err)
			continue
		}
		p.cache(domain, &cert)
	}
	return nil
}

func (p *Provisioner) persistResource(res *certificate.Resource) error {
	certFile, keyFile := p.CertPaths(res.Domain)
	if err := os.WriteFile(certFile, res.Certificate, 0644); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyFile, res.PrivateKey, 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	return nil
}

func (p *Provisioner) applyChallenge(ctx context.Context) error {
	switch p.cfg.Challenge {
	case HTTP01:
		// Use the inline handler so we share port 80 with the redirect server.
		return p.client.Challenge.SetHTTP01Provider(p.httpChallenge)
	case TLSALPN01:
		// TLS-ALPN-01 binds its own listener on the configured TLS port; this
		// is only viable when we are not already listening there ourselves.
		return p.client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", p.cfg.TLSPort))
	case DNS01:
		return p.applyDNSChallenge(ctx)
	default:
		return fmt.Errorf("ssl: unsupported challenge %q", p.cfg.Challenge)
	}
}

func (p *Provisioner) applyDNSChallenge(_ context.Context) error {
	if p.cfg.DNSProvider == "" {
		return errors.New("ssl: dns-01 selected but no DNS provider configured")
	}
	if p.cfg.DNSProvider == Manual {
		mp, err := dns.NewDNSChallengeProviderByName("manual")
		if err != nil {
			return fmt.Errorf("manual dns provider: %w", err)
		}
		return p.client.Challenge.SetDNS01Provider(mp)
	}

	if p.vault == nil {
		return errors.New("ssl: dns-01 selected but vault is nil")
	}
	creds, err := p.vault.Load(p.cfg.DNSProvider)
	if err != nil {
		return fmt.Errorf("loading dns credentials: %w", err)
	}
	restore := setEnv(creds)
	defer restore()

	provider, err := dns.NewDNSChallengeProviderByName(p.cfg.DNSProvider)
	if err != nil {
		return fmt.Errorf("dns provider %s: %w", p.cfg.DNSProvider, err)
	}
	return p.client.Challenge.SetDNS01Provider(provider)
}

func setEnv(kv map[string]string) func() {
	prev := map[string]string{}
	have := map[string]bool{}
	for k, v := range kv {
		if existing, ok := os.LookupEnv(k); ok {
			prev[k] = existing
			have[k] = true
		}
		_ = os.Setenv(k, v)
	}
	return func() {
		for k := range kv {
			if have[k] {
				_ = os.Setenv(k, prev[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
}

func safeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.ReplaceAll(d, string(filepath.Separator), "_")
	d = strings.ReplaceAll(d, "..", "_")
	if d == "" {
		d = "default"
	}
	return d
}

// CertExpiry returns the NotAfter of the leaf cert on disk for the given
// primary domain, or zero time if no cert is cached.
func (p *Provisioner) CertExpiry(domain string) (time.Time, error) {
	certFile, _ := p.CertPaths(domain)
	data, err := os.ReadFile(certFile)
	if err != nil {
		return time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return time.Time{}, errors.New("no PEM block in cert")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return leaf.NotAfter, nil
}

// acmeUser implements registration.User.
type acmeUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string                        { return u.Email }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

func loadOrCreateAccount(dataDir, email string) (*acmeUser, error) {
	keyPath := filepath.Join(dataDir, "account.key")
	regPath := filepath.Join(dataDir, "account.reg")

	user := &acmeUser{Email: email}

	if data, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, errors.New("invalid account.key PEM")
		}
		k, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse account.key: %w", err)
		}
		user.key = k
	} else if errors.Is(err, os.ErrNotExist) {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generating account key: %w", err)
		}
		user.key = k
		der, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		pemData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
		if err := os.WriteFile(keyPath, pemData, 0600); err != nil {
			return nil, fmt.Errorf("write account.key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("read account.key: %w", err)
	}

	if data, err := os.ReadFile(regPath); err == nil {
		reg := &registration.Resource{}
		if err := decodeJSON(data, reg); err != nil {
			return nil, fmt.Errorf("parse account.reg: %w", err)
		}
		user.Registration = reg
	}
	return user, nil
}

func saveAccount(dataDir string, user *acmeUser) error {
	if user.Registration == nil {
		return nil
	}
	regPath := filepath.Join(dataDir, "account.reg")
	data, err := encodeJSON(user.Registration)
	if err != nil {
		return err
	}
	return os.WriteFile(regPath, data, 0600)
}

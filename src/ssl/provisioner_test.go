package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateSelfSigned writes a self-signed cert+key pair for the given domain
// to the provided directory and returns the file paths.
func generateSelfSigned(t *testing.T, dir, domain string, notAfter time.Time) (certFile, keyFile string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		DNSNames:     []string{domain},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	certFile = filepath.Join(dir, domain+".crt")
	keyFile = filepath.Join(dir, domain+".key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(priv)
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return
}

func TestSafeDomain(t *testing.T) {
	cases := map[string]string{
		"":                "default",
		" Example.COM ":   "example.com",
		"foo/bar":         "foo_bar",
		"a..b":            "a_b",
		"good.example.io": "good.example.io",
	}
	for in, want := range cases {
		if got := safeDomain(in); got != want {
			t.Errorf("safeDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCertExpiry_ReadsLeaf(t *testing.T) {
	dir := t.TempDir()
	expiry := time.Now().Add(45 * 24 * time.Hour).Truncate(time.Second)
	generateSelfSigned(t, dir, "expiry.example.com", expiry)

	p := &Provisioner{cfg: ProvisionerConfig{DataDir: dir}}
	got, err := p.CertExpiry("expiry.example.com")
	if err != nil {
		t.Fatalf("CertExpiry: %v", err)
	}
	if !got.Equal(expiry) {
		t.Errorf("expiry = %v, want %v", got, expiry)
	}
}

func TestLoadStaticCert_PopulatesCache(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateSelfSigned(t, dir, "static.example.com", time.Now().Add(24*time.Hour))

	p := &Provisioner{
		cfg:   ProvisionerConfig{DataDir: dir},
		certs: map[string]*tls.Certificate{},
	}
	if err := p.LoadStaticCert(certFile, keyFile); err != nil {
		t.Fatalf("LoadStaticCert: %v", err)
	}
	got := p.CachedDomains()
	if len(got) == 0 {
		t.Fatal("cache empty after LoadStaticCert")
	}
	found := false
	for _, d := range got {
		if d == "static.example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("cache missing static.example.com, got %v", got)
	}
}

func TestSetEnv_RestoresPrev(t *testing.T) {
	const k = "CASMAN_TEST_SSL_ENV"
	t.Setenv(k, "before")
	restore := setEnv(map[string]string{k: "during"})
	if got := os.Getenv(k); got != "during" {
		t.Errorf("during setEnv: %q, want during", got)
	}
	restore()
	if got := os.Getenv(k); got != "before" {
		t.Errorf("after restore: %q, want before", got)
	}
}

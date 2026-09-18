package geoip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("Enabled should default to false")
	}
	if !cfg.ASN || !cfg.Country {
		t.Error("ASN and Country should default to true")
	}
	if cfg.City || cfg.WHOIS {
		t.Error("City and WHOIS should default to false")
	}
}

func TestNew_BuildsDenyList(t *testing.T) {
	g := New(Config{DenyCountries: []string{"CN", "RU"}})
	if !g.denyCountries["CN"] || !g.denyCountries["RU"] {
		t.Error("deny countries not populated")
	}
	if g.denyCountries["US"] {
		t.Error("unexpected country in deny list")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "present.mmdb")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(f) {
		t.Error("expected existing file to report true")
	}
	if fileExists(filepath.Join(dir, "missing.mmdb")) {
		t.Error("expected missing file to report false")
	}
}

func TestInit_Disabled(t *testing.T) {
	g := New(Config{Enabled: false})
	if err := g.Init(context.Background()); err != nil {
		t.Fatalf("Init disabled should not error: %v", err)
	}
	if g.IsAvailable() {
		t.Error("disabled GeoIP should not be available")
	}
}

func TestInit_MissingDir(t *testing.T) {
	g := New(Config{Enabled: true, Dir: ""})
	if err := g.Init(context.Background()); err == nil {
		t.Error("expected error when Dir is empty")
	}
}

func TestIsAvailable_InitiallyFalse(t *testing.T) {
	g := New(DefaultConfig())
	if g.IsAvailable() {
		t.Error("new GeoIP should not be available before Init")
	}
}

func TestLookup_NotAvailable(t *testing.T) {
	g := New(DefaultConfig())
	if _, err := g.Lookup("1.2.3.4"); err == nil {
		t.Error("expected error when not available")
	}
}

func TestLookup_InvalidIP(t *testing.T) {
	g := New(DefaultConfig())
	g.available = true
	if _, err := g.Lookup("not-an-ip"); err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestLookup_NoDatabasesLoaded(t *testing.T) {
	g := New(DefaultConfig())
	g.available = true
	res, err := g.Lookup("8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IP != "8.8.8.8" {
		t.Errorf("IP = %q", res.IP)
	}
	if res.CountryCode != "" || res.ASN != 0 {
		t.Error("expected empty lookup result with no databases loaded")
	}
}

func TestIsBlocked_NotAvailable(t *testing.T) {
	g := New(DefaultConfig())
	if g.IsBlocked("1.2.3.4") {
		t.Error("should not be blocked when not available")
	}
}

func TestGetCountry_NotAvailable(t *testing.T) {
	g := New(DefaultConfig())
	if g.GetCountry("1.2.3.4") != "" {
		t.Error("expected empty country when not available")
	}
}

func TestSetDenyCountries(t *testing.T) {
	g := New(Config{DenyCountries: []string{"CN"}})
	g.SetDenyCountries([]string{"US", "GB"})
	if g.denyCountries["CN"] {
		t.Error("old deny country should be cleared")
	}
	if !g.denyCountries["US"] || !g.denyCountries["GB"] {
		t.Error("new deny countries not applied")
	}
}

func TestClose_NoDatabases(t *testing.T) {
	g := New(DefaultConfig())
	if err := g.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if g.IsAvailable() {
		t.Error("should not be available after Close")
	}
}

func TestLastUpdate_ZeroInitially(t *testing.T) {
	g := New(DefaultConfig())
	if !g.LastUpdate().IsZero() {
		t.Error("expected zero LastUpdate before any update")
	}
}

func TestLoadDatabases_NoFilesPresent(t *testing.T) {
	dir := t.TempDir()
	g := New(Config{Dir: dir, ASN: true, Country: true, City: true})
	if err := g.loadDatabases(); err != nil {
		t.Fatalf("loadDatabases: %v", err)
	}
	if g.asnDB != nil || g.countryDB != nil || g.cityDB != nil {
		t.Error("expected no databases loaded when files are absent")
	}
}

func TestUpdate_MissingDir(t *testing.T) {
	g := New(Config{Dir: ""})
	if err := g.Update(context.Background()); err == nil {
		t.Error("expected error when Dir is empty")
	}
}

func TestDownloadFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mmdb-content"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "out.mmdb")
	if err := downloadFile(context.Background(), srv.URL, dst); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(data) != "mmdb-content" {
		t.Errorf("content = %q", string(data))
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dst := filepath.Join(dir, "out.mmdb")
	if err := downloadFile(context.Background(), srv.URL, dst); err == nil {
		t.Error("expected error on non-200 response")
	}
	if fileExists(dst) {
		t.Error("destination file should not exist after failed download")
	}
}

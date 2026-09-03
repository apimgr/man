// Package geoip provides GeoIP lookup functionality using MMDB databases.
// See AI.md PART 20 for details.
// Databases are downloaded on first run and updated weekly via scheduler.
package geoip

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// Database URLs from sapics/ip-location-db via jsdelivr CDN
const (
	ASNURL     = "https://cdn.jsdelivr.net/npm/@ip-location-db/asn-mmdb/asn.mmdb"
	CountryURL = "https://cdn.jsdelivr.net/npm/@ip-location-db/geo-whois-asn-country-mmdb/geo-whois-asn-country.mmdb"
	CityURL    = "https://cdn.jsdelivr.net/npm/@ip-location-db/dbip-city-mmdb/dbip-city-ipv4.mmdb"
	WHOISURL   = "https://cdn.jsdelivr.net/npm/@ip-location-db/geo-whois-asn-country-mmdb/geo-whois-asn-country.mmdb"
)

// Config holds GeoIP configuration.
type Config struct {
	Enabled       bool
	Dir           string
	DenyCountries []string
	ASN           bool
	Country       bool
	City          bool
	WHOIS         bool
}

// DefaultConfig returns default GeoIP configuration.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		Dir:     "",
		ASN:     true,
		Country: true,
		City:    false,
		WHOIS:   false,
	}
}

// ASNResult holds ASN lookup results.
type ASNResult struct {
	Number       uint   `maxminddb:"autonomous_system_number"`
	Organization string `maxminddb:"autonomous_system_organization"`
}

// CountryResult holds country lookup results.
type CountryResult struct {
	CountryCode string `maxminddb:"country_code"`
}

// CityResult holds city lookup results.
type CityResult struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

// LookupResult holds combined lookup results.
type LookupResult struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code,omitempty"`
	City        string  `json:"city,omitempty"`
	Region      string  `json:"region,omitempty"`
	PostalCode  string  `json:"postal_code,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	TimeZone    string  `json:"timezone,omitempty"`
	ASN         uint    `json:"asn,omitempty"`
	ASNOrg      string  `json:"asn_org,omitempty"`
	Blocked     bool    `json:"blocked,omitempty"`
}

// GeoIP handles GeoIP lookups.
type GeoIP struct {
	config        Config
	mu            sync.RWMutex
	asnDB         *maxminddb.Reader
	countryDB     *maxminddb.Reader
	cityDB        *maxminddb.Reader
	whoisDB       *maxminddb.Reader
	denyCountries map[string]bool
	lastUpdate    time.Time
	available     bool
}

// New creates a new GeoIP instance.
func New(cfg Config) *GeoIP {
	g := &GeoIP{
		config:        cfg,
		denyCountries: make(map[string]bool),
	}

	// Build deny list
	for _, c := range cfg.DenyCountries {
		g.denyCountries[c] = true
	}

	return g
}

// Init initializes GeoIP by loading or downloading databases.
func (g *GeoIP) Init(ctx context.Context) error {
	if !g.config.Enabled {
		log.Println("GeoIP: disabled")
		return nil
	}

	// Ensure directory exists
	if g.config.Dir == "" {
		return fmt.Errorf("GeoIP directory not configured")
	}

	if err := os.MkdirAll(g.config.Dir, 0755); err != nil {
		return fmt.Errorf("creating GeoIP directory: %w", err)
	}

	// Check if databases exist, download if needed
	needsDownload := false
	if g.config.ASN && !fileExists(filepath.Join(g.config.Dir, "asn.mmdb")) {
		needsDownload = true
	}
	if g.config.Country && !fileExists(filepath.Join(g.config.Dir, "country.mmdb")) {
		needsDownload = true
	}
	if g.config.City && !fileExists(filepath.Join(g.config.Dir, "city.mmdb")) {
		needsDownload = true
	}

	if needsDownload {
		log.Println("GeoIP: downloading databases...")
		if err := g.Update(ctx); err != nil {
			return fmt.Errorf("downloading databases: %w", err)
		}
	}

	// Load databases
	if err := g.loadDatabases(); err != nil {
		return fmt.Errorf("loading databases: %w", err)
	}

	g.mu.Lock()
	g.available = true
	g.mu.Unlock()

	log.Println("GeoIP: initialized successfully")
	return nil
}

// loadDatabases loads MMDB files into memory.
func (g *GeoIP) loadDatabases() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Close existing readers
	if g.asnDB != nil {
		g.asnDB.Close()
		g.asnDB = nil
	}
	if g.countryDB != nil {
		g.countryDB.Close()
		g.countryDB = nil
	}
	if g.cityDB != nil {
		g.cityDB.Close()
		g.cityDB = nil
	}
	if g.whoisDB != nil {
		g.whoisDB.Close()
		g.whoisDB = nil
	}

	var err error

	if g.config.ASN {
		path := filepath.Join(g.config.Dir, "asn.mmdb")
		if fileExists(path) {
			g.asnDB, err = maxminddb.Open(path)
			if err != nil {
				log.Printf("GeoIP: failed to load ASN database: %v", err)
			} else {
				log.Printf("GeoIP: loaded ASN database")
			}
		}
	}

	if g.config.Country {
		path := filepath.Join(g.config.Dir, "country.mmdb")
		if fileExists(path) {
			g.countryDB, err = maxminddb.Open(path)
			if err != nil {
				log.Printf("GeoIP: failed to load country database: %v", err)
			} else {
				log.Printf("GeoIP: loaded country database")
			}
		}
	}

	if g.config.City {
		path := filepath.Join(g.config.Dir, "city.mmdb")
		if fileExists(path) {
			g.cityDB, err = maxminddb.Open(path)
			if err != nil {
				log.Printf("GeoIP: failed to load city database: %v", err)
			} else {
				log.Printf("GeoIP: loaded city database")
			}
		}
	}

	return nil
}

// Update downloads the latest GeoIP databases.
func (g *GeoIP) Update(ctx context.Context) error {
	g.mu.Lock()
	dir := g.config.Dir
	g.mu.Unlock()

	if dir == "" {
		return fmt.Errorf("GeoIP directory not configured")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating GeoIP directory: %w", err)
	}

	var downloadErrors []error

	// Download ASN database
	if g.config.ASN {
		if err := downloadFile(ctx, ASNURL, filepath.Join(dir, "asn.mmdb")); err != nil {
			downloadErrors = append(downloadErrors, fmt.Errorf("ASN: %w", err))
		} else {
			log.Println("GeoIP: downloaded ASN database")
		}
	}

	// Download Country database
	if g.config.Country {
		if err := downloadFile(ctx, CountryURL, filepath.Join(dir, "country.mmdb")); err != nil {
			downloadErrors = append(downloadErrors, fmt.Errorf("Country: %w", err))
		} else {
			log.Println("GeoIP: downloaded Country database")
		}
	}

	// Download City database
	if g.config.City {
		if err := downloadFile(ctx, CityURL, filepath.Join(dir, "city.mmdb")); err != nil {
			downloadErrors = append(downloadErrors, fmt.Errorf("City: %w", err))
		} else {
			log.Println("GeoIP: downloaded City database")
		}
	}

	// Reload databases
	if err := g.loadDatabases(); err != nil {
		return fmt.Errorf("reloading databases: %w", err)
	}

	g.mu.Lock()
	g.lastUpdate = time.Now()
	g.available = true
	g.mu.Unlock()

	if len(downloadErrors) > 0 {
		return fmt.Errorf("some databases failed to download: %v", downloadErrors)
	}

	return nil
}

// Lookup performs a GeoIP lookup for the given IP address.
func (g *GeoIP) Lookup(ipStr string) (*LookupResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.available {
		return nil, fmt.Errorf("GeoIP not available")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	result := &LookupResult{IP: ipStr}

	// ASN lookup
	if g.asnDB != nil {
		var asn ASNResult
		if err := g.asnDB.Lookup(ip, &asn); err == nil {
			result.ASN = asn.Number
			result.ASNOrg = asn.Organization
		}
	}

	// Country lookup
	if g.countryDB != nil {
		var country CountryResult
		if err := g.countryDB.Lookup(ip, &country); err == nil {
			result.CountryCode = country.CountryCode
			// Check if country is blocked
			if g.denyCountries[country.CountryCode] {
				result.Blocked = true
			}
		}
	}

	// City lookup
	if g.cityDB != nil {
		var city CityResult
		if err := g.cityDB.Lookup(ip, &city); err == nil {
			if name, ok := city.City.Names["en"]; ok {
				result.City = name
			}
			if len(city.Subdivisions) > 0 {
				if name, ok := city.Subdivisions[0].Names["en"]; ok {
					result.Region = name
				}
			}
			if result.CountryCode == "" {
				result.CountryCode = city.Country.ISOCode
			}
			result.PostalCode = city.Postal.Code
			result.Latitude = city.Location.Latitude
			result.Longitude = city.Location.Longitude
			result.TimeZone = city.Location.TimeZone
		}
	}

	return result, nil
}

// IsBlocked checks if an IP's country is in the deny list.
func (g *GeoIP) IsBlocked(ipStr string) bool {
	result, err := g.Lookup(ipStr)
	if err != nil {
		return false
	}
	return result.Blocked
}

// GetCountry returns the country code for an IP address.
func (g *GeoIP) GetCountry(ipStr string) string {
	result, err := g.Lookup(ipStr)
	if err != nil {
		return ""
	}
	return result.CountryCode
}

// IsAvailable returns whether GeoIP is enabled and operational.
func (g *GeoIP) IsAvailable() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.available
}

// LastUpdate returns when the databases were last updated.
func (g *GeoIP) LastUpdate() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastUpdate
}

// SetDenyCountries updates the list of blocked countries.
func (g *GeoIP) SetDenyCountries(countries []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.denyCountries = make(map[string]bool)
	for _, c := range countries {
		g.denyCountries[c] = true
	}
}

// Close closes all database readers.
func (g *GeoIP) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.asnDB != nil {
		g.asnDB.Close()
	}
	if g.countryDB != nil {
		g.countryDB.Close()
	}
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.whoisDB != nil {
		g.whoisDB.Close()
	}

	g.available = false
	return nil
}

// downloadFile downloads a file from URL to the given path.
func downloadFile(ctx context.Context, url, path string) error {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Download to temp file first
	tmpPath := path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Rename to final path
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

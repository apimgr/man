// Package config handles configuration loading and management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Project constants
const (
	ProjectName = "casman"
	ProjectOrg  = "casapps"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Paths    PathConfig     `yaml:"-"`
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port             string `yaml:"port"`
	HTTPSPort        string `yaml:"https_port"`
	HTTPRedirectPort string `yaml:"http_redirect_port"`
	FQDN             string `yaml:"fqdn"`
	Address          string `yaml:"address"`
	Mode             string `yaml:"mode"`
	AdminPath        string `yaml:"admin_path"`

	Branding BrandingConfig `yaml:"branding"`
	SSL      SSLConfig      `yaml:"ssl"`
	Metrics  *MetricsConfig `yaml:"metrics"`
	GeoIP    *GeoIPConfig   `yaml:"geoip"`
	Backup   *BackupConfig  `yaml:"backup"`
	Tor      *TorConfig     `yaml:"tor"`
}

// TorConfig holds Tor hidden-service configuration per AI.md PART 32.
type TorConfig struct {
	// Binary is the path to a tor executable; empty = auto-detect via PATH.
	Binary string `yaml:"binary"`
	// VirtualPort is the port advertised on the .onion (default 80).
	VirtualPort int `yaml:"virtual_port"`
	// BootstrapTimeout caps how long Start() waits for Tor to bootstrap.
	BootstrapTimeout string `yaml:"bootstrap_timeout"`
	// SafeLogging enables Tor's SafeLogging directive.
	SafeLogging bool `yaml:"safe_logging"`
	// UseNetwork routes the server's outbound HTTP through the bundled
	// Tor SOCKS5 proxy. Per-user override (PART 32 + PART 34) is out of
	// scope for casman because it does not enable multi-user accounts.
	UseNetwork bool `yaml:"use_network"`
}

// MetricsConfig holds Prometheus metrics configuration per PART 21.
type MetricsConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Endpoint       string `yaml:"endpoint"`
	Token          string `yaml:"token"`
	IncludeSystem  bool   `yaml:"include_system"`
	IncludeRuntime bool   `yaml:"include_runtime"`
}

// GeoIPConfig holds GeoIP configuration per PART 20.
type GeoIPConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Dir           string   `yaml:"dir"`
	DenyCountries []string `yaml:"deny_countries"`
	Databases     struct {
		ASN     bool `yaml:"asn"`
		Country bool `yaml:"country"`
		City    bool `yaml:"city"`
		WHOIS   bool `yaml:"whois"`
	} `yaml:"databases"`
}

// BackupConfig holds backup configuration per PART 22.
type BackupConfig struct {
	Dir        string `yaml:"dir"`
	Retention  struct {
		MaxBackups  int `yaml:"max_backups"`
		KeepWeekly  int `yaml:"keep_weekly"`
		KeepMonthly int `yaml:"keep_monthly"`
		KeepYearly  int `yaml:"keep_yearly"`
	} `yaml:"retention"`
	Encryption struct {
		Enabled      bool   `yaml:"enabled"`
		PasswordHint string `yaml:"password_hint"`
	} `yaml:"encryption"`
	Compliance bool `yaml:"compliance"`
}

// BrandingConfig holds branding settings.
type BrandingConfig struct {
	Title       string `yaml:"title"`
	Tagline     string `yaml:"tagline"`
	Description string `yaml:"description"`
}

// SSLConfig holds SSL/TLS settings.
type SSLConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Cert       string `yaml:"cert"`
	Key        string `yaml:"key"`
	MinVersion string `yaml:"min_version"`
	// DataDir is the on-disk root for ACME-issued certs and the account key.
	// Empty defaults to {data_dir}/ssl per AI.md PART 15.
	DataDir string `yaml:"data_dir"`

	LetsEncrypt LetsEncryptConfig `yaml:"letsencrypt"`
}

// LetsEncryptConfig holds Let's Encrypt settings.
type LetsEncryptConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Email     string `yaml:"email"`
	Challenge string `yaml:"challenge"`
	Staging   bool   `yaml:"staging"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	Path   string `yaml:"path"`
	DSN    string `yaml:"dsn"`
}

// PathConfig holds resolved directory paths.
type PathConfig struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	LogDir    string
	BackupDir string
}

// Save writes the current configuration back to {config_dir}/server.yml using
// an atomic temp+rename so a partial write cannot corrupt the file. The file
// permission is 0600 because the SSL block can hold sensitive defaults.
func (c *Config) Save() error {
	if c.Paths.ConfigDir == "" {
		return fmt.Errorf("config: empty ConfigDir")
	}
	configFile := filepath.Join(c.Paths.ConfigDir, "server.yml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp, err := os.CreateTemp(c.Paths.ConfigDir, "server.yml.*.tmp")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, configFile); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Load loads configuration from the specified directories.
func Load(configDir, dataDir string) (*Config, error) {
	// Resolve paths
	paths := resolvePaths(configDir, dataDir)

	// Ensure directories exist
	if err := ensureDirs(paths); err != nil {
		return nil, err
	}

	// Default configuration
	cfg := defaultConfig(paths)

	// Load config file if exists
	configFile := filepath.Join(paths.ConfigDir, "server.yml")
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Store resolved paths
	cfg.Paths = paths

	// Resolve path variables in config
	cfg.Database.Path = resolvePlaceholders(cfg.Database.Path, paths)

	return cfg, nil
}

// resolvePaths determines the appropriate directories based on context.
func resolvePaths(configDir, dataDir string) PathConfig {
	isRoot := os.Getuid() == 0
	isContainer := isRunningInContainer()

	paths := PathConfig{}

	// Config directory
	if configDir != "" {
		paths.ConfigDir = configDir
	} else if isContainer {
		paths.ConfigDir = "/config"
	} else if isRoot {
		paths.ConfigDir = filepath.Join("/etc", ProjectOrg, ProjectName)
	} else {
		paths.ConfigDir = filepath.Join(userConfigDir(), ProjectOrg, ProjectName)
	}

	// Data directory
	if dataDir != "" {
		paths.DataDir = dataDir
	} else if isContainer {
		paths.DataDir = "/data"
	} else if isRoot {
		paths.DataDir = filepath.Join("/var/lib", ProjectOrg, ProjectName)
	} else {
		paths.DataDir = filepath.Join(userDataDir(), ProjectOrg, ProjectName)
	}

	// Cache directory
	if isContainer {
		paths.CacheDir = "/data/cache"
	} else if isRoot {
		paths.CacheDir = filepath.Join("/var/cache", ProjectOrg, ProjectName)
	} else {
		paths.CacheDir = filepath.Join(userCacheDir(), ProjectOrg, ProjectName)
	}

	// Log directory
	if isContainer {
		paths.LogDir = "/data/logs"
	} else if isRoot {
		paths.LogDir = filepath.Join("/var/log", ProjectOrg, ProjectName)
	} else {
		paths.LogDir = filepath.Join(userDataDir(), "log", ProjectOrg, ProjectName)
	}

	// Backup directory
	if isContainer {
		paths.BackupDir = "/data/backups"
	} else if isRoot {
		paths.BackupDir = filepath.Join("/var/lib/Backups", ProjectOrg, ProjectName)
	} else {
		paths.BackupDir = filepath.Join(userDataDir(), "Backups", ProjectOrg, ProjectName)
	}

	return paths
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig(paths PathConfig) *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}

	return &Config{
		Server: ServerConfig{
			Port:             "",
			HTTPSPort:        "443",
			HTTPRedirectPort: "80",
			FQDN:             hostname,
			Address:          "[::]",
			Mode:             "production",
			AdminPath:        "admin",
			Branding: BrandingConfig{
				Title:       "casman",
				Tagline:     "Universal Man Pages",
				Description: "Man pages from BSD, macOS, Linux, and more",
			},
			SSL: SSLConfig{
				Enabled:    false,
				MinVersion: "TLS1.2",
				LetsEncrypt: LetsEncryptConfig{
					Challenge: "http-01",
				},
			},
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   "{data_dir}/db/server.db",
		},
		Paths: paths,
	}
}

// ensureDirs creates required directories if they don't exist.
func ensureDirs(paths PathConfig) error {
	dirs := []string{
		paths.ConfigDir,
		paths.DataDir,
		paths.CacheDir,
		paths.LogDir,
		filepath.Join(paths.DataDir, "db"),
	}

	isRoot := os.Getuid() == 0
	perm := os.FileMode(0700)
	if isRoot {
		perm = 0755
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return nil
}

// resolvePlaceholders replaces path placeholders with actual values.
func resolvePlaceholders(path string, paths PathConfig) string {
	replacer := strings.NewReplacer(
		"{config_dir}", paths.ConfigDir,
		"{data_dir}", paths.DataDir,
		"{cache_dir}", paths.CacheDir,
		"{log_dir}", paths.LogDir,
		"{backup_dir}", paths.BackupDir,
	)
	return replacer.Replace(path)
}

// isRunningInContainer checks if running inside a container.
func isRunningInContainer() bool {
	// Check for /.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// Check for container environment variable
	if os.Getenv("container") != "" {
		return true
	}
	return false
}

// userConfigDir returns the user's config directory.
func userConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	case "windows":
		return os.Getenv("APPDATA")
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return xdg
		}
		return filepath.Join(os.Getenv("HOME"), ".config")
	}
}

// userDataDir returns the user's data directory.
func userDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	case "windows":
		return os.Getenv("LOCALAPPDATA")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return xdg
		}
		return filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
}

// userCacheDir returns the user's cache directory.
func userCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "windows":
		return os.Getenv("TEMP")
	default:
		if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
			return xdg
		}
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
}


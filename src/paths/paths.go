// Package paths provides OS-specific path resolution for casman.
// See AI.md PART 4 for details.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// Project constants
const (
	ProjectName = "casman"
	ProjectOrg  = "casapps"
)

// Paths holds all resolved directory paths.
type Paths struct {
	ConfigDir   string
	ConfigFile  string
	DataDir     string
	CacheDir    string
	LogDir      string
	LogFile     string
	BackupDir   string
	PIDFile     string
	SSLDir      string
	SecurityDir string
	DBDir       string
}

// Resolve determines the appropriate paths based on OS and privilege level.
// CLI overrides can be passed to override auto-detection.
// See AI.md PART 4 for details.
func Resolve(configOverride, dataOverride string) Paths {
	isPrivileged := os.Getuid() == 0
	isContainer := isRunningInContainer()

	if isContainer {
		return resolveContainer()
	}

	switch runtime.GOOS {
	case "darwin":
		return resolveDarwin(isPrivileged, configOverride, dataOverride)
	case "windows":
		return resolveWindows(isPrivileged, configOverride, dataOverride)
	case "freebsd", "openbsd", "netbsd", "dragonfly":
		return resolveBSD(isPrivileged, configOverride, dataOverride)
	default:
		return resolveLinux(isPrivileged, configOverride, dataOverride)
	}
}

// resolveContainer returns container-specific paths.
func resolveContainer() Paths {
	return Paths{
		ConfigDir:   filepath.Join("/config", ProjectName),
		ConfigFile:  filepath.Join("/config", ProjectName, "server.yml"),
		DataDir:     filepath.Join("/data", ProjectName),
		CacheDir:    filepath.Join("/data", ProjectName, "cache"),
		LogDir:      filepath.Join("/data/log", ProjectName),
		LogFile:     filepath.Join("/data/log", ProjectName, "server.log"),
		BackupDir:   filepath.Join("/data/backups", ProjectName),
		PIDFile:     filepath.Join("/data", ProjectName, ProjectName+".pid"),
		SSLDir:      filepath.Join("/config", ProjectName, "ssl"),
		SecurityDir: filepath.Join("/config", ProjectName, "security"),
		DBDir:       "/data/db",
	}
}

// resolveLinux returns Linux-specific paths.
func resolveLinux(isPrivileged bool, configOverride, dataOverride string) Paths {
	if isPrivileged {
		return linuxPrivileged(configOverride, dataOverride)
	}
	return linuxUser(configOverride, dataOverride)
}

func linuxPrivileged(configOverride, dataOverride string) Paths {
	configDir := filepath.Join("/etc", ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join("/var/lib", ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join("/var/cache", ProjectOrg, ProjectName),
		LogDir:      filepath.Join("/var/log", ProjectOrg, ProjectName),
		LogFile:     filepath.Join("/var/log", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join("/mnt/Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join("/var/run", ProjectOrg, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(dataDir, "db"),
	}
}

func linuxUser(configOverride, dataOverride string) Paths {
	home := os.Getenv("HOME")
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	xdgData := os.Getenv("XDG_DATA_HOME")
	xdgCache := os.Getenv("XDG_CACHE_HOME")

	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	if xdgData == "" {
		xdgData = filepath.Join(home, ".local", "share")
	}
	if xdgCache == "" {
		xdgCache = filepath.Join(home, ".cache")
	}

	configDir := filepath.Join(xdgConfig, ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join(xdgData, ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join(xdgCache, ProjectOrg, ProjectName),
		LogDir:      filepath.Join(home, ".local", "log", ProjectOrg, ProjectName),
		LogFile:     filepath.Join(home, ".local", "log", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join(xdgData, "Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join(dataDir, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(dataDir, "db"),
	}
}

// resolveDarwin returns macOS-specific paths.
func resolveDarwin(isPrivileged bool, configOverride, dataOverride string) Paths {
	if isPrivileged {
		return darwinPrivileged(configOverride, dataOverride)
	}
	return darwinUser(configOverride, dataOverride)
}

func darwinPrivileged(configOverride, dataOverride string) Paths {
	configDir := filepath.Join("/Library/Application Support", ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join("/Library/Application Support", ProjectOrg, ProjectName, "data")
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join("/Library/Caches", ProjectOrg, ProjectName),
		LogDir:      filepath.Join("/Library/Logs", ProjectOrg, ProjectName),
		LogFile:     filepath.Join("/Library/Logs", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join("/Library/Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join("/var/run", ProjectOrg, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(configDir, "db"),
	}
}

func darwinUser(configOverride, dataOverride string) Paths {
	home := os.Getenv("HOME")

	configDir := filepath.Join(home, "Library/Application Support", ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join(home, "Library/Application Support", ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join(home, "Library/Caches", ProjectOrg, ProjectName),
		LogDir:      filepath.Join(home, "Library/Logs", ProjectOrg, ProjectName),
		LogFile:     filepath.Join(home, "Library/Logs", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join(home, "Library/Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join(dataDir, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(dataDir, "db"),
	}
}

// resolveBSD returns BSD-specific paths.
func resolveBSD(isPrivileged bool, configOverride, dataOverride string) Paths {
	if isPrivileged {
		return bsdPrivileged(configOverride, dataOverride)
	}
	return bsdUser(configOverride, dataOverride)
}

func bsdPrivileged(configOverride, dataOverride string) Paths {
	configDir := filepath.Join("/usr/local/etc", ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join("/var/db", ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join("/var/cache", ProjectOrg, ProjectName),
		LogDir:      filepath.Join("/var/log", ProjectOrg, ProjectName),
		LogFile:     filepath.Join("/var/log", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join("/var/backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join("/var/run", ProjectOrg, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(dataDir, "db"),
	}
}

func bsdUser(configOverride, dataOverride string) Paths {
	home := os.Getenv("HOME")

	configDir := filepath.Join(home, ".config", ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join(home, ".local", "share", ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join(home, ".cache", ProjectOrg, ProjectName),
		LogDir:      filepath.Join(home, ".local", "log", ProjectOrg, ProjectName),
		LogFile:     filepath.Join(home, ".local", "log", ProjectOrg, ProjectName, "server.log"),
		BackupDir:   filepath.Join(home, ".local", "share", "Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join(dataDir, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(dataDir, "db"),
	}
}

// resolveWindows returns Windows-specific paths.
func resolveWindows(isPrivileged bool, configOverride, dataOverride string) Paths {
	if isPrivileged {
		return windowsPrivileged(configOverride, dataOverride)
	}
	return windowsUser(configOverride, dataOverride)
}

func windowsPrivileged(configOverride, dataOverride string) Paths {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}

	configDir := filepath.Join(programData, ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join(programData, ProjectOrg, ProjectName, "data")
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join(programData, ProjectOrg, ProjectName, "cache"),
		LogDir:      filepath.Join(programData, ProjectOrg, ProjectName, "logs"),
		LogFile:     filepath.Join(programData, ProjectOrg, ProjectName, "logs", "server.log"),
		BackupDir:   filepath.Join(programData, "Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join(dataDir, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(programData, ProjectOrg, ProjectName, "db"),
	}
}

func windowsUser(configOverride, dataOverride string) Paths {
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")

	configDir := filepath.Join(appData, ProjectOrg, ProjectName)
	if configOverride != "" {
		configDir = configOverride
	}

	dataDir := filepath.Join(localAppData, ProjectOrg, ProjectName)
	if dataOverride != "" {
		dataDir = dataOverride
	}

	return Paths{
		ConfigDir:   configDir,
		ConfigFile:  filepath.Join(configDir, "server.yml"),
		DataDir:     dataDir,
		CacheDir:    filepath.Join(localAppData, ProjectOrg, ProjectName, "cache"),
		LogDir:      filepath.Join(localAppData, ProjectOrg, ProjectName, "logs"),
		LogFile:     filepath.Join(localAppData, ProjectOrg, ProjectName, "logs", "server.log"),
		BackupDir:   filepath.Join(localAppData, "Backups", ProjectOrg, ProjectName),
		PIDFile:     filepath.Join(dataDir, ProjectName+".pid"),
		SSLDir:      filepath.Join(configDir, "ssl"),
		SecurityDir: filepath.Join(configDir, "security"),
		DBDir:       filepath.Join(localAppData, ProjectOrg, ProjectName, "db"),
	}
}

// isRunningInContainer checks if running inside a container.
func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if os.Getenv("container") != "" {
		return true
	}
	return false
}

// EnsureDirectories creates all required directories.
func (p Paths) EnsureDirectories() error {
	dirs := []string{
		p.ConfigDir,
		p.DataDir,
		p.CacheDir,
		p.LogDir,
		p.BackupDir,
		p.SSLDir,
		p.SecurityDir,
		p.DBDir,
	}

	perm := os.FileMode(0700)
	if os.Getuid() == 0 {
		perm = 0755
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, perm); err != nil {
			return err
		}
	}

	return nil
}

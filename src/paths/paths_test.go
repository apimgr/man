package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveContainer(t *testing.T) {
	p := resolveContainer()
	if p.ConfigDir != filepath.Join("/config", ProjectName) {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
	if p.DataDir != filepath.Join("/data", ProjectName) {
		t.Errorf("DataDir = %q", p.DataDir)
	}
	if p.DBDir != "/data/db" {
		t.Errorf("DBDir = %q", p.DBDir)
	}
}

func TestLinuxPrivileged_Defaults(t *testing.T) {
	p := linuxPrivileged("", "")
	if p.ConfigDir != filepath.Join("/etc", ProjectOrg, ProjectName) {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
	if p.DataDir != filepath.Join("/var/lib", ProjectOrg, ProjectName) {
		t.Errorf("DataDir = %q", p.DataDir)
	}
	if p.SSLDir != filepath.Join(p.ConfigDir, "ssl") {
		t.Errorf("SSLDir = %q", p.SSLDir)
	}
}

func TestLinuxPrivileged_Overrides(t *testing.T) {
	p := linuxPrivileged("/custom/config", "/custom/data")
	if p.ConfigDir != "/custom/config" {
		t.Errorf("ConfigDir override not applied: %q", p.ConfigDir)
	}
	if p.DataDir != "/custom/data" {
		t.Errorf("DataDir override not applied: %q", p.DataDir)
	}
	if p.DBDir != filepath.Join("/custom/data", "db") {
		t.Errorf("DBDir = %q", p.DBDir)
	}
}

func TestLinuxUser_UsesXDGVars(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("XDG_CACHE_HOME", "/xdg/cache")

	p := linuxUser("", "")
	if p.ConfigDir != filepath.Join("/xdg/config", ProjectOrg, ProjectName) {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
	if p.DataDir != filepath.Join("/xdg/data", ProjectOrg, ProjectName) {
		t.Errorf("DataDir = %q", p.DataDir)
	}
	if p.CacheDir != filepath.Join("/xdg/cache", ProjectOrg, ProjectName) {
		t.Errorf("CacheDir = %q", p.CacheDir)
	}
}

func TestLinuxUser_FallsBackToHomeWhenNoXDG(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	p := linuxUser("", "")
	if p.ConfigDir != filepath.Join("/home/tester", ".config", ProjectOrg, ProjectName) {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
	if p.DataDir != filepath.Join("/home/tester", ".local", "share", ProjectOrg, ProjectName) {
		t.Errorf("DataDir = %q", p.DataDir)
	}
}

func TestLinuxUser_Overrides(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	p := linuxUser("/override/cfg", "/override/data")
	if p.ConfigDir != "/override/cfg" {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
	if p.DataDir != "/override/data" {
		t.Errorf("DataDir = %q", p.DataDir)
	}
}

func TestDarwinPrivilegedAndUser(t *testing.T) {
	priv := darwinPrivileged("", "")
	if priv.ConfigDir != filepath.Join("/Library/Application Support", ProjectOrg, ProjectName) {
		t.Errorf("priv ConfigDir = %q", priv.ConfigDir)
	}

	t.Setenv("HOME", "/Users/tester")
	user := darwinUser("", "")
	if user.ConfigDir != filepath.Join("/Users/tester", "Library/Application Support", ProjectOrg, ProjectName) {
		t.Errorf("user ConfigDir = %q", user.ConfigDir)
	}
}

func TestBSDPrivilegedAndUser(t *testing.T) {
	priv := bsdPrivileged("", "")
	if priv.ConfigDir != filepath.Join("/usr/local/etc", ProjectOrg, ProjectName) {
		t.Errorf("priv ConfigDir = %q", priv.ConfigDir)
	}

	t.Setenv("HOME", "/home/bsduser")
	user := bsdUser("", "")
	if user.ConfigDir != filepath.Join("/home/bsduser", ".config", ProjectOrg, ProjectName) {
		t.Errorf("user ConfigDir = %q", user.ConfigDir)
	}
}

func TestWindowsPrivilegedAndUser(t *testing.T) {
	t.Setenv("ProgramData", `C:\ProgramData`)
	priv := windowsPrivileged("", "")
	if priv.ConfigDir != filepath.Join(`C:\ProgramData`, ProjectOrg, ProjectName) {
		t.Errorf("priv ConfigDir = %q", priv.ConfigDir)
	}

	t.Setenv("APPDATA", `C:\Users\tester\AppData\Roaming`)
	t.Setenv("LOCALAPPDATA", `C:\Users\tester\AppData\Local`)
	user := windowsUser("", "")
	if user.ConfigDir != filepath.Join(`C:\Users\tester\AppData\Roaming`, ProjectOrg, ProjectName) {
		t.Errorf("user ConfigDir = %q", user.ConfigDir)
	}
	if user.DataDir != filepath.Join(`C:\Users\tester\AppData\Local`, ProjectOrg, ProjectName) {
		t.Errorf("user DataDir = %q", user.DataDir)
	}
}

func TestWindowsPrivileged_DefaultProgramData(t *testing.T) {
	t.Setenv("ProgramData", "")
	p := windowsPrivileged("", "")
	if p.ConfigDir != filepath.Join(`C:\ProgramData`, ProjectOrg, ProjectName) {
		t.Errorf("ConfigDir = %q", p.ConfigDir)
	}
}

func TestResolveDispatchesByGOOS(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	// Just verify it returns a populated, non-empty Paths struct without
	// panicking, regardless of which OS branch or container branch triggers.
	p := Resolve("", "")
	if p.ConfigDir == "" || p.DataDir == "" || p.ConfigFile == "" {
		t.Errorf("Resolve() returned incomplete Paths: %+v", p)
	}
}

func TestResolveLinux_PrivilegedVsUser(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	priv := resolveLinux(true, "", "")
	user := resolveLinux(false, "", "")
	if priv.ConfigDir == user.ConfigDir {
		t.Error("privileged and user linux paths should differ")
	}
}

func TestResolveDarwin_PrivilegedVsUser(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	priv := resolveDarwin(true, "", "")
	user := resolveDarwin(false, "", "")
	if priv.ConfigDir == user.ConfigDir {
		t.Error("privileged and user darwin paths should differ")
	}
}

func TestResolveBSD_PrivilegedVsUser(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	priv := resolveBSD(true, "", "")
	user := resolveBSD(false, "", "")
	if priv.ConfigDir == user.ConfigDir {
		t.Error("privileged and user bsd paths should differ")
	}
}

func TestResolveWindows_PrivilegedVsUser(t *testing.T) {
	t.Setenv("ProgramData", `C:\ProgramData`)
	t.Setenv("APPDATA", `C:\Users\tester\AppData\Roaming`)
	t.Setenv("LOCALAPPDATA", `C:\Users\tester\AppData\Local`)
	priv := resolveWindows(true, "", "")
	user := resolveWindows(false, "", "")
	if priv.ConfigDir == user.ConfigDir {
		t.Error("privileged and user windows paths should differ")
	}
}

func TestIsRunningInContainer_EnvVar(t *testing.T) {
	t.Setenv("container", "podman")
	if !isRunningInContainer() {
		t.Error("expected true when container env var set")
	}
}

func TestEnsureDirectories(t *testing.T) {
	base := t.TempDir()
	p := Paths{
		ConfigDir:   filepath.Join(base, "config"),
		DataDir:     filepath.Join(base, "data"),
		CacheDir:    filepath.Join(base, "cache"),
		LogDir:      filepath.Join(base, "log"),
		BackupDir:   filepath.Join(base, "backup"),
		SSLDir:      filepath.Join(base, "ssl"),
		SecurityDir: filepath.Join(base, "security"),
		DBDir:       filepath.Join(base, "db"),
	}

	if err := p.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories: %v", err)
	}

	for _, dir := range []string{p.ConfigDir, p.DataDir, p.CacheDir, p.LogDir, p.BackupDir, p.SSLDir, p.SecurityDir, p.DBDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("dir %q not created: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}
}

package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupDirs(t *testing.T) (configDir, dataDir, backupDir string) {
	t.Helper()
	base := t.TempDir()
	configDir = filepath.Join(base, "config")
	dataDir = filepath.Join(base, "data")
	backupDir = filepath.Join(base, "backup")

	if err := os.MkdirAll(filepath.Join(dataDir, "db"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "server.yml"), []byte("port: 8080\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "db", "server.db"), []byte("fake-db-data"), 0644); err != nil {
		t.Fatal(err)
	}
	return configDir, dataDir, backupDir
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Retention.MaxBackups != 1 {
		t.Errorf("MaxBackups = %d, want 1", cfg.Retention.MaxBackups)
	}
	if cfg.Encryption.Enabled {
		t.Error("Encryption should default to disabled")
	}
}

func TestBackup_Unencrypted_RoundTrip(t *testing.T) {
	configDir, dataDir, backupDir := setupDirs(t)
	cfg := DefaultConfig()
	cfg.Dir = backupDir
	m := New(cfg, "1.0.0", configDir, dataDir)

	path, err := m.Backup(context.Background(), "", "", "tester")
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if filepath.Ext(path) == ".enc" {
		t.Error("unencrypted backup should not have .enc suffix")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backup file not created: %v", err)
	}

	restoreConfigDir := t.TempDir()
	restoreDataDir := t.TempDir()
	rm := New(cfg, "1.0.0", restoreConfigDir, restoreDataDir)
	if err := rm.Restore(context.Background(), path, ""); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	restored, err := os.ReadFile(filepath.Join(restoreConfigDir, "server.yml"))
	if err != nil {
		t.Fatalf("restored server.yml missing: %v", err)
	}
	if string(restored) != "port: 8080\n" {
		t.Errorf("restored content mismatch: %q", string(restored))
	}
}

func TestBackup_Encrypted_RoundTrip(t *testing.T) {
	configDir, dataDir, backupDir := setupDirs(t)
	cfg := DefaultConfig()
	cfg.Dir = backupDir
	m := New(cfg, "1.0.0", configDir, dataDir)

	path, err := m.Backup(context.Background(), "", "hunter2", "tester")
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if filepath.Ext(path) != ".enc" {
		t.Errorf("encrypted backup should have .enc suffix, got %q", path)
	}

	restoreConfigDir := t.TempDir()
	restoreDataDir := t.TempDir()
	rm := New(cfg, "1.0.0", restoreConfigDir, restoreDataDir)

	if err := rm.Restore(context.Background(), path, "wrongpassword"); err == nil {
		t.Error("expected error restoring with wrong password")
	}

	if err := rm.Restore(context.Background(), path, "hunter2"); err != nil {
		t.Fatalf("Restore with correct password: %v", err)
	}
	if _, err := os.Stat(filepath.Join(restoreConfigDir, "server.yml")); err != nil {
		t.Fatalf("restored file missing: %v", err)
	}
}

func TestBackup_ComplianceRequiresPassword(t *testing.T) {
	configDir, dataDir, backupDir := setupDirs(t)
	cfg := DefaultConfig()
	cfg.Dir = backupDir
	cfg.Compliance = true
	m := New(cfg, "1.0.0", configDir, dataDir)

	if _, err := m.Backup(context.Background(), "", "", "tester"); err == nil {
		t.Error("expected error when compliance mode set without password")
	}

	if _, err := m.Backup(context.Background(), "", "hunter2", "tester"); err != nil {
		t.Errorf("compliance mode with password should succeed: %v", err)
	}
}

func TestBackup_NoFilesToBackup(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	backupDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Dir = backupDir
	m := New(cfg, "1.0.0", configDir, dataDir)

	if _, err := m.Backup(context.Background(), "", "", "tester"); err == nil {
		t.Error("expected error when there are no files to back up")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	m := &Manager{}
	plaintext := []byte("super secret backup contents")

	ciphertext, err := m.encrypt(plaintext, "correct-password")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := m.decrypt(ciphertext, "correct-password")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}

	if _, err := m.decrypt(ciphertext, "wrong-password"); err == nil {
		t.Error("expected error decrypting with wrong password")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	m := &Manager{}
	if _, err := m.decrypt([]byte("short"), "password"); err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestListBackups_EmptyDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = t.TempDir()
	m := New(cfg, "1.0.0", t.TempDir(), t.TempDir())

	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected no backups, got %d", len(backups))
	}
}

func TestListBackups_MissingDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = filepath.Join(t.TempDir(), "does-not-exist")
	m := New(cfg, "1.0.0", t.TempDir(), t.TempDir())

	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups on missing dir should not error: %v", err)
	}
	if backups != nil {
		t.Errorf("expected nil backups, got %v", backups)
	}
}

func TestListBackups_FiltersAndSorts(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"casman_backup_2024-01-01_000000.tar.gz",
		"casman_backup_2024-06-01_000000.tar.gz.enc",
		"not-a-backup.tar.gz",
		"casman_backup_ignored.txt",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := DefaultConfig()
	cfg.Dir = dir
	m := New(cfg, "1.0.0", t.TempDir(), t.TempDir())

	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 valid backups, got %d: %+v", len(backups), backups)
	}
	for _, b := range backups {
		if b.Name == "not-a-backup.tar.gz" || b.Name == "casman_backup_ignored.txt" {
			t.Errorf("unexpected backup included: %s", b.Name)
		}
	}
}

func TestApplyRetention_DeletesOldest(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"casman_backup_a.tar.gz",
		"casman_backup_b.tar.gz",
		"casman_backup_c.tar.gz",
	}
	for i, n := range files {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		modTime := time.Now().Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(p, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	cfg := DefaultConfig()
	cfg.Dir = dir
	cfg.Retention.MaxBackups = 1
	m := New(cfg, "1.0.0", t.TempDir(), t.TempDir())

	if err := m.ApplyRetention(); err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}

	remaining, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 backup remaining, got %d", len(remaining))
	}
	if remaining[0].Name != "casman_backup_c.tar.gz" {
		t.Errorf("expected newest backup retained, got %s", remaining[0].Name)
	}
}

func TestApplyRetention_SkipsIncrementals(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"casman_backup_full.tar.gz",
		"casman_backup_2024-01-01-daily.tar.gz",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := DefaultConfig()
	cfg.Dir = dir
	cfg.Retention.MaxBackups = 1
	m := New(cfg, "1.0.0", t.TempDir(), t.TempDir())

	if err := m.ApplyRetention(); err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}

	remaining, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("expected both backups retained (incremental skipped from count), got %d", len(remaining))
	}
}

func TestFileExistsAndDirExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(f) {
		t.Error("expected fileExists true")
	}
	if fileExists(dir) {
		t.Error("expected fileExists false for a directory")
	}
	if !dirExists(dir) {
		t.Error("expected dirExists true")
	}
	if dirExists(f) {
		t.Error("expected dirExists false for a file")
	}
	if fileExists(filepath.Join(dir, "missing")) {
		t.Error("expected fileExists false for missing path")
	}
}

func TestVerifyBackup_MissingFile(t *testing.T) {
	m := &Manager{}
	if err := m.verifyBackup("/nonexistent/path.tar.gz", ""); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestVerifyBackup_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.tar.gz")
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatal(err)
	}
	m := &Manager{}
	if err := m.verifyBackup(p, ""); err == nil {
		t.Error("expected error for empty file")
	}
}

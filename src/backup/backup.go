// Package backup implements backup and restore functionality.
// See AI.md PART 22 for details.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Config holds backup configuration.
type Config struct {
	Dir        string
	Retention  RetentionConfig
	Encryption EncryptionConfig
	Compliance bool
}

// RetentionConfig holds retention settings.
type RetentionConfig struct {
	MaxBackups  int
	KeepWeekly  int
	KeepMonthly int
	KeepYearly  int
}

// EncryptionConfig holds encryption settings.
type EncryptionConfig struct {
	Enabled      bool
	PasswordHint string
}

// DefaultConfig returns default backup configuration.
func DefaultConfig() Config {
	return Config{
		Retention: RetentionConfig{
			MaxBackups:  1,
			KeepWeekly:  0,
			KeepMonthly: 0,
			KeepYearly:  0,
		},
		Encryption: EncryptionConfig{
			Enabled: false,
		},
		Compliance: false,
	}
}

// Manifest holds backup metadata.
type Manifest struct {
	Version          string    `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
	AppVersion       string    `json:"app_version"`
	Contents         []string  `json:"contents"`
	Encrypted        bool      `json:"encrypted"`
	EncryptionMethod string    `json:"encryption_method,omitempty"`
	Checksum         string    `json:"checksum"`
}

// Manager handles backup operations.
type Manager struct {
	config     Config
	appVersion string
	configDir  string
	dataDir    string
}

// New creates a new backup Manager.
func New(cfg Config, appVersion, configDir, dataDir string) *Manager {
	return &Manager{
		config:     cfg,
		appVersion: appVersion,
		configDir:  configDir,
		dataDir:    dataDir,
	}
}

// Dir returns the configured backup directory. Used by admin handlers to
// resolve user-supplied filenames to full paths inside the backup root.
func (m *Manager) Dir() string { return m.config.Dir }

// Backup creates a backup archive.
func (m *Manager) Backup(ctx context.Context, filename, password, createdBy string) (string, error) {
	// Check compliance mode
	if m.config.Compliance && password == "" {
		return "", fmt.Errorf("compliance mode requires encrypted backups - password required")
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(m.config.Dir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	// Generate filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("2006-01-02_150405")
		filename = fmt.Sprintf("casman_backup_%s.tar.gz", timestamp)
	}

	// Add encryption extension if needed
	encrypt := password != "" || m.config.Encryption.Enabled
	if encrypt && !strings.HasSuffix(filename, ".enc") {
		filename += ".enc"
	}

	backupPath := filepath.Join(m.config.Dir, filename)

	// Collect files to backup
	files, err := m.collectBackupFiles()
	if err != nil {
		return "", fmt.Errorf("collecting files: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files to backup")
	}

	// Create manifest
	manifest := Manifest{
		Version:    "1.0.0",
		CreatedAt:  time.Now().UTC(),
		CreatedBy:  createdBy,
		AppVersion: m.appVersion,
		Contents:   files,
		Encrypted:  encrypt,
	}
	if encrypt {
		manifest.EncryptionMethod = "AES-256-GCM"
	}

	// Create archive
	var archiveData []byte
	if encrypt {
		archiveData, err = m.createArchiveInMemory(files, manifest)
		if err != nil {
			return "", fmt.Errorf("creating archive: %w", err)
		}

		// Compute checksum before encryption
		hash := sha256.Sum256(archiveData)
		manifest.Checksum = "sha256:" + hex.EncodeToString(hash[:])

		// Encrypt
		archiveData, err = m.encrypt(archiveData, password)
		if err != nil {
			return "", fmt.Errorf("encrypting backup: %w", err)
		}

		// Write encrypted file
		if err := os.WriteFile(backupPath, archiveData, 0600); err != nil {
			return "", fmt.Errorf("writing backup: %w", err)
		}
	} else {
		// Create directly to file
		if err := m.createArchiveToFile(backupPath, files, manifest); err != nil {
			return "", fmt.Errorf("creating archive: %w", err)
		}
	}

	// Verify backup
	if err := m.verifyBackup(backupPath, password); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("verification failed: %w", err)
	}

	log.Printf("Backup: created %s (%d files)", filename, len(files))
	return backupPath, nil
}

// collectBackupFiles returns list of files to backup.
func (m *Manager) collectBackupFiles() ([]string, error) {
	var files []string

	// server.yml
	serverYml := filepath.Join(m.configDir, "server.yml")
	if fileExists(serverYml) {
		files = append(files, serverYml)
	}

	// server.db
	serverDB := filepath.Join(m.dataDir, "db", "server.db")
	if fileExists(serverDB) {
		files = append(files, serverDB)
	}

	// users.db (if exists)
	usersDB := filepath.Join(m.dataDir, "db", "users.db")
	if fileExists(usersDB) {
		files = append(files, usersDB)
	}

	// Custom templates
	templatesDir := filepath.Join(m.configDir, "template")
	if dirExists(templatesDir) {
		if err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			log.Printf("Backup: warning scanning templates: %v", err)
		}
	}

	// Custom themes
	themesDir := filepath.Join(m.configDir, "theme")
	if dirExists(themesDir) {
		if err := filepath.Walk(themesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			log.Printf("Backup: warning scanning themes: %v", err)
		}
	}

	return files, nil
}

// createArchiveInMemory creates a tar.gz archive in memory.
func (m *Manager) createArchiveInMemory(files []string, manifest Manifest) ([]byte, error) {
	var buf strings.Builder
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Add manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    "manifest.json",
		Mode:    0644,
		Size:    int64(len(manifestData)),
		ModTime: time.Now(),
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(manifestData); err != nil {
		return nil, err
	}

	// Add files
	for _, file := range files {
		if err := m.addFileToTar(tw, file); err != nil {
			return nil, fmt.Errorf("adding %s: %w", file, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// createArchiveToFile creates a tar.gz archive to a file.
func (m *Manager) createArchiveToFile(path string, files []string, manifest Manifest) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Add manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    "manifest.json",
		Mode:    0644,
		Size:    int64(len(manifestData)),
		ModTime: time.Now(),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(manifestData); err != nil {
		return err
	}

	// Add files
	for _, file := range files {
		if err := m.addFileToTar(tw, file); err != nil {
			return fmt.Errorf("adding %s: %w", file, err)
		}
	}

	return nil
}

// addFileToTar adds a file to the tar archive.
func (m *Manager) addFileToTar(tw *tar.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Use relative path in archive
	relPath, err := filepath.Rel(m.configDir, filePath)
	if err != nil {
		relPath, err = filepath.Rel(m.dataDir, filePath)
		if err != nil {
			relPath = filepath.Base(filePath)
		} else {
			relPath = "data/" + relPath
		}
	} else {
		relPath = "config/" + relPath
	}

	header := &tar.Header{
		Name:    relPath,
		Mode:    int64(stat.Mode()),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// encrypt encrypts data using AES-256-GCM with Argon2id key derivation.
func (m *Manager) encrypt(data []byte, password string) ([]byte, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key using Argon2id
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Prepend salt + nonce to ciphertext
	result := make([]byte, len(salt)+len(nonce)+len(ciphertext))
	copy(result, salt)
	copy(result[len(salt):], nonce)
	copy(result[len(salt)+len(nonce):], ciphertext)

	return result, nil
}

// decrypt decrypts data using AES-256-GCM with Argon2id key derivation.
func (m *Manager) decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 28 {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	// Extract salt and nonce
	salt := data[:16]
	nonce := data[16:28]
	ciphertext := data[28:]

	// Derive key
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?)")
	}

	return plaintext, nil
}

// verifyBackup verifies a backup file.
func (m *Manager) verifyBackup(path, password string) error {
	// Check file exists
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Check file size
	if stat.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Decrypt if needed
	if strings.HasSuffix(path, ".enc") {
		data, err = m.decrypt(data, password)
		if err != nil {
			return fmt.Errorf("decryption: %w", err)
		}
	}

	// Verify it's valid gzip
	gzr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gzr.Close()

	// Verify tar and find manifest
	tr := tar.NewReader(gzr)
	foundManifest := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		if header.Name == "manifest.json" {
			foundManifest = true
			var manifest Manifest
			if err := json.NewDecoder(tr).Decode(&manifest); err != nil {
				return fmt.Errorf("invalid manifest: %w", err)
			}
		}
	}

	if !foundManifest {
		return fmt.Errorf("manifest not found")
	}

	return nil
}

// Restore restores from a backup archive.
func (m *Manager) Restore(ctx context.Context, backupPath, password string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}

	// Decrypt if needed
	if strings.HasSuffix(backupPath, ".enc") {
		if password == "" {
			return fmt.Errorf("encrypted backup requires password")
		}
		data, err = m.decrypt(data, password)
		if err != nil {
			return err
		}
	}

	// Extract
	gzr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Skip manifest
		if header.Name == "manifest.json" {
			continue
		}

		// Determine target path
		var targetPath string
		if strings.HasPrefix(header.Name, "config/") {
			targetPath = filepath.Join(m.configDir, strings.TrimPrefix(header.Name, "config/"))
		} else if strings.HasPrefix(header.Name, "data/") {
			targetPath = filepath.Join(m.dataDir, strings.TrimPrefix(header.Name, "data/"))
		} else {
			// Default to config dir
			targetPath = filepath.Join(m.configDir, header.Name)
		}

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Extract file
		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("extracting file: %w", err)
		}
		outFile.Close()

		// Restore permissions
		if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
			log.Printf("Backup: warning setting permissions on %s: %v", targetPath, err)
		}

		log.Printf("Backup: restored %s", header.Name)
	}

	log.Println("Backup: restore completed")
	return nil
}

// ListBackups returns available backups sorted by date (newest first).
func (m *Manager) ListBackups() ([]BackupInfo, error) {
	var backups []BackupInfo

	entries, err := os.ReadDir(m.config.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return backups, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "casman_backup_") {
			continue
		}
		if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".tar.gz.enc") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		encrypted := strings.HasSuffix(name, ".enc")

		backups = append(backups, BackupInfo{
			Name:      name,
			Path:      filepath.Join(m.config.Dir, name),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Encrypted: encrypted,
		})
	}

	// Sort by date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// BackupInfo holds information about a backup file.
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Encrypted bool      `json:"encrypted"`
}

// ApplyRetention applies retention policy and deletes old backups.
func (m *Manager) ApplyRetention() error {
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}

	// Skip incrementals
	var fullBackups []BackupInfo
	for _, b := range backups {
		if !strings.Contains(b.Name, "-daily") && !strings.Contains(b.Name, "-hourly") {
			fullBackups = append(fullBackups, b)
		}
	}

	if len(fullBackups) <= m.config.Retention.MaxBackups {
		return nil
	}

	// Delete oldest backups beyond retention
	toDelete := fullBackups[m.config.Retention.MaxBackups:]
	for _, b := range toDelete {
		log.Printf("Backup: deleting old backup %s (retention policy)", b.Name)
		if err := os.Remove(b.Path); err != nil {
			log.Printf("Backup: warning deleting %s: %v", b.Name, err)
		}
	}

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Package secret holds the long-lived AES-256-GCM master key used to seal
// server-managed secrets such as DNS provider credentials. See AI.md PART 15.
//
// The key is sourced in this order:
//  1. CASMAN_MASTER_SECRET environment variable, hex-encoded 32 bytes.
//  2. {config_dir}/security/master.key, 32 raw bytes, mode 0600.
//
// If neither is present, a fresh 32-byte key is generated and persisted to
// the file path with 0600 permissions on first run.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	envVar      = "CASMAN_MASTER_SECRET"
	keyFileName = "master.key"
	keyDirName  = "security"
	keyLen      = 32
)

// Vault wraps an AES-256-GCM cipher backed by a long-lived key.
type Vault struct {
	gcm cipher.AEAD
}

// LoadOrCreate resolves the master key from env or disk, creating the file on
// first run when neither source exists.
func LoadOrCreate(configDir string) (*Vault, error) {
	if hexKey := os.Getenv(envVar); hexKey != "" {
		key, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, fmt.Errorf("decoding %s: %w", envVar, err)
		}
		if len(key) != keyLen {
			return nil, fmt.Errorf("%s must be %d hex-encoded bytes (got %d)", envVar, keyLen, len(key))
		}
		return newVault(key)
	}

	keyPath := filepath.Join(configDir, keyDirName, keyFileName)
	key, err := os.ReadFile(keyPath)
	if err == nil {
		if len(key) != keyLen {
			return nil, fmt.Errorf("%s: expected %d bytes, got %d", keyPath, keyLen, len(key))
		}
		return newVault(key)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading %s: %w", keyPath, err)
	}

	key = make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating master key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", filepath.Dir(keyPath), err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("writing %s: %w", keyPath, err)
	}
	return newVault(key)
}

func newVault(key []byte) (*Vault, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init: %w", err)
	}
	return &Vault{gcm: gcm}, nil
}

// Encrypt seals plaintext with a fresh nonce; the nonce is prepended to the
// returned ciphertext.
func (v *Vault) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, v.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	out := make([]byte, 0, len(nonce)+len(plaintext)+v.gcm.Overhead())
	out = append(out, nonce...)
	return v.gcm.Seal(out, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt; it returns an error if the ciphertext is too short
// or the authentication tag fails.
func (v *Vault) Decrypt(ciphertext []byte) ([]byte, error) {
	ns := v.gcm.NonceSize()
	if len(ciphertext) < ns+v.gcm.Overhead() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, sealed := ciphertext[:ns], ciphertext[ns:]
	return v.gcm.Open(nil, nonce, sealed, nil)
}

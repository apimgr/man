package secret

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreate_GeneratesAndReusesKey(t *testing.T) {
	dir := t.TempDir()

	v1, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	keyPath := filepath.Join(dir, keyDirName, keyFileName)
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("key file perm = %o, want 0600", perm)
	}

	plain := []byte("hello world")
	sealed, err := v1.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	v2, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	got, err := v2.Decrypt(sealed)
	if err != nil {
		t.Fatalf("decrypt across instances: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, plain)
	}
}

func TestLoadOrCreate_PrefersEnv(t *testing.T) {
	dir := t.TempDir()

	envKey := bytes.Repeat([]byte{0x42}, keyLen)
	t.Setenv(envVar, hex.EncodeToString(envKey))

	v, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("load with env: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, keyDirName, keyFileName)); !os.IsNotExist(err) {
		t.Errorf("env key path was created on disk; expected env-only")
	}

	plain := []byte("secret")
	sealed, err := v.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := v.Decrypt(sealed)
	if err != nil || !bytes.Equal(got, plain) {
		t.Errorf("env round-trip failed: got %q err %v", got, err)
	}
}

func TestLoadOrCreate_RejectsMalformedEnv(t *testing.T) {
	t.Setenv(envVar, "not-hex")
	if _, err := LoadOrCreate(t.TempDir()); err == nil {
		t.Error("expected error for non-hex env value")
	}

	t.Setenv(envVar, hex.EncodeToString([]byte{0x01, 0x02}))
	if _, err := LoadOrCreate(t.TempDir()); err == nil {
		t.Error("expected error for short env value")
	}
}

func TestDecrypt_RejectsTamper(t *testing.T) {
	v, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	sealed, err := v.Encrypt([]byte("payload"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	sealed[len(sealed)-1] ^= 0xFF
	if _, err := v.Decrypt(sealed); err == nil {
		t.Error("expected auth failure on tampered ciphertext")
	}
}

func TestDecrypt_RejectsShort(t *testing.T) {
	v, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := v.Decrypt([]byte{0x00}); err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

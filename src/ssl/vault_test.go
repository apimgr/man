package ssl

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/casapps/casman/src/secret"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	sealer, err := secret.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}

	v, err := NewVault(db, sealer)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	return v
}

func TestVault_SaveLoadRoundTrip(t *testing.T) {
	v := newTestVault(t)

	want := map[string]string{
		"api_token": "super-secret-token",
		"zone_id":   "abc123",
	}
	if err := v.Save("cloudflare", want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := v.Load("cloudflare")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch: got %v want %v", got, want)
	}
}

func TestVault_LoadMissing(t *testing.T) {
	v := newTestVault(t)
	if _, err := v.Load("nope"); !errors.Is(err, ErrCredentialsNotFound) {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}

func TestVault_OverwriteOnSave(t *testing.T) {
	v := newTestVault(t)
	if err := v.Save("godaddy", map[string]string{"key": "v1"}); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if err := v.Save("godaddy", map[string]string{"key": "v2"}); err != nil {
		t.Fatalf("save v2: %v", err)
	}
	got, _ := v.Load("godaddy")
	if got["key"] != "v2" {
		t.Errorf("expected upsert to take latest, got %v", got)
	}
}

func TestVault_ListAndDelete(t *testing.T) {
	v := newTestVault(t)
	for _, p := range []string{"route53", "cloudflare", "manual"} {
		if err := v.Save(p, map[string]string{"x": p}); err != nil {
			t.Fatalf("save %s: %v", p, err)
		}
	}
	got, err := v.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []string{"cloudflare", "manual", "route53"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("list = %v, want %v", got, want)
	}
	if err := v.Delete("manual"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = v.List()
	want = []string{"cloudflare", "route53"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after delete list = %v, want %v", got, want)
	}
}

func TestVault_EmptyProviderRejected(t *testing.T) {
	v := newTestVault(t)
	if err := v.Save("", map[string]string{"x": "y"}); err == nil {
		t.Error("expected empty provider to be rejected")
	}
}

func TestNewVault_NilArgs(t *testing.T) {
	if _, err := NewVault(nil, nil); err == nil {
		t.Error("expected error for nil db")
	}
}

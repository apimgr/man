package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// redirectTransport rewrites every request to point at a test server,
// so package-level GitHub URLs can be exercised without real network access.
type redirectTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.base.Scheme
	req.URL.Host = t.base.Host
	return t.rt.RoundTrip(req)
}

func newTestUpdater(t *testing.T, handler http.Handler) (*Updater, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	u := New("1.0.0", BranchStable)
	u.httpClient = &http.Client{Transport: &redirectTransport{base: base, rt: http.DefaultTransport}}
	return u, srv
}

func TestNew_DefaultsBranch(t *testing.T) {
	u := New("1.0.0", "")
	if u.GetBranch() != BranchStable {
		t.Errorf("GetBranch = %q, want %q", u.GetBranch(), BranchStable)
	}
}

func TestSetBranch(t *testing.T) {
	u := New("1.0.0", BranchStable)
	for _, b := range []string{BranchStable, BranchBeta, BranchDaily} {
		if err := u.SetBranch(b); err != nil {
			t.Errorf("SetBranch(%q): %v", b, err)
		}
		if u.GetBranch() != b {
			t.Errorf("GetBranch = %q, want %q", u.GetBranch(), b)
		}
	}
}

func TestSetBranch_Invalid(t *testing.T) {
	u := New("1.0.0", BranchStable)
	if err := u.SetBranch("nightly"); err == nil {
		t.Error("expected error for invalid branch")
	}
}

func TestGetBinaryName(t *testing.T) {
	name := GetBinaryName()
	want := "casman-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Errorf("GetBinaryName = %q, want %q", name, want)
	}
}

func TestVerifyChecksum_Match(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.bin")
	content := []byte("hello world")
	if err := os.WriteFile(f, content, 0644); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(content)
	if err := VerifyChecksum(f, hex.EncodeToString(h[:])); err != nil {
		t.Errorf("VerifyChecksum: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.bin")
	if err := os.WriteFile(f, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum(f, "0000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected mismatch error")
	}
}

func TestVerifyChecksum_MissingFile(t *testing.T) {
	if err := VerifyChecksum("/nonexistent/file", "abc"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestMatchesBranch(t *testing.T) {
	u := New("1.0.0", BranchStable)

	u.branch = BranchBeta
	if !u.matchesBranch(Release{TagName: "v1.0.0-beta"}) {
		t.Error("expected beta match")
	}
	if u.matchesBranch(Release{TagName: "v1.0.0"}) {
		t.Error("expected no beta match")
	}

	u.branch = BranchDaily
	if !u.matchesBranch(Release{TagName: "20240101120000"}) {
		t.Error("expected daily match")
	}
	if u.matchesBranch(Release{TagName: "v1.0.0"}) {
		t.Error("expected no daily match")
	}

	u.branch = BranchStable
	if !u.matchesBranch(Release{TagName: "v1.0.0", Prerelease: false}) {
		t.Error("expected stable match for non-prerelease")
	}
	if u.matchesBranch(Release{TagName: "v1.0.0", Prerelease: true}) {
		t.Error("expected no stable match for prerelease")
	}
}

func TestCheck_NoNewerVersion(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"1.0.0","draft":false,"prerelease":false,"assets":[]}`))
	}))
	defer srv.Close()

	info, err := u.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Error("expected no update available for same version")
	}
}

func TestCheck_NewerVersionNoAsset(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"2.0.0","draft":false,"prerelease":false,"assets":[]}`))
	}))
	defer srv.Close()

	if _, err := u.Check(context.Background()); err == nil {
		t.Error("expected error when no matching asset found")
	}
}

func TestCheck_NewerVersionWithAsset(t *testing.T) {
	assetName := GetBinaryName()
	body := `{"tag_name":"2.0.0","body":"notes","draft":false,"prerelease":false,"assets":[{"name":"` + assetName + `","size":123,"browser_download_url":"http://example.com/dl"}]}`
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	info, err := u.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatal("expected update available")
	}
	if info.NewVersion != "2.0.0" {
		t.Errorf("NewVersion = %q", info.NewVersion)
	}
	if info.AssetSize != 123 {
		t.Errorf("AssetSize = %d", info.AssetSize)
	}
}

func TestCheck_NotFoundMeansNoRelease(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	info, err := u.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Error("expected no update available")
	}
}

func TestCheck_ServerError(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := u.Check(context.Background()); err == nil {
		t.Error("expected error on server error")
	}
}

func TestFetchRelease_DraftIgnored(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"2.0.0","draft":true}`))
	}))
	defer srv.Close()

	release, err := u.fetchRelease(context.Background())
	if err != nil {
		t.Fatalf("fetchRelease: %v", err)
	}
	if release != nil {
		t.Error("expected nil release for draft")
	}
}

func TestFetchRelease_BetaFiltersList(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"tag_name":"2.0.0","draft":false},{"tag_name":"2.0.0-beta","draft":false}]`))
	}))
	defer srv.Close()
	u.branch = BranchBeta

	release, err := u.fetchRelease(context.Background())
	if err != nil {
		t.Fatalf("fetchRelease: %v", err)
	}
	if release == nil || release.TagName != "2.0.0-beta" {
		t.Errorf("expected beta release, got %+v", release)
	}
}

func TestDownload_Success(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("binary-content"))
	}))
	defer srv.Close()

	path, err := u.download(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != "binary-content" {
		t.Errorf("content = %q", content)
	}
}

func TestDownload_ErrorStatus(t *testing.T) {
	u, srv := newTestUpdater(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := u.download(context.Background(), srv.URL); err == nil {
		t.Error("expected error for non-200 status")
	}
}

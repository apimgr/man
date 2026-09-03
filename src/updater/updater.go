// Package updater implements self-update functionality.
// See AI.md PART 23 for details.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHub repository info
	GitHubOwner = "casapps"
	GitHubRepo  = "casman"

	// Update branches
	BranchStable = "stable"
	BranchBeta   = "beta"
	BranchDaily  = "daily"
)

// Release represents a GitHub release.
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Prerelease bool    `json:"prerelease"`
	Draft      bool    `json:"draft"`
	CreatedAt  string  `json:"created_at"`
	Assets     []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo holds information about an available update.
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	NewVersion     string
	ReleaseNotes   string
	DownloadURL    string
	AssetName      string
	AssetSize      int64
}

// Updater handles update operations.
type Updater struct {
	currentVersion string
	branch         string
	httpClient     *http.Client
}

// New creates a new Updater.
func New(currentVersion, branch string) *Updater {
	if branch == "" {
		branch = BranchStable
	}

	return &Updater{
		currentVersion: currentVersion,
		branch:         branch,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// SetBranch changes the update branch.
func (u *Updater) SetBranch(branch string) error {
	switch branch {
	case BranchStable, BranchBeta, BranchDaily:
		u.branch = branch
		return nil
	default:
		return fmt.Errorf("invalid branch: %s (valid: stable, beta, daily)", branch)
	}
}

// GetBranch returns the current update branch.
func (u *Updater) GetBranch() string {
	return u.branch
}

// Check checks for available updates without installing.
func (u *Updater) Check(ctx context.Context) (*UpdateInfo, error) {
	release, err := u.fetchRelease(ctx)
	if err != nil {
		return nil, err
	}

	if release == nil {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: u.currentVersion,
		}, nil
	}

	// Check if this is actually newer
	if release.TagName == u.currentVersion || release.TagName == "v"+u.currentVersion {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: u.currentVersion,
		}, nil
	}

	// Find the asset for this platform
	assetName := GetBinaryName()
	var downloadURL string
	var assetSize int64

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			assetSize = asset.Size
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return &UpdateInfo{
		Available:      true,
		CurrentVersion: u.currentVersion,
		NewVersion:     release.TagName,
		ReleaseNotes:   release.Body,
		DownloadURL:    downloadURL,
		AssetName:      assetName,
		AssetSize:      assetSize,
	}, nil
}

// Update downloads and installs the update.
func (u *Updater) Update(ctx context.Context) (*UpdateInfo, error) {
	info, err := u.Check(ctx)
	if err != nil {
		return nil, err
	}

	if !info.Available {
		return info, nil
	}

	// Download the new binary
	tmpPath, err := u.download(ctx, info.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpPath)

	// Make executable (Unix)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Get current binary path
	currentPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Replace binary (platform-specific)
	if err := ReplaceBinary(currentPath, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to replace binary: %w", err)
	}

	return info, nil
}

// fetchRelease fetches the appropriate release from GitHub.
func (u *Updater) fetchRelease(ctx context.Context) (*Release, error) {
	var url string

	switch u.branch {
	case BranchStable:
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)
	default:
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", GitHubOwner, GitHubRepo)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "casman-updater/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No releases available
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	if u.branch == BranchStable {
		var release Release
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, err
		}
		if release.Draft {
			return nil, nil
		}
		return &release, nil
	}

	// For beta/daily, filter releases
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	for _, r := range releases {
		if r.Draft {
			continue
		}
		if u.matchesBranch(r) {
			return &r, nil
		}
	}

	return nil, nil
}

// matchesBranch checks if a release matches the current branch.
func (u *Updater) matchesBranch(r Release) bool {
	switch u.branch {
	case BranchBeta:
		return strings.HasSuffix(r.TagName, "-beta")
	case BranchDaily:
		// Daily builds are timestamps: YYYYMMDDHHMMSS
		return len(r.TagName) == 14 && !strings.Contains(r.TagName, ".")
	default:
		return !r.Prerelease
	}
}

// download downloads the binary to a temp file.
func (u *Updater) download(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "casman-updater/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "casman-update-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()

	// Download with progress
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return tmpPath, nil
}

// VerifyChecksum verifies the SHA256 checksum of a file.
func VerifyChecksum(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// GetBinaryName returns the expected binary name for this platform.
func GetBinaryName() string {
	name := "casman-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// RestartSelf restarts the current process after update.
// This is platform-specific and defined in updater_unix.go and updater_windows.go.

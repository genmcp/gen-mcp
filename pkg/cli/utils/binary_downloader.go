package utils

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	GitHubReleasesURL      = "https://github.com/genmcp/gen-mcp/releases/download"
	GitHubAPIURL           = "https://api.github.com/repos/genmcp/gen-mcp/releases/latest"
	CacheKeepVersionsCount = 3 // Number of versions to keep per platform in cache
)

// BinaryDownloader handles downloading server binaries from GitHub releases
type BinaryDownloader struct {
	cache    *BinaryCache
	client   *http.Client
	verifier *SigstoreVerifier
}

// NewBinaryDownloader creates a new binary downloader
func NewBinaryDownloader() (*BinaryDownloader, error) {
	cache, err := NewBinaryCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create binary cache: %w", err)
	}

	verifier, err := NewSigstoreVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to create sigstore verifier: %w", err)
	}

	return &BinaryDownloader{
		cache:    cache,
		client:   &http.Client{},
		verifier: verifier,
	}, nil
}

// GetBinary retrieves a server binary for the specified version and platform
// Returns the path to the binary file
func (bd *BinaryDownloader) GetBinary(version, goos, goarch string) (string, error) {
	if version == "latest" {
		actualVersion, err := bd.fetchLatestVersion()
		if err != nil {
			return "", fmt.Errorf("failed to fetch latest version: %w", err)
		}
		version = actualVersion
	}

	platform := fmt.Sprintf("%s-%s", goos, goarch)

	// Check cache first
	if cachedPath, found := bd.cache.Get(version, platform); found {
		return cachedPath, nil
	}

	// Download and verify
	binaryPath, err := bd.downloadAndVerify(version, goos, goarch)
	if err != nil {
		// Download failed, try to use cached version as fallback
		if cachedVersion, cachedPath := bd.cache.GetLatestCached(platform); cachedVersion != "" {
			fmt.Printf("Warning: Failed to download %s: %v\n", version, err)
			fmt.Printf("Using cached version %s as fallback\n", cachedVersion)
			return cachedPath, nil
		}
		return "", fmt.Errorf("failed to download and no cached version available: %w", err)
	}

	// Extract temp directory from binary path for cleanup
	// Binary is extracted to: /tmp/genmcp-download-xxx/genmcp-server-linux-amd64
	tempDir := filepath.Dir(binaryPath)
	defer os.RemoveAll(tempDir) // Clean up after caching

	if err := bd.CleanOldBinaries(CacheKeepVersionsCount); err != nil {
		fmt.Printf("Warning: Failed to clean old binaries: %v\n", err)
	}

	cachedPath, err := bd.cache.Add(version, platform, binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to cache binary: %w", err)
	}

	return cachedPath, nil
}

// downloadAndVerify downloads a binary and verifies it with cosign
// Returns the path to the extracted binary. Caller is responsible for cleanup.
func (bd *BinaryDownloader) downloadAndVerify(version, goos, goarch string) (string, error) {
	tempDir, err := os.MkdirTemp("", "genmcp-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	platform := fmt.Sprintf("%s-%s", goos, goarch)
	zipFilename := fmt.Sprintf("genmcp-server-%s.zip", platform)

	zipPath := filepath.Join(tempDir, zipFilename)
	if err := bd.downloadFile(version, zipFilename, zipPath); err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}

	bundlePath := filepath.Join(tempDir, zipFilename+".bundle")
	if err := bd.downloadFile(version, zipFilename+".bundle", bundlePath); err != nil {
		return "", fmt.Errorf("failed to download bundle: %w", err)
	}

	if err := bd.verifier.VerifyBlob(zipPath, bundlePath); err != nil {
		return "", fmt.Errorf("sigstore verification failed: %w", err)
	}

	binaryPath, err := bd.extractBinary(zipPath, tempDir, goos, goarch)
	if err != nil {
		return "", fmt.Errorf("failed to extract binary: %w", err)
	}

	return binaryPath, nil
}

// downloadFile downloads a file from GitHub releases
func (bd *BinaryDownloader) downloadFile(version, filename, destPath string) error {
	url := fmt.Sprintf("%s/%s/%s", GitHubReleasesURL, version, filename)

	resp, err := bd.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractBinary extracts the binary from a ZIP file
func (bd *BinaryDownloader) extractBinary(zipPath, destDir, goos, goarch string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	expectedName := fmt.Sprintf("genmcp-server-%s-%s", goos, goarch)
	if goos == "windows" {
		expectedName += ".exe"
	}

	for _, f := range r.File {
		if f.Name != expectedName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		destPath := filepath.Join(destDir, f.Name)
		dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}
		defer dest.Close()

		if _, err := io.Copy(dest, rc); err != nil {
			return "", err
		}

		return destPath, nil
	}

	return "", fmt.Errorf("binary %s not found in ZIP archive", expectedName)
}

// GetCurrentBinary gets the binary for the current platform and a specific version
func (bd *BinaryDownloader) GetCurrentBinary(version string) (string, error) {
	return bd.GetBinary(version, runtime.GOOS, runtime.GOARCH)
}

// fetchLatestVersion fetches the latest release version from GitHub API
func (bd *BinaryDownloader) fetchLatestVersion() (string, error) {
	resp, err := bd.client.Get(GitHubAPIURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name in latest release response")
	}

	return release.TagName, nil
}

// CleanOldBinaries removes old cached binaries (keeps last N versions)
func (bd *BinaryDownloader) CleanOldBinaries(keepVersions int) error {
	return bd.cache.Clean(keepVersions)
}

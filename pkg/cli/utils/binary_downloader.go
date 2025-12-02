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
	"time"
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
	verbose  bool // Enable progress output
}

// NewBinaryDownloader creates a new binary downloader
func NewBinaryDownloader(verbose bool) (*BinaryDownloader, error) {
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
		client: &http.Client{
			Timeout: 5 * time.Minute, // timeout for binary downloads
		},
		verifier: verifier,
		verbose:  verbose,
	}, nil
}

// printf outputs formatted message if verbose mode is enabled
func (bd *BinaryDownloader) printf(format string, args ...interface{}) {
	if bd.verbose {
		fmt.Printf(format, args...)
	}
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
		bd.printf("Resolved latest version: %s\n", version)
	}

	platform := fmt.Sprintf("%s-%s", goos, goarch)

	// Try to get requested version from cache first
	cachedZipPath, foundInCache := bd.cache.Get(version, platform)
	if foundInCache {
		bd.printf("Using cached binary: %s (%s)\n", version, platform)
		binaryPath, err := bd.verifyAndExtractZip(cachedZipPath, version, goos, goarch)
		if err == nil {
			return binaryPath, nil
		}
		fmt.Printf("Warning: Cached binary verification failed: %v\n", err)
		fmt.Printf("Re-downloading %s...\n", version)
	}

	// Download and verify the requested version
	bd.printf("Downloading binary: %s (%s)\n", version, platform)
	zipPath, err := bd.downloadAndVerify(version, goos, goarch)
	if err != nil {

		return "", fmt.Errorf("failed to download and no cached version available: %w", err)
	}

	tempDir := filepath.Dir(zipPath)
	defer func() { _ = os.RemoveAll(tempDir) }() // Clean up after caching

	if err := bd.CleanOldBinaries(CacheKeepVersionsCount); err != nil {
		fmt.Printf("Warning: Failed to clean old binaries: %v\n", err)
	}

	cachedZipPath, err = bd.cache.Add(version, platform, zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to cache binary: %w", err)
	}

	// Extract binary from cached zip
	binaryPath, err := bd.verifyAndExtractZip(cachedZipPath, version, goos, goarch)
	if err != nil {
		return "", fmt.Errorf("failed to extract cached binary: %w", err)
	}

	return binaryPath, nil
}

// verifyAndExtractZip verifies a zip file with Sigstore and extracts the binary
// Returns the path to the extracted binary in a temp directory. Caller is responsible for cleanup.
func (bd *BinaryDownloader) verifyAndExtractZip(zipPath, version, goos, goarch string) (string, error) {
	if err := bd.verifyZipWithBundle(zipPath, version, goos, goarch); err != nil {
		return "", err
	}

	extractDir, err := os.MkdirTemp("", "genmcp-extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create extract dir: %w", err)
	}

	binaryPath, err := bd.extractBinary(zipPath, extractDir, goos, goarch)
	if err != nil {
		_ = os.RemoveAll(extractDir)
		return "", fmt.Errorf("failed to extract binary: %w", err)
	}

	return binaryPath, nil
}

// verifyZipWithBundle downloads a fresh bundle and verifies a zip file with Sigstore
func (bd *BinaryDownloader) verifyZipWithBundle(zipPath, version, goos, goarch string) error {
	tempDir, err := os.MkdirTemp("", "genmcp-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	platform := fmt.Sprintf("%s-%s", goos, goarch)
	zipFilename := fmt.Sprintf("genmcp-server-%s.zip", platform)
	bundlePath := filepath.Join(tempDir, zipFilename+".bundle")

	if err := bd.downloadFile(version, zipFilename+".bundle", bundlePath); err != nil {
		return fmt.Errorf("failed to download bundle: %w", err)
	}

	bd.printf("  Verifying with Sigstore...\n")
	if err := bd.verifier.VerifyBlob(zipPath, bundlePath); err != nil {
		return fmt.Errorf("sigstore verification failed: %w", err)
	}

	return nil
}

// downloadAndVerify downloads a binary and verifies it with Sigstore
// Returns the path to the verified zip file in temp directory.
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

	if err := bd.verifyZipWithBundle(zipPath, version, goos, goarch); err != nil {
		return "", err
	}

	return zipPath, nil
}

// downloadFile downloads a file from GitHub releases
func (bd *BinaryDownloader) downloadFile(version, filename, destPath string) error {
	url := fmt.Sprintf("%s/%s/%s", GitHubReleasesURL, version, filename)

	bd.printf("  Downloading: %s\n", filename)

	resp, err := bd.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractBinary extracts the binary from a ZIP file
func (bd *BinaryDownloader) extractBinary(zipPath, destDir, goos, goarch string) (string, error) {
	bd.printf("  Extracting binary...\n")

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()

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
		defer func() { _ = rc.Close() }()

		destPath := filepath.Join(destDir, f.Name)
		dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}
		defer func() { _ = dest.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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

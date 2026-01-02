package binarycache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BinaryCacheEntry represents a cached binary
type BinaryCacheEntry struct {
	Version      string `json:"version"`
	Platform     string `json:"platform"`      // e.g., "linux-amd64"
	ZipPath      string `json:"zip_path"`      // absolute path to verified zip file
	SHA256       string `json:"sha256"`        // checksum of zip file
	DownloadedAt string `json:"downloaded_at"` // timestamp
}

// BinaryCache manages cached server binaries
type BinaryCache struct {
	cacheDir  string
	prefix    string
	indexPath string
	entries   map[string]BinaryCacheEntry // key: "version-platform"
}

// NewBinaryCache creates a new binary cache manager using the default user cache dir and the provided name for the cache
func NewBinaryCache(cfg *Config) (*BinaryCache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	cacheDir = filepath.Join(cacheDir, cfg.GetCacheName())

	return NewBinaryCacheWithDir(cacheDir, cfg.GetBinaryPrefix())
}

// NewBinaryCacheWithDir creates a new binary cache manager with a custom cache directory
func NewBinaryCacheWithDir(cacheDir, prefix string) (*BinaryCache, error) {
	binariesDir := filepath.Join(cacheDir, "binaries")
	if err := os.MkdirAll(binariesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create binaries cache dir: %w", err)
	}

	indexPath := filepath.Join(cacheDir, "binaries-index.json")

	bc := &BinaryCache{
		cacheDir:  binariesDir,
		prefix:    prefix,
		indexPath: indexPath,
		entries:   make(map[string]BinaryCacheEntry),
	}

	if err := bc.loadIndex(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load cache index: %w", err)
	}

	return bc, nil
}

// Get retrieves a cached binary path for a version and platform
func (bc *BinaryCache) Get(version, platform string) (string, bool) {
	key := fmt.Sprintf("%s-%s", version, platform)
	entry, exists := bc.entries[key]
	if !exists {
		return "", false
	}

	if _, err := os.Stat(entry.ZipPath); err != nil {
		delete(bc.entries, key)
		_ = bc.saveIndex()
		return "", false
	}

	if err := bc.verifyChecksum(entry.ZipPath, entry.SHA256); err != nil {
		delete(bc.entries, key)
		_ = bc.saveIndex()
		return "", false
	}

	return entry.ZipPath, true
}

// Add adds a binary zip file to the cache
func (bc *BinaryCache) Add(version, platform, sourcePath string) (string, error) {
	key := fmt.Sprintf("%s-%s", version, platform)

	checksum, err := bc.calculateChecksum(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	filename := fmt.Sprintf("%s-%s-%s", bc.prefix, version, platform)
	if filepath.Ext(sourcePath) != "" {
		filename += filepath.Ext(sourcePath)
	}
	destPath := filepath.Join(bc.cacheDir, filename)

	if err := bc.copyFile(sourcePath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy zip to cache: %w", err)
	}

	entry := BinaryCacheEntry{
		Version:      version,
		Platform:     platform,
		ZipPath:      destPath,
		SHA256:       checksum,
		DownloadedAt: time.Now().Format(time.RFC3339),
	}
	bc.entries[key] = entry

	if err := bc.saveIndex(); err != nil {
		return "", fmt.Errorf("failed to save cache index: %w", err)
	}

	return destPath, nil
}

// loadIndex loads the cache index from disk
func (bc *BinaryCache) loadIndex() error {
	data, err := os.ReadFile(bc.indexPath)
	if err != nil {
		return err
	}

	var entries []BinaryCacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		key := fmt.Sprintf("%s-%s", entry.Version, entry.Platform)
		bc.entries[key] = entry
	}

	return nil
}

// saveIndex saves the cache index to disk
func (bc *BinaryCache) saveIndex() error {
	entries := make([]BinaryCacheEntry, 0, len(bc.entries))
	for _, entry := range bc.entries {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bc.indexPath, data, 0644)
}

// calculateChecksum calculates SHA256 checksum of a file
func (bc *BinaryCache) calculateChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyChecksum verifies the checksum of a file
func (bc *BinaryCache) verifyChecksum(path, expectedChecksum string) error {
	actualChecksum, err := bc.calculateChecksum(path)
	if err != nil {
		return err
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// copyFile copies a file from src to dst
func (bc *BinaryCache) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// GetLatestCached returns the most recently cached version for a platform
func (bc *BinaryCache) GetLatestCached(platform string) (string, string) {
	var latestEntry *BinaryCacheEntry

	for _, entry := range bc.entries {
		if entry.Platform == platform {
			if latestEntry == nil || entry.DownloadedAt > latestEntry.DownloadedAt {
				latestEntry = &entry
			}
		}
	}

	if latestEntry != nil {
		return latestEntry.Version, latestEntry.ZipPath
	}

	return "", ""
}

// Clean removes old cached binaries (keeps last N versions)
func (bc *BinaryCache) Clean(keepVersions int) error {
	platformVersions := make(map[string][]BinaryCacheEntry)
	for _, entry := range bc.entries {
		platformVersions[entry.Platform] = append(platformVersions[entry.Platform], entry)
	}

	for platform, entries := range platformVersions {
		if len(entries) <= keepVersions {
			continue
		}
		// Sort by download timestamp (newest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].DownloadedAt > entries[j].DownloadedAt
		})
		// Remove old entries
		for i := keepVersions; i < len(entries); i++ {
			key := fmt.Sprintf("%s-%s", entries[i].Version, platform)
			if err := os.Remove(entries[i].ZipPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			delete(bc.entries, key)
		}
	}

	return bc.saveIndex()
}

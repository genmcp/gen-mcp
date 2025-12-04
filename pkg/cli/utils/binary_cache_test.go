package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBinaryCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "genmcp-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	bc, err := NewBinaryCacheWithDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create binary cache: %v", err)
	}

	t.Run("Get nonexistent entry", func(t *testing.T) {
		_, found := bc.Get("v1.0.0", "linux-amd64")
		if found {
			t.Error("Expected entry to not be found")
		}
	})

	t.Run("Add and Get entry", func(t *testing.T) {
		tmpBinary := filepath.Join(tmpDir, "test-binary")
		testContent := []byte("fake binary content")
		if err := os.WriteFile(tmpBinary, testContent, 0755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		cachedPath, err := bc.Add("v1.0.0", "linux-amd64", tmpBinary)
		if err != nil {
			t.Fatalf("Failed to add binary to cache: %v", err)
		}

		retrievedPath, found := bc.Get("v1.0.0", "linux-amd64")
		if !found {
			t.Error("Expected entry to be found")
		}

		if retrievedPath != cachedPath {
			t.Errorf("Expected path %s, got %s", cachedPath, retrievedPath)
		}

		content, err := os.ReadFile(cachedPath)
		if err != nil {
			t.Fatalf("Failed to read cached binary: %v", err)
		}

		if string(content) != string(testContent) {
			t.Error("Cached binary content doesn't match original")
		}
	})

	t.Run("Checksum verification", func(t *testing.T) {
		tmpBinary := filepath.Join(tmpDir, "test-binary-2")
		if err := os.WriteFile(tmpBinary, []byte("test content"), 0755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		cachedPath, err := bc.Add("v1.0.1", "linux-amd64", tmpBinary)
		if err != nil {
			t.Fatalf("Failed to add binary to cache: %v", err)
		}

		_, found := bc.Get("v1.0.1", "linux-amd64")
		if !found {
			t.Error("Expected entry to be found")
		}

		if err := os.WriteFile(cachedPath, []byte("corrupted content"), 0755); err != nil {
			t.Fatalf("Failed to corrupt cached file: %v", err)
		}

		_, found = bc.Get("v1.0.1", "linux-amd64")
		if found {
			t.Error("Expected corrupted entry to not be found after checksum verification")
		}
	})

	t.Run("Persistence across instances", func(t *testing.T) {
		tmpBinary := filepath.Join(tmpDir, "test-binary-3")
		if err := os.WriteFile(tmpBinary, []byte("persistent content"), 0755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		if _, err := bc.Add("v1.0.2", "linux-amd64", tmpBinary); err != nil {
			t.Fatalf("Failed to add binary to cache: %v", err)
		}

		bc2, err := NewBinaryCacheWithDir(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create new binary cache instance: %v", err)
		}

		_, found := bc2.Get("v1.0.2", "linux-amd64")
		if !found {
			t.Error("Expected entry to persist across cache instances")
		}
	})

	t.Run("GetLatestCached", func(t *testing.T) {
		tmpBinary1 := filepath.Join(tmpDir, "test-binary-1")
		if err := os.WriteFile(tmpBinary1, []byte("version 1"), 0755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		tmpBinary2 := filepath.Join(tmpDir, "test-binary-2")
		if err := os.WriteFile(tmpBinary2, []byte("version 2"), 0755); err != nil {
			t.Fatalf("Failed to create test binary: %v", err)
		}

		if _, err := bc.Add("v1.0.0", "linux-amd64", tmpBinary1); err != nil {
			t.Fatalf("Failed to add binary: %v", err)
		}

		if _, err := bc.Add("v1.0.1", "linux-amd64", tmpBinary2); err != nil {
			t.Fatalf("Failed to add binary: %v", err)
		}

		version, path := bc.GetLatestCached("linux-amd64")
		if version == "" {
			t.Error("Expected to find cached version")
		}

		if path == "" {
			t.Error("Expected valid path")
		}

		version, path = bc.GetLatestCached("windows-amd64")
		if version != "" || path != "" {
			t.Error("Expected empty result for platform with no cache")
		}
	})
}

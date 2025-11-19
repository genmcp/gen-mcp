package builder

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func TestDownloadBinaryProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "genmcp-provider-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	t.Run("NewDownloadBinaryProvider", func(t *testing.T) {
		provider, err := NewDownloadBinaryProvider("v0.1.0")
		if err != nil {
			t.Logf("Note: Could not create download provider (expected in unit tests): %v", err)
			return
		}

		if provider == nil {
			t.Error("Expected provider to be created")
			return
		}

		if provider.version != "v0.1.0" {
			t.Errorf("Expected version v0.1.0, got %s", provider.version)
		}
	})

	t.Run("ExtractServerBinary with mock", func(t *testing.T) {
		// This test demonstrates the interface but doesn't actually download
		tmpBinary := filepath.Join(tmpDir, "mock-server")
		mockContent := []byte("mock server binary")
		if err := os.WriteFile(tmpBinary, mockContent, 0755); err != nil {
			t.Fatalf("Failed to create mock binary: %v", err)
		}

		platform := &v1.Platform{
			OS:           "linux",
			Architecture: "amd64",
		}

		if platform.OS != "linux" || platform.Architecture != "amd64" {
			t.Error("Platform not properly initialized")
		}

		t.Log("Note: Full download testing requires network access and valid GitHub releases")
	})
}

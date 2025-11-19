package builder

import (
	"strings"
	"testing"
)

func TestNewDownloadBinaryProvider(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedVersion string
	}{
		{
			name:            "creates provider with specified version",
			version:         "v0.1.0",
			expectedVersion: "v0.1.0",
		},
		{
			name:            "creates provider with latest version",
			version:         "latest",
			expectedVersion: "latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewDownloadBinaryProvider(tc.version)
			if err != nil {
				// Check if error is due to missing network access (Sigstore TUF fetch)
				if strings.Contains(err.Error(), "fetch trusted root") ||
					strings.Contains(err.Error(), "TUF") ||
					strings.Contains(err.Error(), "network") {
					t.Skipf("Skipping test: requires network access to fetch Sigstore trusted root: %v", err)
				}
				t.Fatalf("unexpected error creating download provider: %v", err)
			}

			if provider == nil {
				t.Fatal("Expected provider to be created, got nil")
			}

			if provider.version != tc.expectedVersion {
				t.Errorf("Expected version %s, got %s", tc.expectedVersion, provider.version)
			}

			if provider.downloader == nil {
				t.Error("Expected downloader to be initialized")
			}
		})
	}
}

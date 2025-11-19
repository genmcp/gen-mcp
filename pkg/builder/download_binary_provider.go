package builder

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// DownloadBinaryProvider implements BinaryProvider by downloading from GitHub releases
type DownloadBinaryProvider struct {
	downloader *utils.BinaryDownloader
	version    string
}

// NewDownloadBinaryProvider creates a new provider that downloads binaries
func NewDownloadBinaryProvider(version string) (*DownloadBinaryProvider, error) {
	downloader, err := utils.NewBinaryDownloader()
	if err != nil {
		return nil, fmt.Errorf("failed to create binary downloader: %w", err)
	}

	return &DownloadBinaryProvider{
		downloader: downloader,
		version:    version,
	}, nil
}

// ExtractServerBinary downloads and returns the server binary for the specified platform
func (dp *DownloadBinaryProvider) ExtractServerBinary(platform *v1.Platform) ([]byte, fs.FileInfo, error) {
	binaryPath, err := dp.downloader.GetBinary(dp.version, platform.OS, platform.Architecture)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get binary for platform %s/%s: %w", platform.OS, platform.Architecture, err)
	}

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read binary: %w", err)
	}

	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat binary: %w", err)
	}

	return data, fileInfo, nil
}

package builder

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type mockFileSystem struct {
	mock.Mock
}

func (m *mockFileSystem) Stat(name string) (fs.FileInfo, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(fs.FileInfo), args.Error(1)
}

func (m *mockFileSystem) ReadFile(name string) ([]byte, error) {
	args := m.Called(name)
	return args.Get(0).([]byte), args.Error(1)
}

type mockBinaryProvider struct {
	mock.Mock
}

func (m *mockBinaryProvider) ExtractServerBinary(platform *v1.Platform) ([]byte, fs.FileInfo, error) {
	args := m.Called(platform)
	if args.Get(1) == nil {
		return args.Get(0).([]byte), nil, args.Error(2)
	}
	return args.Get(0).([]byte), args.Get(1).(fs.FileInfo), args.Error(2)
}

type mockImageDownloader struct {
	mock.Mock
}

func (m *mockImageDownloader) DownloadImage(ctx context.Context, baseImage string, platform *v1.Platform) (v1.Image, error) {
	args := m.Called(ctx, baseImage, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(v1.Image), args.Error(1)
}

type mockImageSaver struct {
	mock.Mock
}

func (m *mockImageSaver) SaveImage(ctx context.Context, img v1.Image, ref string) error {
	args := m.Called(ctx, img, ref)
	return args.Error(0)
}

func (m *mockImageSaver) SaveImageIndex(ctx context.Context, idx v1.ImageIndex, ref string) error {
	args := m.Called(ctx, idx, ref)
	return args.Error(0)
}

type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// testImage is a minimal image implementation using partial package
type testImage struct {
	mediaType types.MediaType
}

func (t *testImage) MediaType() (types.MediaType, error) {
	return t.mediaType, nil
}

func (t *testImage) RawConfigFile() ([]byte, error) {
	return []byte(`{
		"architecture": "amd64",
		"os": "linux",
		"config": {
			"Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
			"User": "root"
		},
		"rootfs": {
			"type": "layers",
			"diff_ids": []
		}
	}`), nil
}

func (t *testImage) RawManifest() ([]byte, error) {
	return []byte(`{
		"schemaVersion": 2,
		"mediaType": "` + string(t.mediaType) + `",
		"config": {
			"size": 1469,
			"digest": "sha256:test"
		},
		"layers": []
	}`), nil
}

func (t *testImage) LayerByDiffID(diffID v1.Hash) (partial.UncompressedLayer, error) {
	return nil, errors.New("no layers in test image")
}

func newTestImage(mediaType types.MediaType) v1.Image {
	img, err := partial.UncompressedToImage(&testImage{mediaType: mediaType})
	if err != nil {
		panic(err) // This should never happen in tests
	}
	return img
}

func TestImageBuilder_Build(t *testing.T) {
	tt := []struct {
		name           string
		buildOptions   BuildOptions
		setupMocks     func(*mockFileSystem, *mockBinaryProvider, *mockImageDownloader)
		expectedError  string
		validateResult func(t *testing.T, img v1.Image)
	}{
		{
			name: "successful build with default options",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
				ImageTag:               "test:latest",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				// Mock base image download
				baseImg := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)

				// Mock binary extraction
				binaryData := []byte("fake-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				// Mock MCP file operations
				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: test-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil)
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return(mcpToolDefsData, nil)

				// Mock MCP server config file operations
				mcpServerConfigData := []byte("fake-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/test/mcpserver.yaml").Return(mcpServerConfigInfo, nil)
				mfs.On("ReadFile", "/test/mcpserver.yaml").Return(mcpServerConfigData, nil)
			},
			validateResult: func(t *testing.T, img v1.Image) {
				assert.NotNil(t, img, "should return a valid image")
			},
		},
		{
			name: "sets mcp image tag correctly",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
				ImageTag:               "test:latest",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				// Mock base image download
				baseImg := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)

				// Mock binary extraction
				binaryData := []byte("fake-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				// Mock MCP file operations
				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: test-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil)
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return(mcpToolDefsData, nil)

				// Mock MCP server config file operations
				mcpServerConfigData := []byte("fake-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/test/mcpserver.yaml").Return(mcpServerConfigInfo, nil)
				mfs.On("ReadFile", "/test/mcpserver.yaml").Return(mcpServerConfigData, nil)
			},
			validateResult: func(t *testing.T, img v1.Image) {
				configFile, err := img.ConfigFile()
				assert.NoError(t, err)

				// Verify config labels (for container runtime)
				assert.Equal(t, "test-server", configFile.Config.Labels[McpServerNameLabel])
				assert.Equal(t, "test-server", configFile.Config.Labels[ImageTitleLabel])

				// Verify manifest annotations (for registry display like Quay.io)
				manifest, err := img.Manifest()
				assert.NoError(t, err)
				assert.Equal(t, "test-server", manifest.Annotations[McpServerNameLabel])
				assert.Equal(t, "test-server", manifest.Annotations[ImageTitleLabel])
			},
		},
		{
			name: "build with custom platform",
			buildOptions: BuildOptions{
				Platform:               &v1.Platform{OS: "windows", Architecture: "amd64"},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				baseImg := newTestImage(types.OCIManifestSchema1)
				mid.On("DownloadImage", mock.Anything, "custom:base", &v1.Platform{OS: "windows", Architecture: "amd64"}).Return(baseImg, nil)

				binaryData := []byte("windows-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server.exe", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "windows", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: custom-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/custom/mcpfile.yaml").Return(mcpToolDefsInfo, nil)
				mfs.On("ReadFile", "/custom/mcpfile.yaml").Return(mcpToolDefsData, nil)

				mcpServerConfigData := []byte("custom-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/custom/mcpserver.yaml").Return(mcpServerConfigInfo, nil)
				mfs.On("ReadFile", "/custom/mcpserver.yaml").Return(mcpServerConfigData, nil)
			},
			validateResult: func(t *testing.T, img v1.Image) {
				assert.NotNil(t, img, "should return a valid image")
			},
		},
		{
			name: "failure - base image download error",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(nil, errors.New("download failed"))
			},
			expectedError: "failed to download base image: download failed",
		},
		{
			name: "failure - binary extraction error",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				baseImg := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return([]byte{}, nil, errors.New("binary not found"))
			},
			expectedError: "failed to extract server binary: binary not found",
		},
		{
			name: "failure - MCP file stat error",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/nonexistent/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				baseImg := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)

				binaryData := []byte("fake-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				mfs.On("Stat", "/nonexistent/mcpfile.yaml").Return(nil, errors.New("file not found"))
			},
			expectedError: "failed to stat MCP file: file not found",
		},
		{
			name: "failure - MCP file read error",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				baseImg := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)

				binaryData := []byte("fake-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: 100}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil)
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return([]byte{}, errors.New("read permission denied"))
			},
			expectedError: "failed to read MCP file: read permission denied",
		},
		{
			name: "failure - unsupported base image media type",
			buildOptions: BuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				baseImg := newTestImage("application/vnd.unsupported")
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImg, nil)

				binaryData := []byte("fake-binary-data")
				binaryInfo := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryData))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryData, binaryInfo, nil)

				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: test-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil)
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return(mcpToolDefsData, nil)

				mcpServerConfigData := []byte("fake-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/test/mcpserver.yaml").Return(mcpServerConfigInfo, nil)
				mfs.On("ReadFile", "/test/mcpserver.yaml").Return(mcpServerConfigData, nil)
			},
			expectedError: "failed to get media type for layers: invalid base image media type",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup mocks
			mockFS := &mockFileSystem{}
			mockBP := &mockBinaryProvider{}
			mockID := &mockImageDownloader{}
			mockIS := &mockImageSaver{}

			tc.setupMocks(mockFS, mockBP, mockID)

			// Create builder with mocked dependencies
			builder := &ImageBuilder{
				fs:              mockFS,
				binaryProvider:  mockBP,
				imageDownloader: mockID,
				imageSaver:      mockIS,
			}

			// Execute test
			ctx := context.Background()
			result, err := builder.Build(ctx, tc.buildOptions)

			// Validate results
			if tc.expectedError != "" {
				assert.Error(t, err, "should return an error")
				assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text")
				assert.Nil(t, result, "should not return a result on error")
			} else {
				assert.NoError(t, err, "should not return an error")
				if tc.validateResult != nil {
					tc.validateResult(t, result)
				}
			}

			// Verify all expectations were met
			mockFS.AssertExpectations(t)
			mockBP.AssertExpectations(t)
			mockID.AssertExpectations(t)
		})
	}
}

func TestImageBuilder_Save(t *testing.T) {
	tt := []struct {
		name          string
		imageRef      string
		setupMocks    func(*mockImageSaver)
		expectedError string
	}{
		{
			name:     "successful push",
			imageRef: "docker.io/test/image:latest",
			setupMocks: func(mis *mockImageSaver) {
				mis.On("SaveImage", mock.Anything, mock.Anything, "docker.io/test/image:latest").Return(nil)
			},
		},
		{
			name:     "push failure",
			imageRef: "registry.example.com/test/image:v1.0.0",
			setupMocks: func(mis *mockImageSaver) {
				mis.On("SaveImage", mock.Anything, mock.Anything, "registry.example.com/test/image:v1.0.0").Return(errors.New("push failed"))
			},
			expectedError: "push failed",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup mocks
			mockIS := &mockImageSaver{}
			tc.setupMocks(mockIS)

			// Create builder with mocked registry client
			builder := &ImageBuilder{
				imageSaver: mockIS,
			}

			// Create fake image for testing
			img := &fake.FakeImage{}

			// Execute test
			ctx := context.Background()
			err := builder.Save(ctx, img, tc.imageRef)

			// Validate results
			if tc.expectedError != "" {
				assert.Error(t, err, "should return an error")
				assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text")
			} else {
				assert.NoError(t, err, "should not return an error")
			}

			// Verify all expectations were met
			mockIS.AssertExpectations(t)
		})
	}
}

func TestBuildOptions_SetDefaults(t *testing.T) {
	tt := []struct {
		name           string
		input          BuildOptions
		expectedOutput BuildOptions
	}{
		{
			name:  "empty options should get defaults",
			input: BuildOptions{},
			expectedOutput: BuildOptions{
				BaseImage: DefaultBaseImage,
				Platform:  &v1.Platform{OS: "linux", Architecture: "amd64"},
			},
		},
		{
			name: "partial options should only set missing defaults",
			input: BuildOptions{
				BaseImage: "custom:image",
			},
			expectedOutput: BuildOptions{
				BaseImage: "custom:image",
				Platform:  &v1.Platform{OS: "linux", Architecture: "amd64"},
			},
		},
		{
			name: "full options should remain unchanged",
			input: BuildOptions{
				Platform:               &v1.Platform{OS: "windows", Architecture: "arm64"},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/path/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/path/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
			expectedOutput: BuildOptions{
				Platform:               &v1.Platform{OS: "windows", Architecture: "arm64"},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/path/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/path/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.input.SetDefaults()
			assert.Equal(t, tc.expectedOutput, tc.input, "SetDefaults should produce expected output")
		})
	}
}

func TestGetLayerMediaType(t *testing.T) {
	tt := []struct {
		name          string
		setupImage    func() v1.Image
		expectedType  types.MediaType
		expectedError string
	}{
		{
			name: "OCI manifest should return OCI layer type",
			setupImage: func() v1.Image {
				return newTestImage(types.OCIManifestSchema1)
			},
			expectedType: types.OCILayer,
		},
		{
			name: "Docker manifest should return Docker layer type",
			setupImage: func() v1.Image {
				return newTestImage(types.DockerManifestSchema2)
			},
			expectedType: types.DockerLayer,
		},
		{
			name: "unsupported media type should return error",
			setupImage: func() v1.Image {
				return newTestImage("application/vnd.unsupported")
			},
			expectedError: "invalid base image media type",
		},
		{
			name: "media type retrieval error should be propagated",
			setupImage: func() v1.Image {
				img := &fake.FakeImage{}
				img.MediaTypeReturns("", errors.New("media type error"))
				return img
			},
			expectedError: "media type error",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := &ImageBuilder{}
			img := tc.setupImage()

			result, err := builder.getLayerMediaType(img)

			if tc.expectedError != "" {
				assert.Error(t, err, "should return an error")
				assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text")
			} else {
				assert.NoError(t, err, "should not return an error")
				assert.Equal(t, tc.expectedType, result, "should return expected media type")
			}
		})
	}
}

func TestExtractTagFromReference(t *testing.T) {
	tt := []struct {
		name      string
		reference string
		expected  string
	}{
		{
			name:      "reference with tag",
			reference: "docker.io/library/nginx:1.21",
			expected:  "1.21",
		},
		{
			name:      "reference with multiple colons",
			reference: "localhost:5000/my-image:v1.0.0",
			expected:  "v1.0.0",
		},
		{
			name:      "reference without tag",
			reference: "docker.io/library/nginx",
			expected:  "",
		},
		{
			name:      "empty reference",
			reference: "",
			expected:  "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractTagFromReference(tc.reference)
			assert.Equal(t, tc.expected, result, "should extract correct tag")
		})
	}
}

func TestImageBuilder_BuildMultiArch(t *testing.T) {
	tt := []struct {
		name           string
		buildOptions   MultiArchBuildOptions
		setupMocks     func(*mockFileSystem, *mockBinaryProvider, *mockImageDownloader)
		expectedError  string
		validateResult func(t *testing.T, idx v1.ImageIndex)
	}{
		{
			name: "successful multi-arch build with default platforms",
			buildOptions: MultiArchBuildOptions{
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
				ImageTag:               "test:latest",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				// Mock for linux/amd64
				baseImgAmd64 := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImgAmd64, nil)

				binaryDataAmd64 := []byte("fake-binary-amd64")
				binaryInfoAmd64 := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryDataAmd64))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryDataAmd64, binaryInfoAmd64, nil)

				// Mock for linux/arm64
				baseImgArm64 := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "arm64"}).Return(baseImgArm64, nil)

				binaryDataArm64 := []byte("fake-binary-arm64")
				binaryInfoArm64 := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryDataArm64))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "arm64"}).Return(binaryDataArm64, binaryInfoArm64, nil)

				// Mock MCP file operations
				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: test-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil).Times(2)
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return(mcpToolDefsData, nil).Times(2)

				// Mock MCP server config file operations
				mcpServerConfigData := []byte("fake-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/test/mcpserver.yaml").Return(mcpServerConfigInfo, nil).Times(2)
				mfs.On("ReadFile", "/test/mcpserver.yaml").Return(mcpServerConfigData, nil).Times(2)
			},
			validateResult: func(t *testing.T, idx v1.ImageIndex) {
				assert.NotNil(t, idx, "should return a valid image index")

				manifest, err := idx.IndexManifest()
				assert.NoError(t, err, "should be able to get index manifest")
				assert.Len(t, manifest.Manifests, 2, "should have 2 platform manifests")

				platforms := make(map[string]bool)
				for _, desc := range manifest.Manifests {
					if desc.Platform != nil {
						key := desc.Platform.OS + "/" + desc.Platform.Architecture
						platforms[key] = true
					}
				}
				assert.True(t, platforms["linux/amd64"], "should have linux/amd64")
				assert.True(t, platforms["linux/arm64"], "should have linux/arm64")
			},
		},
		{
			name: "successful multi-arch build with custom platforms",
			buildOptions: MultiArchBuildOptions{
				Platforms: []*v1.Platform{
					{OS: "linux", Architecture: "amd64"},
					{OS: "windows", Architecture: "amd64"},
				},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				// Mock for linux/amd64
				baseImgLinux := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, "custom:base", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImgLinux, nil)

				binaryDataLinux := []byte("linux-binary")
				binaryInfoLinux := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryDataLinux))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryDataLinux, binaryInfoLinux, nil)

				// Mock for windows/amd64
				baseImgWindows := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, "custom:base", &v1.Platform{OS: "windows", Architecture: "amd64"}).Return(baseImgWindows, nil)

				binaryDataWindows := []byte("windows-binary")
				binaryInfoWindows := &mockFileInfo{name: "genmcp-server.exe", size: int64(len(binaryDataWindows))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "windows", Architecture: "amd64"}).Return(binaryDataWindows, binaryInfoWindows, nil)

				// Mock MCP file operations
				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: custom-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/custom/mcpfile.yaml").Return(mcpToolDefsInfo, nil).Times(2)
				mfs.On("ReadFile", "/custom/mcpfile.yaml").Return(mcpToolDefsData, nil).Times(2)

				// Mock MCP server config file operations
				mcpServerConfigData := []byte("custom-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/custom/mcpserver.yaml").Return(mcpServerConfigInfo, nil).Times(2)
				mfs.On("ReadFile", "/custom/mcpserver.yaml").Return(mcpServerConfigData, nil).Times(2)
			},
			validateResult: func(t *testing.T, idx v1.ImageIndex) {
				assert.NotNil(t, idx, "should return a valid image index")

				manifest, err := idx.IndexManifest()
				assert.NoError(t, err, "should be able to get index manifest")
				assert.Len(t, manifest.Manifests, 2, "should have 2 platform manifests")
			},
		},
		{
			name: "failure - one platform build fails",
			buildOptions: MultiArchBuildOptions{
				Platforms: []*v1.Platform{
					{OS: "linux", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
				MCPToolDefinitionsPath: "/test/mcpfile.yaml",
				MCPServerConfigPath:    "/test/mcpserver.yaml",
			},
			setupMocks: func(mfs *mockFileSystem, mbp *mockBinaryProvider, mid *mockImageDownloader) {
				// First platform succeeds
				baseImgAmd64 := newTestImage(types.DockerManifestSchema2)
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(baseImgAmd64, nil)

				binaryDataAmd64 := []byte("fake-binary-amd64")
				binaryInfoAmd64 := &mockFileInfo{name: "genmcp-server", size: int64(len(binaryDataAmd64))}
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "amd64"}).Return(binaryDataAmd64, binaryInfoAmd64, nil)

				mcpToolDefsData := []byte("kind: MCPToolDefinitions\nschemaVersion: 0.2.0\nname: test-server\nversion: 1.0.0\n")
				mcpToolDefsInfo := &mockFileInfo{name: "mcpfile.yaml", size: int64(len(mcpToolDefsData))}
				mfs.On("Stat", "/test/mcpfile.yaml").Return(mcpToolDefsInfo, nil).Maybe()
				mfs.On("ReadFile", "/test/mcpfile.yaml").Return(mcpToolDefsData, nil).Maybe()

				mcpServerConfigData := []byte("fake-mcp-mcpserver-data")
				mcpServerConfigInfo := &mockFileInfo{name: "mcpserver.yaml", size: int64(len(mcpServerConfigData))}
				mfs.On("Stat", "/test/mcpserver.yaml").Return(mcpServerConfigInfo, nil).Maybe()
				mfs.On("ReadFile", "/test/mcpserver.yaml").Return(mcpServerConfigData, nil).Maybe()

				// Second platform fails
				mid.On("DownloadImage", mock.Anything, DefaultBaseImage, &v1.Platform{OS: "linux", Architecture: "arm64"}).Return(newTestImage(types.DockerManifestSchema2), nil)
				mbp.On("ExtractServerBinary", &v1.Platform{OS: "linux", Architecture: "arm64"}).Return([]byte{}, nil, errors.New("arm64 binary not found"))
			},
			expectedError: "failed to build image for platform linux/arm64",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFS := &mockFileSystem{}
			mockBP := &mockBinaryProvider{}
			mockID := &mockImageDownloader{}
			mockIS := &mockImageSaver{}

			tc.setupMocks(mockFS, mockBP, mockID)

			builder := &ImageBuilder{
				fs:              mockFS,
				binaryProvider:  mockBP,
				imageDownloader: mockID,
				imageSaver:      mockIS,
			}

			ctx := context.Background()
			result, err := builder.BuildMultiArch(ctx, tc.buildOptions)

			if tc.expectedError != "" {
				assert.Error(t, err, "should return an error")
				assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text")
				assert.Nil(t, result, "should not return a result on error")
			} else {
				assert.NoError(t, err, "should not return an error")
				if tc.validateResult != nil {
					tc.validateResult(t, result)
				}
			}

			mockFS.AssertExpectations(t)
			mockBP.AssertExpectations(t)
			mockID.AssertExpectations(t)
		})
	}
}

func TestImageBuilder_SaveIndex(t *testing.T) {
	tt := []struct {
		name          string
		imageRef      string
		setupMocks    func(*mockImageSaver)
		expectedError string
	}{
		{
			name:     "successful push to registry",
			imageRef: "docker.io/test/image:latest",
			setupMocks: func(mis *mockImageSaver) {
				mis.On("SaveImageIndex", mock.Anything, mock.Anything, "docker.io/test/image:latest").Return(nil)
			},
		},
		{
			name:     "push failure",
			imageRef: "registry.example.com/test/image:v1.0.0",
			setupMocks: func(mis *mockImageSaver) {
				mis.On("SaveImageIndex", mock.Anything, mock.Anything, "registry.example.com/test/image:v1.0.0").Return(errors.New("push failed"))
			},
			expectedError: "push failed",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockIS := &mockImageSaver{}
			tc.setupMocks(mockIS)

			builder := &ImageBuilder{
				imageSaver: mockIS,
			}

			idx := mutate.IndexMediaType(empty.Index, types.DockerManifestList)

			ctx := context.Background()
			err := builder.SaveIndex(ctx, idx, tc.imageRef)

			if tc.expectedError != "" {
				assert.Error(t, err, "should return an error")
				assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text")
			} else {
				assert.NoError(t, err, "should not return an error")
			}

			mockIS.AssertExpectations(t)
		})
	}
}

func TestMultiArchBuildOptions_SetDefaults(t *testing.T) {
	tt := []struct {
		name           string
		input          MultiArchBuildOptions
		expectedOutput MultiArchBuildOptions
	}{
		{
			name:  "empty options should get defaults",
			input: MultiArchBuildOptions{},
			expectedOutput: MultiArchBuildOptions{
				BaseImage: DefaultBaseImage,
				Platforms: []*v1.Platform{
					{OS: "linux", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
			},
		},
		{
			name: "partial options should only set missing defaults",
			input: MultiArchBuildOptions{
				BaseImage: "custom:image",
			},
			expectedOutput: MultiArchBuildOptions{
				BaseImage: "custom:image",
				Platforms: []*v1.Platform{
					{OS: "linux", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
			},
		},
		{
			name: "full options should remain unchanged",
			input: MultiArchBuildOptions{
				Platforms: []*v1.Platform{
					{OS: "windows", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/path/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/path/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
			expectedOutput: MultiArchBuildOptions{
				Platforms: []*v1.Platform{
					{OS: "windows", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
				BaseImage:              "custom:base",
				MCPToolDefinitionsPath: "/custom/path/mcpfile.yaml",
				MCPServerConfigPath:    "/custom/path/mcpserver.yaml",
				ImageTag:               "custom:tag",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.input.SetDefaults()
			assert.Equal(t, tc.expectedOutput.BaseImage, tc.input.BaseImage, "BaseImage should match")
			assert.Equal(t, tc.expectedOutput.MCPToolDefinitionsPath, tc.input.MCPToolDefinitionsPath, "MCPToolDefinitionsPath should match")
			assert.Equal(t, tc.expectedOutput.MCPServerConfigPath, tc.input.MCPServerConfigPath, "MCPServerConfigPath should match")
			assert.Equal(t, tc.expectedOutput.ImageTag, tc.input.ImageTag, "ImageTag should match")
			assert.Equal(t, len(tc.expectedOutput.Platforms), len(tc.input.Platforms), "Platforms length should match")

			for i, expectedPlatform := range tc.expectedOutput.Platforms {
				assert.Equal(t, expectedPlatform.OS, tc.input.Platforms[i].OS, "Platform OS should match")
				assert.Equal(t, expectedPlatform.Architecture, tc.input.Platforms[i].Architecture, "Platform Architecture should match")
			}
		})
	}
}

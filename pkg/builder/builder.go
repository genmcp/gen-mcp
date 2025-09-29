package builder

import (
	"archive/tar"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// FileSystem interface for file operations
type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	ReadFile(name string) ([]byte, error)
}

// BinaryProvider interface for accessing server binaries
type BinaryProvider interface {
	ExtractServerBinary(platform *v1.Platform) ([]byte, fs.FileInfo, error)
}

// RegistryClient interface for container registry operations
type RegistryClient interface {
	DownloadImage(ctx context.Context, baseImage string, platform *v1.Platform) (v1.Image, error)
	PushImage(ctx context.Context, img v1.Image, ref string) error
}

// OSFileSystem implements FileSystem using the standard os package
type OSFileSystem struct{}

func (fs *OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (fs *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// EmbedBinaryProvider implements BinaryProvider using embedded binaries
type EmbedBinaryProvider struct {
	binaries embed.FS
}

func (bp *EmbedBinaryProvider) ExtractServerBinary(platform *v1.Platform) ([]byte, fs.FileInfo, error) {
	filename := fmt.Sprintf("binaries/genmcp-server-%s-%s", platform.OS, platform.Architecture)
	if platform.OS == "windows" {
		filename += ".exe"
	}

	fileInfo, err := fs.Stat(bp.binaries, filename)
	if err != nil {
		return nil, nil, err
	}

	binary, err := bp.binaries.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("no binary found for platform %s/%s", platform.OS, platform.Architecture)
	}

	return binary, fileInfo, nil
}

// DefaultRegistryClient implements RegistryClient using go-containerregistry
type DefaultRegistryClient struct{}

func (rc *DefaultRegistryClient) DownloadImage(ctx context.Context, baseImage string, platform *v1.Platform) (v1.Image, error) {
	ref, err := name.ParseReference(baseImage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base image name %s: %w", baseImage, err)
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithPlatform(*platform))
	if err != nil {
		return nil, fmt.Errorf("failed to pull base image %s, %w", baseImage, err)
	}

	return img, nil
}

func (rc *DefaultRegistryClient) PushImage(ctx context.Context, img v1.Image, ref string) error {
	repo, err := name.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("invalid reference %s: %w", ref, err)
	}

	if err = remote.Write(repo, img,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	); err != nil {
		return fmt.Errorf("failed to push image to %s: %w", ref, err)
	}

	return nil
}

//go:embed binaries/genmcp-server-*
var serverBinaries embed.FS

// Magic value required to make file exexutable in windows containers
// taken from https://github.com/ko-build/ko/blob/4cee0bb4ee9655f43cc2ef26dbe0f45fac1eda5c/pkg/build/gobuild.go#L591
const userOwnerAndGroupSID = "AQAAgBQAAAAkAAAAAAAAAAAAAAABAgAAAAAABSAAAAAhAgAAAQIAAAAAAAUgAAAAIQIAAA=="

// various standard oci labels
const (
	ImageTitleLabel       = "org.opencontainers.image.title"
	ImageDescriptionLabel = "org.opencontainers.image.description"
	ImageCreatedLabel     = "org.opencontainers.image.created"
	ImageRefNameLabel     = "org.opencontainers.image.ref.name"
	ImageVersionLabel     = "org.opencontainers.image.ref.version"
)

type ImageBuilder struct {
	fs             FileSystem
	binaryProvider BinaryProvider
	registryClient RegistryClient
}

func New() *ImageBuilder {
	return &ImageBuilder{
		fs:             &OSFileSystem{},
		binaryProvider: &EmbedBinaryProvider{binaries: serverBinaries},
		registryClient: &DefaultRegistryClient{},
	}
}

func (b *ImageBuilder) Build(ctx context.Context, opts BuildOptions) (v1.Image, error) {
	opts.SetDefaults()

	baseImg, err := b.registryClient.DownloadImage(ctx, opts.BaseImage, opts.Platform)
	if err != nil {
		return nil, fmt.Errorf("failed to download base image: %w", err)
	}

	serverBinary, serverBinaryInfo, err := b.binaryProvider.ExtractServerBinary(opts.Platform)
	if err != nil {
		return nil, fmt.Errorf("failed to extract server binary: %w", err)
	}

	mcpFileInfo, err := b.fs.Stat(opts.MCPFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat MCPFile: %w", err)
	}

	mcpFileData, err := b.fs.ReadFile(opts.MCPFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCPFile: %w", err)
	}

	mediaType, err := b.getLayerMediaType(baseImg)
	if err != nil {
		return nil, fmt.Errorf("failed to get media type for layers: %w", err)
	}

	binaryLayer, err := b.createBinaryLayer(serverBinary, serverBinaryInfo, opts.Platform, mediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer for genmcp-server binary: %w", err)
	}

	mcpFileLayer, err := b.createMCPFileLayer(mcpFileData, mcpFileInfo, opts.Platform, mediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer for mcpfile.yaml: %w", err)
	}

	img, err := b.assembleImage(baseImg, opts, binaryLayer, mcpFileLayer)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble final image: %w", err)
	}

	return img, nil
}

func (b *ImageBuilder) Push(ctx context.Context, img v1.Image, ref string) error {
	return b.registryClient.PushImage(ctx, img, ref)
}


func (b *ImageBuilder) getLayerMediaType(baseImg v1.Image) (types.MediaType, error) {
	mt, err := baseImg.MediaType()
	if err != nil {
		return "", err
	}

	switch mt {
	case types.OCIManifestSchema1:
		return types.OCILayer, nil
	case types.DockerManifestSchema2:
		return types.DockerLayer, nil
	default:
		return "", fmt.Errorf("invalid base image media type '%s' expected one of '%s' or '%s'", mt, types.OCIManifestSchema1, types.DockerManifestSchema2)
	}
}


func (b *ImageBuilder) assembleImage(baseImg v1.Image, opts BuildOptions, layers ...v1.Layer) (v1.Image, error) {
	img, err := mutate.AppendLayers(baseImg, layers...)
	if err != nil {
		return nil, fmt.Errorf("failed to add layers to base image: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config while building image: %w", err)
	}

	createTime := time.Now()

	cfg = cfg.DeepCopy()
	cfg.Config.Entrypoint = []string{"/usr/local/bin/genmcp-server"}
	cfg.Config.WorkingDir = "/app"
	cfg.Config.Env = append(cfg.Config.Env, "MCP_FILE_PATH=/app/mcpfile.yaml")
	cfg.Config.User = "1001:1001"
	cfg.Created = v1.Time{Time: createTime}

	if cfg.Config.Labels == nil {
		cfg.Config.Labels = make(map[string]string)
	}

	// add standard OCI labels
	cfg.Config.Labels[ImageTitleLabel] = "genmcp-server"
	cfg.Config.Labels[ImageDescriptionLabel] = "GenMCP Server Image"
	cfg.Config.Labels[ImageCreatedLabel] = createTime.Format(time.RFC3339)

	if opts.ImageTag != "" {
		cfg.Config.Labels[ImageRefNameLabel] = opts.ImageTag

		if tag := extractTagFromReference(opts.ImageTag); tag != "" {
			cfg.Config.Labels[ImageVersionLabel] = tag
		}
	}

	return mutate.ConfigFile(img, cfg)
}

// createBinaryLayer creates a tarball layer with the genmcp-server binary at /usr/local/bin/genmcp-server
func (b *ImageBuilder) createBinaryLayer(
	binaryData []byte,
	fileInfo fs.FileInfo,
	platform *v1.Platform,
	layerMediaType types.MediaType,
) (v1.Layer, error) {
	layerData, err := createTarWithFile("/usr/local/bin", "genmcp-server", platform.OS, binaryData, fileInfo, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer for genmcp-server binary: %w", err)
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(layerData.Bytes())), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(layerMediaType))
}

// createMCPFileLayer creates a tarball layer with the mcpfile.yaml at /app/mcpfile.yaml
func (b *ImageBuilder) createMCPFileLayer(
	mcpFileData []byte,
	fileInfo fs.FileInfo,
	platform *v1.Platform,
	layerMediaType types.MediaType,
) (v1.Layer, error) {
	layerData, err := createTarWithFile("/app", "mcpfile.yaml", platform.OS, mcpFileData, fileInfo, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer for mcpfile.yaml: %w", err)
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(layerData.Bytes())), nil
	}, tarball.WithCompressedCaching, tarball.WithMediaType(layerMediaType))
}

func createTarWithFile(filepath, filename, os string, data []byte, fileInfo fs.FileInfo, mode int64) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	if err := tw.WriteHeader(&tar.Header{
		Name:     filepath,
		Typeflag: tar.TypeDir,
		Mode:     0555,
	}); err != nil {
		return nil, fmt.Errorf("failed to write dir %s to tar: %w", filepath, err)
	}

	header := &tar.Header{
		Name:       filepath + "/" + filename,
		Size:       fileInfo.Size(),
		Typeflag:   tar.TypeReg,
		Mode:       mode,
		PAXRecords: map[string]string{},
	}

	if os == "windows" {
		// need to set magic value for the binary to be executable
		header.PAXRecords["MSWINDOWS.rawsd"] = userOwnerAndGroupSID
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to write header for file %s to tar: %w", filename, err)
	}

	if _, err := tw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data for file %s to tar: %w", filename, err)
	}

	return buf, nil
}

func extractTagFromReference(reference string) string {
	parts := strings.Split(reference, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return ""
}

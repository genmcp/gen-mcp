package builder

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const DefaultBaseImage = "registry.access.redhat.com/ubi9/ubi-minimal:latest"

type BuildOptions struct {
	Platform    *v1.Platform // Target platform (linux/amd64, etc.)
	BaseImage   string       // Base image reference
	MCPFilePath string       // path to the mcp file
	ImageTag    string       // output image tag
}

func (o *BuildOptions) SetDefaults() {
	if o.BaseImage == "" {
		o.BaseImage = DefaultBaseImage
	}
	if o.Platform == nil {
		o.Platform = &v1.Platform{OS: "linux", Architecture: "amd64"}
	}
}

type MultiArchBuildOptions struct {
	Platforms   []*v1.Platform // Target platforms
	BaseImage   string         // Base image reference
	MCPFilePath string         // path to the mcp file
	ImageTag    string         // output image tag
}

func (o *MultiArchBuildOptions) SetDefaults() {
	if o.BaseImage == "" {
		o.BaseImage = DefaultBaseImage
	}
	if len(o.Platforms) == 0 {
		o.Platforms = []*v1.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm64"},
		}
	}
}

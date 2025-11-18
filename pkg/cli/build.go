package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/builder"
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&baseImage, "base-image", "", "base image to build the genmcp image on top of")
	buildCmd.Flags().StringVarP(&mcpToolDefinitionsPath, "file", "f", "mcpfile.yaml", "MCP tool definitions file")
	// TODO: rename
	buildCmd.Flags().StringVarP(&mcpServerConfigPath, "server-config", "s", "mcpserver.yaml", "MCP server configuration file")
	buildCmd.Flags().StringVar(&platform, "platform", "", "platform to build for (e.g., linux/amd64). If not specified, builds multi-arch image for linux/amd64 and linux/arm64")
	buildCmd.Flags().StringVar(&imageTag, "tag", "", "image tag for the registry")
	buildCmd.Flags().BoolVar(&push, "push", false, "push the image to the registry (if false, store locally)")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build container image and save locally or push to registry",
	Run:   executeBuildCmd,
}

var (
	baseImage              string
	mcpToolDefinitionsPath string
	mcpServerConfigPath    string
	platform               string
	imageTag               string
	push                   bool
)

func executeBuildCmd(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	if imageTag == "" {
		fmt.Printf("--tag is required to build an image\n")
		os.Exit(1)
	}

	// Validate MCP files before building
	if err := validateMCPToolDefinitionsFile(mcpToolDefinitionsPath); err != nil {
		fmt.Printf("invalid MCP tool definitions file: %s\n", err.Error())
		os.Exit(1)
	}
	if err := validateMCPServerConfigFile(mcpServerConfigPath); err != nil {
		fmt.Printf("invalid MCP server config file: %s\n", err.Error())
		os.Exit(1)
	}

	b := builder.New(push)

	// Single platform build if --platform is specified
	if platform != "" {
		parsedPlatform, err := v1.ParsePlatform(platform)
		if err != nil {
			fmt.Printf("failed to parse platform '%s': %s\n", platform, err.Error())
			os.Exit(1)
		}

		fmt.Printf("building image for %s...\n", platform)
		opts := builder.BuildOptions{
			Platform:               parsedPlatform,
			BaseImage:              baseImage,
			MCPToolDefinitionsPath: mcpToolDefinitionsPath,
			MCPServerConfigPath:    mcpServerConfigPath,
			ImageTag:               imageTag,
		}

		img, err := b.Build(ctx, opts)
		if err != nil {
			fmt.Printf("failed to build image: %s\n", err.Error())
			os.Exit(1)
		}

		if push {
			fmt.Printf("successfully built image!\npushing image to %s...\n", imageTag)
		} else {
			fmt.Printf("successfully built image!\nsaving image to local container engine as %s...\n", imageTag)
		}

		if err := b.Save(ctx, img, imageTag); err != nil {
			if push {
				fmt.Printf("failed to push image - ensure you are logged in: %s\n", err.Error())
			} else {
				fmt.Printf("failed to save image to local container engine: %s\n", err.Error())
			}
			os.Exit(1)
		}

		if push {
			fmt.Printf("successfully pushed %s\n", imageTag)
		} else {
			fmt.Printf("successfully saved %s to local container engine\n", imageTag)
		}
	} else {
		// Multi-arch build (default when --platform not specified)
		platforms := []string{"linux/amd64", "linux/arm64"}
		var parsedPlatforms []*v1.Platform

		for _, p := range platforms {
			parsed, err := v1.ParsePlatform(p)
			if err != nil {
				fmt.Printf("failed to parse platform '%s': %s\n", p, err.Error())
				os.Exit(1)
			}
			parsedPlatforms = append(parsedPlatforms, parsed)
		}

		fmt.Printf("building multi-arch image for platforms: %v...\n", platforms)
		opts := builder.MultiArchBuildOptions{
			Platforms:              parsedPlatforms,
			BaseImage:              baseImage,
			MCPToolDefinitionsPath: mcpToolDefinitionsPath,
			MCPServerConfigPath:    mcpServerConfigPath,
			ImageTag:               imageTag,
		}

		idx, err := b.BuildMultiArch(ctx, opts)
		if err != nil {
			fmt.Printf("failed to build multi-arch image: %s\n", err.Error())
			os.Exit(1)
		}

		if push {
			fmt.Printf("successfully built multi-arch image!\npushing image index to %s...\n", imageTag)
		} else {
			fmt.Printf("successfully built multi-arch image!\nsaving images to local container engine...\n")
			fmt.Printf("note: local daemon doesn't support manifest lists, saving each platform separately\n")
		}

		if err := b.SaveIndex(ctx, idx, imageTag); err != nil {
			if push {
				fmt.Printf("failed to push image index - ensure you are logged in: %s\n", err.Error())
			} else {
				fmt.Printf("failed to save images to local container engine: %s\n", err.Error())
			}
			os.Exit(1)
		}

		if push {
			fmt.Printf("successfully pushed multi-arch image %s\n", imageTag)
		} else {
			fmt.Printf("successfully saved multi-arch images to local container engine\n")
			fmt.Printf("available tags: %s", imageTag)
			for _, p := range platforms {
				tagSuffix := strings.ReplaceAll(p, "/", "-")
				fmt.Printf(", %s-%s", imageTag, tagSuffix)
			}
			fmt.Printf("\n")
		}
	}
}

// validateMCPToolDefinitionsFile validates an MCP tool definitions file
func validateMCPToolDefinitionsFile(filePath string) error {
	// Read the file to check the kind field
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse just the kind field to verify it's the correct type
	var fileKind struct {
		Kind string `json:"kind" yaml:"kind"`
	}
	if err := json.Unmarshal(data, &fileKind); err != nil {
		// Try YAML unmarshaling
		if yamlErr := yaml.Unmarshal(data, &fileKind); yamlErr != nil {
			return fmt.Errorf("failed to parse file (tried both JSON and YAML): %w", err)
		}
	}

	// Only accept MCPToolDefinitionsFile
	if fileKind.Kind != definitions.KindMCPToolDefinitions {
		return fmt.Errorf("expected MCPToolDefinitions file (kind: %s), but found kind: %s", definitions.KindMCPToolDefinitions, fileKind.Kind)
	}

	mcpFile, err := definitions.ParseMCPFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse MCPToolDefinitions file: %w", err)
	}
	// Note: MCPToolDefinitionsFile doesn't have a Validate method that matches
	// the signature expected, so we'll just validate parsing succeeded
	_ = mcpFile
	return nil
}

// validateMCPServerConfigFile validates an MCP server config file
func validateMCPServerConfigFile(filePath string) error {
	// Read the file to check the kind field
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse just the kind field to verify it's the correct type
	var fileKind struct {
		Kind string `json:"kind" yaml:"kind"`
	}
	if err := json.Unmarshal(data, &fileKind); err != nil {
		// Try YAML unmarshaling
		if yamlErr := yaml.Unmarshal(data, &fileKind); yamlErr != nil {
			return fmt.Errorf("failed to parse file (tried both JSON and YAML): %w", err)
		}
	}

	// Only accept MCPServerConfigFile
	if fileKind.Kind != serverconfig.KindMCPServerConfig {
		return fmt.Errorf("expected MCPServerConfig file (kind: %s), but found kind: %s", serverconfig.KindMCPServerConfig, fileKind.Kind)
	}

	mcpFile, err := serverconfig.ParseMCPFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse MCPServerConfig file: %w", err)
	}
	// Note: MCPServerConfigFile doesn't have a Validate method that matches
	// the signature expected, so we'll just validate parsing succeeded
	_ = mcpFile
	return nil
}

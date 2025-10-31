package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/builder"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&baseImage, "base-image", "", "base image to build the genmcp image on top of")
	buildCmd.Flags().StringVarP(&mcpFile, "file", "f", "mcpfile.yaml", "mcp file to build")
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
	baseImage string
	mcpFile   string
	platform  string
	imageTag  string
	push      bool
)

func executeBuildCmd(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	if imageTag == "" {
		fmt.Printf("--tag is required to build an image\n")
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
			Platform:    parsedPlatform,
			BaseImage:   baseImage,
			MCPFilePath: mcpFile,
			ImageTag:    imageTag,
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
			Platforms:   parsedPlatforms,
			BaseImage:   baseImage,
			MCPFilePath: mcpFile,
			ImageTag:    imageTag,
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
			fmt.Printf("available tags: %s (host platform)", imageTag)
			for _, p := range platforms {
				tagSuffix := strings.ReplaceAll(p, "/", "-")
				fmt.Printf(", %s-%s", imageTag, tagSuffix)
			}
			fmt.Printf("\n")
		}
	}
}

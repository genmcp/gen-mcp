package cli

import (
	"fmt"
	"os"

	"github.com/genmcp/gen-mcp/pkg/builder"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&baseImage, "base-image", "", "base image to build the genmcp image on top of")
	buildCmd.Flags().StringVarP(&mcpFile, "file", "f", "mcpfile.yaml", "mcp file to build")
	buildCmd.Flags().StringVar(&platform, "platform", "linux/amd64", "the platform to build genmcp for")
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

	parsedPlatform, err := v1.ParsePlatform(platform)
	if err != nil {
		fmt.Printf("failed to parse platform flag\n")
		os.Exit(1)
	}

	if imageTag == "" {
		fmt.Printf("--tag is required to build an image\n")
		os.Exit(1)
	}

	opts := builder.BuildOptions{
		Platform:    parsedPlatform,
		BaseImage:   baseImage,
		MCPFilePath: mcpFile,
		ImageTag:    imageTag,
	}

	b := builder.New(push)
	fmt.Printf("building image...\n")
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
}

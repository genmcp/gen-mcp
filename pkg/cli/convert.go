package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Cali0707/AutoMCP/pkg/openapi"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&outputPath, "out", "o", "mcpfile.yaml", "the path to write the mcp file to")
	convertCmd.Flags().StringVarP(&host, "host", "H", "", "the base host for the API, if different than in the OpenAPI spec")
}

var outputPath string
var host string

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert an OpenAPI v2/v3 spec into a MCPFile",
	Args:  cobra.ExactArgs(1),
	Run:   executeConvertCmd,
}

func executeConvertCmd(cobraCmd *cobra.Command, args []string) {
	openApiLocation := args[0]

	var openApiBytes []byte
	var err error
	if isRemoteFile(openApiLocation) {
		openApiBytes, err = getOpenApiSpec(openApiLocation)
		if err != nil {
			fmt.Printf("could not retrieve openapi spec from url %s: %s\n", openApiLocation, err.Error())
			return
		}
	} else {
		openApiBytes, err = os.ReadFile(openApiLocation)
		if err != nil {
			fmt.Printf("could not read openapi spec at path %s: %s\n", openApiLocation, err.Error())
			return
		}
	}

	mcpFile, err := openapi.DocumentToMcpFile(openApiBytes, host)
	if err != nil {
		fmt.Printf("encountered errors while converting openapi document to mcp file: %s\n", err.Error())
	}

	mcpFileBytes, err := yaml.Marshal(mcpFile)
	if err != nil {
		fmt.Printf("could not marshal mcp file: %s\n", err.Error())
		return
	}

	err = os.WriteFile(outputPath, mcpFileBytes, 0644)
	if err != nil {
		fmt.Printf("could not write mcpfile to file at path %s: %s", outputPath, err.Error())
	}

}

func getOpenApiSpec(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get openapi spec: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return bodyBytes, nil
}

func isRemoteFile(location string) bool {
	u, err := url.Parse(location)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

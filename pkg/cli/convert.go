package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	"github.com/genmcp/gen-mcp/pkg/converter/openapi"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&toolDefinitionsPath, "file", "f", "mcpfile.yaml", "the path to write the MCP file to")
	convertCmd.Flags().StringVarP(&serverConfigPath, "server-config", "s", "mcpserver.yaml", "the path to write the server config file to")
	convertCmd.Flags().StringVarP(&host, "host", "H", "", "the base host for the API, if different than in the OpenAPI spec")
}

var toolDefinitionsPath string
var serverConfigPath string
var host string

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert an OpenAPI v2/v3 spec into MCP tool definitions and server config files",
	Args:  cobra.ExactArgs(1),
	Run:   executeConvertCmd,
}

func executeConvertCmd(_ *cobra.Command, args []string) {
	openApiLocation := args[0]

	var openApiBytes []byte
	var err error
	if isRemoteFile(openApiLocation) {
		fmt.Printf("INFO    Fetching OpenAPI spec from %s\n", openApiLocation)
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

	convertedFiles, err := openapi.DocumentToMcpFile(openApiBytes, host)
	if err != nil {
		fmt.Printf("encountered errors while converting openapi document to GenMCP config files: %s\n", err.Error())
	}

	if convertedFiles == nil {
		fmt.Printf("conversion failed, no files generated\n")
		return
	}

	// Count converted tools
	numTools := 0
	if convertedFiles.ToolDefinitions != nil && convertedFiles.ToolDefinitions.Tools != nil {
		numTools = len(convertedFiles.ToolDefinitions.Tools)
	}
	fmt.Printf("INFO    Converted %d endpoints to MCP tools\n", numTools)

	// Write MCP file
	toolDefBytes, err := yaml.Marshal(convertedFiles.ToolDefinitions)
	if err != nil {
		fmt.Printf("could not marshal MCP file: %s\n", err.Error())
		return
	}

	toolDefBytes = utils.AppendToolDefinitionsSchemaHeader(toolDefBytes)

	err = os.WriteFile(toolDefinitionsPath, toolDefBytes, 0644)
	if err != nil {
		fmt.Printf("could not write MCP file to path %s: %s\n", toolDefinitionsPath, err.Error())
		return
	}

	fmt.Printf("INFO    Created %s\n", toolDefinitionsPath)

	// Write server config file
	serverConfigBytes, err := yaml.Marshal(convertedFiles.ServerConfig)
	if err != nil {
		fmt.Printf("could not marshal server config file: %s\n", err.Error())
		return
	}

	serverConfigBytes = utils.AppendServerConfigSchemaHeader(serverConfigBytes)

	err = os.WriteFile(serverConfigPath, serverConfigBytes, 0644)
	if err != nil {
		fmt.Printf("could not write server config file to path %s: %s\n", serverConfigPath, err.Error())
		return
	}

	fmt.Printf("INFO    Created %s\n", serverConfigPath)

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

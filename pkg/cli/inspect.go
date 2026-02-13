package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVarP(&inspectToolDefinitionsPath, "file", "f", "mcpfile.yaml", "the path to the MCP file")
	inspectCmd.Flags().StringVarP(&inspectServerConfigPath, "server-config", "s", "mcpserver.yaml", "the path to the server config file")
	inspectCmd.Flags().BoolVar(&inspectJSONOutput, "json", false, "output in JSON format")
}

var inspectToolDefinitionsPath string
var inspectServerConfigPath string
var inspectJSONOutput bool

var inspectCmd = &cobra.Command{
	Use:   "inspect [name]",
	Short: "Show detailed server information",
	Long: `Display detailed information about an MCP server including tools, prompts, resources, security configuration, and MCP client configuration JSON.

If a server name is provided as an argument, it will look up a running server by name.
Otherwise, use -f and -s flags to specify config files directly.`,
	Args: cobra.MaximumNArgs(1),
	Run:  executeInspectCmd,
}

// InspectOutput represents the complete inspection output
type InspectOutput struct {
	Server          ServerInfo              `json:"server"`
	Transport       TransportInfo           `json:"transport"`
	Security        SecurityInfo            `json:"security"`
	Tools           []ToolInfo              `json:"tools"`
	Prompts         []PromptInfo            `json:"prompts"`
	Resources       []ResourceInfo          `json:"resources"`
	ResourceTemplates []ResourceTemplateInfo `json:"resourceTemplates"`
	MCPClientConfig map[string]interface{}  `json:"mcpClientConfig"`
}

// ServerInfo contains basic server metadata
type ServerInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Instructions string `json:"instructions,omitempty"`
}

// TransportInfo contains transport configuration
type TransportInfo struct {
	Protocol  string      `json:"protocol"`
	Port      int         `json:"port,omitempty"`
	BasePath  string      `json:"basePath,omitempty"`
	Stateless bool        `json:"stateless,omitempty"`
	Health    *HealthInfo `json:"health,omitempty"`
}

// HealthInfo contains health check configuration
type HealthInfo struct {
	Enabled       bool   `json:"enabled"`
	LivenessPath  string `json:"livenessPath"`
	ReadinessPath string `json:"readinessPath"`
}

// SecurityInfo contains security configuration status
type SecurityInfo struct {
	TLS       *TLSInfo       `json:"tls,omitempty"`
	Auth      *AuthInfo      `json:"auth,omitempty"`
	ClientTLS *ClientTLSInfo `json:"clientTls,omitempty"`
}

// TLSInfo contains TLS status
type TLSInfo struct {
	Enabled bool `json:"enabled"`
}

// AuthInfo contains auth status
type AuthInfo struct {
	Enabled              bool     `json:"enabled"`
	JWKSURI              string   `json:"jwksUri,omitempty"`
	AuthorizationServers []string `json:"authorizationServers,omitempty"`
}

// ClientTLSInfo contains client TLS status
type ClientTLSInfo struct {
	Enabled            bool `json:"enabled"`
	InsecureSkipVerify bool `json:"insecureSkipVerify"`
}

// ToolInfo contains tool information for display
type ToolInfo struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	InvocationType string   `json:"invocationType,omitempty"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

// PromptInfo contains prompt information for display
type PromptInfo struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	ArgumentCount  int      `json:"argumentCount,omitempty"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

// ResourceInfo contains resource information for display
type ResourceInfo struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	URI            string   `json:"uri"`
	MIMEType       string   `json:"mimeType,omitempty"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

// ResourceTemplateInfo contains resource template information for display
type ResourceTemplateInfo struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	URITemplate    string   `json:"uriTemplate"`
	MIMEType       string   `json:"mimeType,omitempty"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

func executeInspectCmd(cmd *cobra.Command, args []string) {
	var toolDefinitionsPath, serverConfigPath string

	// If a name argument is provided, look up the server by name
	if len(args) > 0 {
		serverName := args[0]
		processManager := utils.GetProcessManager()
		processes, err := processManager.ListProcesses()
		if err != nil {
			fmt.Printf("failed to list running servers: %s\n", err.Error())
			os.Exit(1)
		}

		// Find server by name
		var found *utils.ProcessInfo
		for _, info := range processes {
			if info.Name == serverName {
				found = &info
				break
			}
		}

		if found == nil {
			fmt.Printf("no running server found with name: %s\n", serverName)
			fmt.Println("\nAvailable running servers:")
			for _, info := range processes {
				if utils.IsProcessAlive(info.PID) {
					fmt.Printf("  - %s (PID: %d)\n", info.Name, info.PID)
				}
			}
			os.Exit(1)
		}

		toolDefinitionsPath = found.MCPFilePath
		serverConfigPath = found.ServerConfigPath
	} else {
		// Use file flags
		var err error
		serverConfigPath, err = filepath.Abs(inspectServerConfigPath)
		if err != nil {
			fmt.Printf("failed to resolve server config file path: %s\n", err.Error())
			os.Exit(1)
		}

		if _, err := os.Stat(serverConfigPath); err != nil {
			fmt.Printf("no file found at server config path: %s\n", serverConfigPath)
			os.Exit(1)
		}

		// If -f was not explicitly set, look for mcpfile.yaml in the same directory as the server config
		mcpFilePath := inspectToolDefinitionsPath
		if !cmd.Flags().Changed("file") {
			serverConfigDir := filepath.Dir(serverConfigPath)
			mcpFilePath = filepath.Join(serverConfigDir, "mcpfile.yaml")
		}

		toolDefinitionsPath, err = filepath.Abs(mcpFilePath)
		if err != nil {
			fmt.Printf("failed to resolve MCP file path: %s\n", err.Error())
			os.Exit(1)
		}

		if _, err := os.Stat(toolDefinitionsPath); err != nil {
			fmt.Printf("no file found at MCP file path: %s\n", toolDefinitionsPath)
			os.Exit(1)
		}
	}

	// Parse MCP file
	toolDefs, err := definitions.ParseMCPFile(toolDefinitionsPath)
	if err != nil {
		fmt.Printf("invalid MCP file: %s\n", err)
		os.Exit(1)
	}

	// Parse server config file
	serverConfig, err := serverconfig.ParseMCPFile(serverConfigPath)
	if err != nil {
		fmt.Printf("invalid server config file: %s\n", err)
		os.Exit(1)
	}

	// Build inspection output
	output := buildInspectOutput(toolDefs, serverConfig, toolDefinitionsPath, serverConfigPath)

	if inspectJSONOutput {
		printJSONOutput(output)
	} else {
		printHumanReadableOutput(output)
	}
}

func buildInspectOutput(
	toolDefs *definitions.MCPToolDefinitionsFile,
	serverConfig *serverconfig.MCPServerConfigFile,
	toolDefsPath, serverConfigPath string,
) InspectOutput {
	output := InspectOutput{
		Server: ServerInfo{
			Name:         toolDefs.Name,
			Version:      toolDefs.Version,
			Instructions: toolDefs.Instructions,
		},
		Tools:             make([]ToolInfo, 0),
		Prompts:           make([]PromptInfo, 0),
		Resources:         make([]ResourceInfo, 0),
		ResourceTemplates: make([]ResourceTemplateInfo, 0),
	}

	// Build transport info
	output.Transport = buildTransportInfo(serverConfig)

	// Build security info
	output.Security = buildSecurityInfo(serverConfig)

	// Build tools
	for _, tool := range toolDefs.Tools {
		output.Tools = append(output.Tools, ToolInfo{
			Name:           tool.Name,
			Description:    tool.Description,
			InvocationType: tool.GetInvocationType(),
			RequiredScopes: tool.RequiredScopes,
		})
	}

	// Build prompts
	for _, prompt := range toolDefs.Prompts {
		argCount := 0
		if prompt.Arguments != nil {
			argCount = len(prompt.Arguments)
		}
		output.Prompts = append(output.Prompts, PromptInfo{
			Name:           prompt.Name,
			Description:    prompt.Description,
			ArgumentCount:  argCount,
			RequiredScopes: prompt.RequiredScopes,
		})
	}

	// Build resources
	for _, resource := range toolDefs.Resources {
		output.Resources = append(output.Resources, ResourceInfo{
			Name:           resource.Name,
			Description:    resource.Description,
			URI:            resource.URI,
			MIMEType:       resource.MIMEType,
			RequiredScopes: resource.RequiredScopes,
		})
	}

	// Build resource templates
	for _, rt := range toolDefs.ResourceTemplates {
		output.ResourceTemplates = append(output.ResourceTemplates, ResourceTemplateInfo{
			Name:           rt.Name,
			Description:    rt.Description,
			URITemplate:    rt.URITemplate,
			MIMEType:       rt.MIMEType,
			RequiredScopes: rt.RequiredScopes,
		})
	}

	// Build MCP client config
	output.MCPClientConfig = buildMCPClientConfig(toolDefs.Name, serverConfig, toolDefsPath, serverConfigPath)

	return output
}

func buildTransportInfo(serverConfig *serverconfig.MCPServerConfigFile) TransportInfo {
	info := TransportInfo{
		Protocol: serverconfig.TransportProtocolStreamableHttp,
	}

	if serverConfig.Runtime != nil {
		info.Protocol = serverConfig.Runtime.TransportProtocol

		if serverConfig.Runtime.StreamableHTTPConfig != nil {
			httpConfig := serverConfig.Runtime.StreamableHTTPConfig
			info.Port = httpConfig.Port
			info.BasePath = httpConfig.BasePath
			info.Stateless = httpConfig.IsStateless()

			if httpConfig.Health != nil {
				info.Health = &HealthInfo{
					Enabled:       httpConfig.Health.IsEnabled(),
					LivenessPath:  httpConfig.Health.LivenessPath,
					ReadinessPath: httpConfig.Health.ReadinessPath,
				}
			}
		}
	}

	return info
}

func buildSecurityInfo(serverConfig *serverconfig.MCPServerConfigFile) SecurityInfo {
	security := SecurityInfo{}

	if serverConfig.Runtime == nil {
		return security
	}

	// Check server TLS
	if serverConfig.Runtime.StreamableHTTPConfig != nil &&
		serverConfig.Runtime.StreamableHTTPConfig.TLS != nil {
		tls := serverConfig.Runtime.StreamableHTTPConfig.TLS
		if tls.CertFile != "" || tls.KeyFile != "" {
			security.TLS = &TLSInfo{Enabled: true}
		}
	}

	// Check auth
	if serverConfig.Runtime.StreamableHTTPConfig != nil &&
		serverConfig.Runtime.StreamableHTTPConfig.Auth != nil {
		auth := serverConfig.Runtime.StreamableHTTPConfig.Auth
		if auth.JWKSURI != "" || len(auth.AuthorizationServers) > 0 {
			security.Auth = &AuthInfo{
				Enabled:              true,
				JWKSURI:              auth.JWKSURI,
				AuthorizationServers: auth.AuthorizationServers,
			}
		}
	}

	// Check client TLS
	if serverConfig.Runtime.ClientTLSConfig != nil {
		clientTLS := serverConfig.Runtime.ClientTLSConfig
		if len(clientTLS.CACertFiles) > 0 || clientTLS.CACertDir != "" || clientTLS.InsecureSkipVerify {
			security.ClientTLS = &ClientTLSInfo{
				Enabled:            true,
				InsecureSkipVerify: clientTLS.InsecureSkipVerify,
			}
		}
	}

	return security
}

func buildMCPClientConfig(serverName string, serverConfig *serverconfig.MCPServerConfigFile, toolDefsPath, serverConfigPath string) map[string]interface{} {
	mcpServers := make(map[string]interface{})

	protocol := serverconfig.TransportProtocolStreamableHttp
	if serverConfig.Runtime != nil {
		protocol = serverConfig.Runtime.TransportProtocol
	}

	if protocol == serverconfig.TransportProtocolStdio {
		// Stdio transport config
		mcpServers[serverName] = map[string]interface{}{
			"command": "genmcp",
			"args":    []string{"run", "-f", toolDefsPath, "-s", serverConfigPath},
		}
	} else {
		// HTTP transport config
		port := serverconfig.DefaultPort
		basePath := serverconfig.DefaultBasePath
		scheme := "http"

		if serverConfig.Runtime != nil && serverConfig.Runtime.StreamableHTTPConfig != nil {
			httpConfig := serverConfig.Runtime.StreamableHTTPConfig
			if httpConfig.Port > 0 {
				port = httpConfig.Port
			}
			if httpConfig.BasePath != "" {
				basePath = httpConfig.BasePath
			}
			if httpConfig.TLS != nil && (httpConfig.TLS.CertFile != "" || httpConfig.TLS.KeyFile != "") {
				scheme = "https"
			}
		}

		url := fmt.Sprintf("%s://localhost:%d%s", scheme, port, basePath)
		mcpServers[serverName] = map[string]interface{}{
			"type": "http",
			"url":  url,
		}
	}

	return map[string]interface{}{
		"mcpServers": mcpServers,
	}
}

func printJSONOutput(output InspectOutput) {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal JSON: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func printHumanReadableOutput(output InspectOutput) {
	// Server info
	fmt.Printf("Server: %s (v%s)\n", output.Server.Name, output.Server.Version)
	fmt.Printf("Transport: %s\n", output.Transport.Protocol)

	if output.Transport.Protocol == serverconfig.TransportProtocolStreamableHttp {
		scheme := "http"
		if output.Security.TLS != nil && output.Security.TLS.Enabled {
			scheme = "https"
		}
		fmt.Printf("Endpoint: %s://localhost:%d%s\n", scheme, output.Transport.Port, output.Transport.BasePath)
	}

	if output.Server.Instructions != "" {
		fmt.Printf("Instructions: %s\n", truncateString(output.Server.Instructions, 80))
	}

	// Security section
	fmt.Println("\nSecurity:")
	if output.Security.TLS != nil && output.Security.TLS.Enabled {
		fmt.Println("  TLS: enabled (cert configured)")
	} else {
		fmt.Println("  TLS: disabled")
	}

	if output.Security.Auth != nil && output.Security.Auth.Enabled {
		fmt.Println("  Auth: enabled (OAuth 2.0)")
	} else {
		fmt.Println("  Auth: disabled")
	}

	if output.Security.ClientTLS != nil && output.Security.ClientTLS.Enabled {
		if output.Security.ClientTLS.InsecureSkipVerify {
			fmt.Println("  Client TLS: enabled (insecureSkipVerify: true)")
		} else {
			fmt.Println("  Client TLS: enabled (custom CA)")
		}
	}

	// Health endpoints
	if output.Transport.Health != nil && output.Transport.Health.Enabled {
		fmt.Println("\nHealth Endpoints:")
		fmt.Printf("  Liveness: %s\n", output.Transport.Health.LivenessPath)
		fmt.Printf("  Readiness: %s\n", output.Transport.Health.ReadinessPath)
	}

	// Capabilities
	fmt.Println("\nCapabilities:")

	// Tools
	fmt.Printf("  Tools (%d):\n", len(output.Tools))
	for _, tool := range output.Tools {
		fmt.Printf("    - %s: %s\n", tool.Name, truncateString(tool.Description, 60))
	}

	// Prompts
	if len(output.Prompts) > 0 {
		fmt.Printf("  Prompts (%d):\n", len(output.Prompts))
		for _, prompt := range output.Prompts {
			fmt.Printf("    - %s: %s\n", prompt.Name, truncateString(prompt.Description, 60))
		}
	}

	// Resources
	if len(output.Resources) > 0 {
		fmt.Printf("  Resources (%d):\n", len(output.Resources))
		for _, resource := range output.Resources {
			fmt.Printf("    - %s: %s (uri: %s)\n", resource.Name, truncateString(resource.Description, 40), resource.URI)
		}
	}

	// Resource Templates
	if len(output.ResourceTemplates) > 0 {
		fmt.Printf("  Resource Templates (%d):\n", len(output.ResourceTemplates))
		for _, rt := range output.ResourceTemplates {
			fmt.Printf("    - %s: %s (uriTemplate: %s)\n", rt.Name, truncateString(rt.Description, 40), rt.URITemplate)
		}
	}

	// MCP Client Config
	fmt.Println("\nMCP Client Configuration:")
	clientConfigJSON, _ := json.MarshalIndent(output.MCPClientConfig, "", "  ")
	// Indent each line for visual clarity
	lines := strings.Split(string(clientConfigJSON), "\n")
	for _, line := range lines {
		fmt.Printf("  %s\n", line)
	}
}

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

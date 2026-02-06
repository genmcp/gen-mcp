package cli

import (
	"encoding/json"
	"testing"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestBuildTransportInfo(t *testing.T) {
	tests := map[string]struct {
		serverConfig *serverconfig.MCPServerConfigFile
		expected     TransportInfo
	}{
		"nil runtime defaults to streamablehttp": {
			serverConfig: &serverconfig.MCPServerConfigFile{},
			expected: TransportInfo{
				Protocol: serverconfig.TransportProtocolStreamableHttp,
			},
		},
		"streamablehttp with config": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:      9090,
							BasePath:  "/api/mcp",
							Stateless: ptr.To(false),
							Health: &serverconfig.HealthConfig{
								Enabled:       ptr.To(true),
								LivenessPath:  "/live",
								ReadinessPath: "/ready",
							},
						},
					},
				},
			},
			expected: TransportInfo{
				Protocol:  serverconfig.TransportProtocolStreamableHttp,
				Port:      9090,
				BasePath:  "/api/mcp",
				Stateless: false,
				Health: &HealthInfo{
					Enabled:       true,
					LivenessPath:  "/live",
					ReadinessPath: "/ready",
				},
			},
		},
		"stdio transport": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStdio,
					},
				},
			},
			expected: TransportInfo{
				Protocol: serverconfig.TransportProtocolStdio,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildTransportInfo(tc.serverConfig)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildSecurityInfo(t *testing.T) {
	tests := map[string]struct {
		serverConfig *serverconfig.MCPServerConfigFile
		expected     SecurityInfo
	}{
		"nil runtime returns empty security": {
			serverConfig: &serverconfig.MCPServerConfigFile{},
			expected:     SecurityInfo{},
		},
		"tls enabled": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							TLS: &serverconfig.TLSConfig{
								CertFile: "/path/to/cert.pem",
								KeyFile:  "/path/to/key.pem",
							},
						},
					},
				},
			},
			expected: SecurityInfo{
				TLS: &TLSInfo{Enabled: true},
			},
		},
		"auth enabled with jwks": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Auth: &serverconfig.AuthConfig{
								JWKSURI:              "https://auth.example.com/.well-known/jwks.json",
								AuthorizationServers: []string{"https://auth.example.com"},
							},
						},
					},
				},
			},
			expected: SecurityInfo{
				Auth: &AuthInfo{
					Enabled:              true,
					JWKSURI:              "https://auth.example.com/.well-known/jwks.json",
					AuthorizationServers: []string{"https://auth.example.com"},
				},
			},
		},
		"client tls with custom ca": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						ClientTLSConfig: &serverconfig.ClientTLSConfig{
							CACertFiles: []string{"/path/to/ca.pem"},
						},
					},
				},
			},
			expected: SecurityInfo{
				ClientTLS: &ClientTLSInfo{
					Enabled:            true,
					InsecureSkipVerify: false,
				},
			},
		},
		"client tls with insecure skip verify": {
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						ClientTLSConfig: &serverconfig.ClientTLSConfig{
							InsecureSkipVerify: true,
						},
					},
				},
			},
			expected: SecurityInfo{
				ClientTLS: &ClientTLSInfo{
					Enabled:            true,
					InsecureSkipVerify: true,
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildSecurityInfo(tc.serverConfig)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildMCPClientConfig(t *testing.T) {
	tests := map[string]struct {
		serverName       string
		serverConfig     *serverconfig.MCPServerConfigFile
		toolDefsPath     string
		serverConfigPath string
		expectedType     string // "http" or "stdio"
		expectedURL      string // for http
	}{
		"http transport with defaults": {
			serverName: "test-server",
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:     8080,
							BasePath: "/mcp",
						},
					},
				},
			},
			toolDefsPath:     "/path/to/mcpfile.yaml",
			serverConfigPath: "/path/to/mcpserver.yaml",
			expectedType:     "http",
			expectedURL:      "http://localhost:8080/mcp",
		},
		"http transport with tls": {
			serverName: "secure-server",
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:     443,
							BasePath: "/api",
							TLS: &serverconfig.TLSConfig{
								CertFile: "/cert.pem",
							},
						},
					},
				},
			},
			toolDefsPath:     "/path/to/mcpfile.yaml",
			serverConfigPath: "/path/to/mcpserver.yaml",
			expectedType:     "http",
			expectedURL:      "https://localhost:443/api",
		},
		"stdio transport": {
			serverName: "cli-server",
			serverConfig: &serverconfig.MCPServerConfigFile{
				MCPServerConfig: serverconfig.MCPServerConfig{
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: serverconfig.TransportProtocolStdio,
					},
				},
			},
			toolDefsPath:     "/path/to/mcpfile.yaml",
			serverConfigPath: "/path/to/mcpserver.yaml",
			expectedType:     "stdio",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildMCPClientConfig(tc.serverName, tc.serverConfig, tc.toolDefsPath, tc.serverConfigPath)

			// Verify structure
			mcpServers, ok := result["mcpServers"].(map[string]interface{})
			require.True(t, ok, "mcpServers should be a map")

			serverConfig, ok := mcpServers[tc.serverName].(map[string]interface{})
			require.True(t, ok, "server config should exist")

			if tc.expectedType == "http" {
				assert.Equal(t, "http", serverConfig["type"])
				assert.Equal(t, tc.expectedURL, serverConfig["url"])
			} else {
				assert.Equal(t, "genmcp", serverConfig["command"])
				args, ok := serverConfig["args"].([]string)
				require.True(t, ok)
				assert.Contains(t, args, tc.toolDefsPath)
				assert.Contains(t, args, tc.serverConfigPath)
			}
		})
	}
}

func TestBuildInspectOutput(t *testing.T) {
	toolDefs := &definitions.MCPToolDefinitionsFile{
		Kind:          definitions.KindMCPToolDefinitions,
		SchemaVersion: "0.2.0",
		MCPToolDefinitions: definitions.MCPToolDefinitions{
			Name:         "test-api",
			Version:      "1.0.0",
			Instructions: "Test instructions",
			Tools: []*definitions.Tool{
				{Name: "tool1", Description: "First tool"},
				{Name: "tool2", Description: "Second tool"},
			},
			Prompts: []*definitions.Prompt{
				{Name: "prompt1", Description: "First prompt"},
			},
			Resources: []*definitions.Resource{
				{Name: "resource1", Description: "First resource", URI: "file://test.json"},
			},
			ResourceTemplates: []*definitions.ResourceTemplate{
				{Name: "template1", Description: "First template", URITemplate: "resource://{id}"},
			},
		},
	}

	serverConfig := &serverconfig.MCPServerConfigFile{
		MCPServerConfig: serverconfig.MCPServerConfig{
			Runtime: &serverconfig.ServerRuntime{
				TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
					Port:     8080,
					BasePath: "/mcp",
				},
			},
		},
	}

	result := buildInspectOutput(toolDefs, serverConfig, "/path/to/mcpfile.yaml", "/path/to/mcpserver.yaml")

	// Verify server info
	assert.Equal(t, "test-api", result.Server.Name)
	assert.Equal(t, "1.0.0", result.Server.Version)
	assert.Equal(t, "Test instructions", result.Server.Instructions)

	// Verify transport
	assert.Equal(t, serverconfig.TransportProtocolStreamableHttp, result.Transport.Protocol)
	assert.Equal(t, 8080, result.Transport.Port)
	assert.Equal(t, "/mcp", result.Transport.BasePath)

	// Verify counts
	assert.Len(t, result.Tools, 2)
	assert.Len(t, result.Prompts, 1)
	assert.Len(t, result.Resources, 1)
	assert.Len(t, result.ResourceTemplates, 1)

	// Verify tool details
	assert.Equal(t, "tool1", result.Tools[0].Name)
	assert.Equal(t, "First tool", result.Tools[0].Description)

	// Verify MCP client config exists
	assert.NotNil(t, result.MCPClientConfig)
	assert.Contains(t, result.MCPClientConfig, "mcpServers")
}

func TestTruncateString(t *testing.T) {
	tests := map[string]struct {
		input    string
		maxLen   int
		expected string
	}{
		"short string unchanged": {
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		"exact length unchanged": {
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		"long string truncated": {
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is...",
		},
		"newlines replaced": {
			input:    "line1\nline2\nline3",
			maxLen:   50,
			expected: "line1 line2 line3",
		},
		"newlines replaced and truncated": {
			input:    "line1\nline2\nline3",
			maxLen:   10,
			expected: "line1 l...",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := truncateString(tc.input, tc.maxLen)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInspectOutputJSONSerialization(t *testing.T) {
	output := InspectOutput{
		Server: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
		Transport: TransportInfo{
			Protocol: "streamablehttp",
			Port:     8080,
			BasePath: "/mcp",
		},
		Security: SecurityInfo{
			TLS:  &TLSInfo{Enabled: true},
			Auth: &AuthInfo{Enabled: true, JWKSURI: "https://example.com/jwks"},
		},
		Tools: []ToolInfo{
			{Name: "tool1", Description: "Test tool"},
		},
		Prompts:           []PromptInfo{},
		Resources:         []ResourceInfo{},
		ResourceTemplates: []ResourceTemplateInfo{},
		MCPClientConfig: map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"test-server": map[string]interface{}{
					"type": "http",
					"url":  "http://localhost:8080/mcp",
				},
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(output)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled InspectOutput
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, output.Server.Name, unmarshaled.Server.Name)
	assert.Equal(t, output.Server.Version, unmarshaled.Server.Version)
	assert.Equal(t, output.Transport.Protocol, unmarshaled.Transport.Protocol)
	assert.Equal(t, output.Security.TLS.Enabled, unmarshaled.Security.TLS.Enabled)
	assert.Len(t, unmarshaled.Tools, 1)
}

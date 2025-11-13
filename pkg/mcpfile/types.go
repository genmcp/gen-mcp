package mcpfile

import (
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

const (
	MCPFileVersion                  = "0.1.0"
	TransportProtocolStreamableHttp = "streamablehttp"
	TransportProtocolStdio          = "stdio"
)

// MCPServer defines the metadata and capabilities of an MCP server.
// It embeds MCPToolDefinitions for tool definitions and MCPServerConfig for server configuration.
type MCPServer struct {
	// Tool definitions (tools, prompts, resources, resource templates)
	definitions.MCPToolDefinitions

	// Server configuration (runtime, etc.)
	serverconfig.MCPServerConfig
}

// Name returns the server name, preferring the value from server config, falling back to tool definitions
func (m *MCPServer) Name() string {
	if m.MCPServerConfig.Name != "" {
		return m.MCPServerConfig.Name
	}
	return m.MCPToolDefinitions.Name
}

// Version returns the server version, preferring the value from server config, falling back to tool definitions
func (m *MCPServer) Version() string {
	if m.MCPServerConfig.Version != "" {
		return m.MCPServerConfig.Version
	}
	return m.MCPToolDefinitions.Version
}

// Instructions returns the instructions, preferring the value from server config, falling back to tool definitions
func (m *MCPServer) Instructions() string {
	if m.MCPServerConfig.Instructions != "" {
		return m.MCPServerConfig.Instructions
	}
	return m.MCPToolDefinitions.Instructions
}

// InvocationBases returns merged invocation bases (server config takes precedence for conflicts)
func (m *MCPServer) InvocationBases() map[string]*invocation.InvocationConfigWrapper {
	result := make(map[string]*invocation.InvocationConfigWrapper)
	// First add from tool definitions
	if m.MCPToolDefinitions.InvocationBases != nil {
		for k, v := range m.MCPToolDefinitions.InvocationBases {
			result[k] = v
		}
	}
	// Then override/add from server config (server config takes precedence)
	if m.MCPServerConfig.InvocationBases != nil {
		for k, v := range m.MCPServerConfig.InvocationBases {
			result[k] = v
		}
	}
	return result
}

// MCPFile is the root structure of an MCP configuration file.
type MCPFile struct {
	// Version of the MCP file format.
	FileVersion string `json:"mcpFileVersion" jsonschema:"required"`

	// MCP server definition.
	MCPServer `json:",inline"`
}

var _ invocation.Primitive = (*definitions.Tool)(nil)
var _ invocation.Primitive = (*definitions.Prompt)(nil)
var _ invocation.Primitive = (*definitions.Resource)(nil)
var _ invocation.Primitive = (*definitions.ResourceTemplate)(nil)

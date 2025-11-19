package mcpserver

import (
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// MCPServer defines the metadata and capabilities of an MCP server.
// It embeds MCPToolDefinitions for tool definitions and MCPServerConfig for server configuration.
type MCPServer struct {
	// Tool definitions (tools, prompts, resources, resource templates)
	definitions.MCPToolDefinitions

	// Server configuration (runtime, etc.)
	//nolint:govet
	serverconfig.MCPServerConfig
}

// Name returns the server name from tool definitions
func (m *MCPServer) Name() string {
	return m.MCPToolDefinitions.Name
}

// Version returns the server version from tool definitions
func (m *MCPServer) Version() string {
	return m.MCPToolDefinitions.Version
}

// Instructions returns the instructions from tool definitions
func (m *MCPServer) Instructions() string {
	return m.MCPToolDefinitions.Instructions
}

// InvocationBases returns invocation bases from tool definitions
func (m *MCPServer) InvocationBases() map[string]*invocation.InvocationConfigWrapper {
	if m.MCPToolDefinitions.InvocationBases == nil {
		return make(map[string]*invocation.InvocationConfigWrapper)
	}
	return m.MCPToolDefinitions.InvocationBases
}

package mcpserver

import (
	"testing"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/stretchr/testify/assert"
)

func TestMCPFileValidate(t *testing.T) {
	mockValidator := func(primitive invocation.Primitive) error {
		return nil
	}

	t.Run("missing name should fail validation", func(t *testing.T) {
		mcpServer := &MCPServer{
			MCPToolDefinitions: definitions.MCPToolDefinitions{
				Version: "1.0.0",
			},
			MCPServerConfig: serverconfig.MCPServerConfig{
				Version: "1.0.0",
				Runtime: &serverconfig.ServerRuntime{
					TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
						Port:     3000,
						BasePath: serverconfig.DefaultBasePath,
					},
				},
			},
		}
		err := mcpServer.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing version should fail validation", func(t *testing.T) {
		mcpServer := &MCPServer{
			MCPToolDefinitions: definitions.MCPToolDefinitions{
				Name: "test-server",
			},
			MCPServerConfig: serverconfig.MCPServerConfig{
				Name: "test-server",
				Runtime: &serverconfig.ServerRuntime{
					TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
						Port:     3000,
						BasePath: serverconfig.DefaultBasePath,
					},
				},
			},
		}
		err := mcpServer.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("valid server should pass validation", func(t *testing.T) {
		mcpServer := &MCPServer{
			MCPToolDefinitions: definitions.MCPToolDefinitions{
				Name:    "test-server",
				Version: "1.0.0",
			},
			MCPServerConfig: serverconfig.MCPServerConfig{
				Name:    "test-server",
				Version: "1.0.0",
				Runtime: &serverconfig.ServerRuntime{
					TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
						Port:     3000,
						BasePath: serverconfig.DefaultBasePath,
					},
				},
			},
		}
		err := mcpServer.Validate(mockValidator)
		assert.NoError(t, err)
	})
}

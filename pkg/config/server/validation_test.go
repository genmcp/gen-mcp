package mcpfile

import (
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/stretchr/testify/assert"
)

func TestMCPFileValidate(t *testing.T) {
	mockValidator := func(primitive invocation.Primitive) error {
		return nil
	}

	t.Run("missing name should fail validation", func(t *testing.T) {
		mcpFile := &MCPServerConfigFile{
			FileVersion: MCPFileVersion,
			MCPServerConfig: MCPServerConfig{
				Version: "1.0.0",
				Runtime: &ServerRuntime{
					TransportProtocol: TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &StreamableHTTPConfig{
						Port:     3000,
						BasePath: DefaultBasePath,
					},
				},
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing version should fail validation", func(t *testing.T) {
		mcpFile := &MCPServerConfigFile{
			FileVersion: MCPFileVersion,
			MCPServerConfig: MCPServerConfig{
				Name: "test-server",
				Runtime: &ServerRuntime{
					TransportProtocol: TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &StreamableHTTPConfig{
						Port:     3000,
						BasePath: DefaultBasePath,
					},
				},
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("valid server should pass validation", func(t *testing.T) {
		mcpFile := &MCPServerConfigFile{
			FileVersion: MCPFileVersion,
			MCPServerConfig: MCPServerConfig{
				Name:    "test-server",
				Version: "1.0.0",
				Runtime: &ServerRuntime{
					TransportProtocol: TransportProtocolStreamableHttp,
					StreamableHTTPConfig: &StreamableHTTPConfig{
						Port:     3000,
						BasePath: DefaultBasePath,
					},
				},
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.NoError(t, err)
	})
}

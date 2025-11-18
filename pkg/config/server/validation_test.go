package server

import (
	"testing"

	"github.com/genmcp/gen-mcp/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestServerFileValidate(t *testing.T) {
	t.Run("missing name should fail validation", func(t *testing.T) {
		serverConfig := &MCPServerConfigFile{
			SchemaVersion: config.SchemaVersion,
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
		err := serverConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing version should fail validation", func(t *testing.T) {
		serverConfig := &MCPServerConfigFile{
			SchemaVersion: config.SchemaVersion,
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
		err := serverConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("valid server should pass validation", func(t *testing.T) {
		serverConfig := &MCPServerConfigFile{
			SchemaVersion: config.SchemaVersion,
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
		err := serverConfig.Validate()
		assert.NoError(t, err)
	})
}

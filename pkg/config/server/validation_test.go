package server

import (
	"testing"

	"github.com/genmcp/gen-mcp/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestServerFileValidate(t *testing.T) {
	t.Run("valid server should pass validation", func(t *testing.T) {
		serverConfig := &MCPServerConfigFile{
			SchemaVersion: config.SchemaVersion,
			MCPServerConfig: MCPServerConfig{
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

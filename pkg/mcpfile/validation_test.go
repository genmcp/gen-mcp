package mcpfile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPFileValidate(t *testing.T) {
	mockValidator := func(invocationType string, data json.RawMessage, tool *Tool) error {
		return nil
	}

	t.Run("missing server should fail validation", func(t *testing.T) {
		mcpFile := &MCPFile{
			FileVersion: MCPFileVersion,
			Server:      nil,
		}
		err := mcpFile.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server is required")
	})

	t.Run("valid server should pass validation", func(t *testing.T) {
		mcpFile := &MCPFile{
			FileVersion: MCPFileVersion,
			Server: &MCPServer{
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

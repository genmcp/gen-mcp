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
		mcpFile := &MCPToolDefinitionsFile{
			FileVersion: MCPFileVersion,
			MCPToolDefinitions: MCPToolDefinitions{
				Version: "1.0.0",
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing version should fail validation", func(t *testing.T) {
		mcpFile := &MCPToolDefinitionsFile{
			FileVersion: MCPFileVersion,
			MCPToolDefinitions: MCPToolDefinitions{
				Name: "test-server",
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("valid server should pass validation", func(t *testing.T) {
		mcpFile := &MCPToolDefinitionsFile{
			FileVersion: MCPFileVersion,
			MCPToolDefinitions: MCPToolDefinitions{
				Name:    "test-server",
				Version: "1.0.0",
			},
		}
		err := mcpFile.Validate(mockValidator)
		assert.NoError(t, err)
	})
}

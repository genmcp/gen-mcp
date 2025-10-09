package mcpfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSeparateConfigIntegration demonstrates the real-world usage of separate config files
func TestSeparateConfigIntegration(t *testing.T) {
	// Parse the server config file
	serverConfig, err := ParseMCPServerConfig("./testdata/mcpserver-basic.yaml")
	assert.NoError(t, err, "should successfully parse server config")
	assert.Equal(t, "test-server", serverConfig.Name)
	assert.Equal(t, "1.0.0", serverConfig.Version)
	assert.NotNil(t, serverConfig.Runtime)
	assert.Equal(t, TransportProtocolStreamableHttp, serverConfig.Runtime.TransportProtocol)
	assert.Equal(t, 8080, serverConfig.Runtime.StreamableHTTPConfig.Port)

	// Parse the mcpfile without runtime
	mcpFile, err := ParseMCPFile("./testdata/mcpfile-without-runtime.yaml")
	assert.NoError(t, err, "should successfully parse mcpfile")
	assert.Nil(t, mcpFile.Runtime, "mcpfile should not have runtime when using separate config")
	assert.Equal(t, 1, len(mcpFile.Tools), "should have tools defined")

	// Verify tool is correctly parsed
	tool := mcpFile.Tools[0]
	assert.Equal(t, "get_user_by_company", tool.Name)
	assert.Equal(t, "Users Provider", tool.Title)
	assert.Equal(t, "Get list of users from a given company", tool.Description)

	// Simulate what RunServersWithConfig does: combine them
	combinedServer := &MCPServer{
		Name:              serverConfig.Name,
		Version:           serverConfig.Version,
		Runtime:           serverConfig.Runtime,
		Tools:             mcpFile.Tools,
		Prompts:           mcpFile.Prompts,
		Resources:         mcpFile.Resources,
		ResourceTemplates: mcpFile.ResourceTemplates,
	}

	// Verify the combined server has everything needed
	assert.Equal(t, "test-server", combinedServer.Name)
	assert.Equal(t, "1.0.0", combinedServer.Version)
	assert.NotNil(t, combinedServer.Runtime)
	assert.Equal(t, 1, len(combinedServer.Tools))
	assert.Equal(t, "get_user_by_company", combinedServer.Tools[0].Name)
}

// TestBackwardCompatibilitySingleFile ensures old mcpfile.yaml format still works
func TestBackwardCompatibilitySingleFile(t *testing.T) {
	// This should work exactly as before
	mcpFile, err := ParseMCPFile("./testdata/one-server-tools.yaml")
	assert.NoError(t, err, "should successfully parse traditional mcpfile")

	// Verify it has both runtime and tools in one file
	assert.NotNil(t, mcpFile.Runtime, "traditional mcpfile should have runtime")
	assert.Equal(t, "test-server", mcpFile.Name)
	assert.Equal(t, "1.0.0", mcpFile.Version)
	assert.Equal(t, 1, len(mcpFile.Tools))

	// Verify default runtime settings are applied
	assert.Equal(t, TransportProtocolStreamableHttp, mcpFile.Runtime.TransportProtocol)
	assert.Equal(t, 3000, mcpFile.Runtime.StreamableHTTPConfig.Port)
	assert.Equal(t, DefaultBasePath, mcpFile.Runtime.StreamableHTTPConfig.BasePath)
	assert.True(t, mcpFile.Runtime.StreamableHTTPConfig.Stateless)
}

// TestMCPFileOnlyTools verifies mcpfile can be tools-only without name/version
func TestMCPFileOnlyTools(t *testing.T) {
	mcpFile, err := ParseMCPFile("./testdata/mcpfile-without-runtime.yaml")
	assert.NoError(t, err)

	// Name and version are empty when not provided
	assert.Empty(t, mcpFile.Name)
	assert.Empty(t, mcpFile.Version)

	// But tools are present
	assert.Equal(t, 1, len(mcpFile.Tools))
	assert.Equal(t, "get_user_by_company", mcpFile.Tools[0].Name)
}

// TestServerConfigDefaults verifies default values are set correctly
func TestServerConfigDefaults(t *testing.T) {
	serverConfig, err := ParseMCPServerConfig("./testdata/mcpserver-basic.yaml")
	assert.NoError(t, err)

	// Verify defaults are applied
	assert.NotNil(t, serverConfig.Runtime.StreamableHTTPConfig)
	assert.True(t, serverConfig.Runtime.StreamableHTTPConfig.Stateless, "should default to stateless")
	assert.Equal(t, "/mcp", serverConfig.Runtime.StreamableHTTPConfig.BasePath, "should have default base path")
}

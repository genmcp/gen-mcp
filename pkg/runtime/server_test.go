package runtime

import (
	"os"
	"path/filepath"
	"testing"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestRunServerLogMessages(t *testing.T) {
	// Create temporary files for testing
	tmpDir := t.TempDir()
	toolDefsPath := filepath.Join(tmpDir, "test-tools.yaml")
	serverConfigPath := filepath.Join(tmpDir, "test-server.yaml")

	// Create a minimal tool definitions file
	toolDefsContent := `kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: test-server
version: "1.0.0"
tools:
- name: test_tool
  description: "A test tool"
  inputSchema:
    type: object
    properties: {}
  invocation:
    http:
      method: GET
      url: http://localhost:8080/test
`
	err := os.WriteFile(toolDefsPath, []byte(toolDefsContent), 0644)
	require.NoError(t, err)

	// Create a minimal server config file
	serverConfigContent := `kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 0  # Use 0 for random port in tests
`
	err = os.WriteFile(serverConfigPath, []byte(serverConfigContent), 0644)
	require.NoError(t, err)

	// Parse the configs to create an MCPServer
	toolDefsFile, err := definitions.ParseMCPFile(toolDefsPath)
	require.NoError(t, err)

	serverConfigFile, err := serverconfig.ParseMCPFile(serverConfigPath)
	require.NoError(t, err)

	mcpServer := &mcpserver.MCPServer{
		MCPToolDefinitions: toolDefsFile.MCPToolDefinitions,
		MCPServerConfig:    serverConfigFile.MCPServerConfig,
	}

	// Override the logger with our observer
	// We need to access the runtime and set a custom logger
	// Since GetBaseLogger uses sync.Once, we'll need to work around that
	// For this test, we'll verify the logger is created correctly
	// and test the log message format separately

	// Verify the server has tools
	assert.Equal(t, 1, len(mcpServer.Tools))

	// Verify logger is created (not nil)
	baseLogger := mcpServer.Runtime.GetBaseLogger()
	assert.NotNil(t, baseLogger)
	// Should be enabled for info level (console logger by default)
	assert.True(t, baseLogger.Core().Enabled(zapcore.InfoLevel))

	// Test that we can log messages
	baseLogger.Info("test message")
	// The logger should work (we can't easily verify console output in unit tests,
	// but we can verify the logger is functional)
}

func TestRunServerWithCustomLogger(t *testing.T) {
	// Test that custom logging config is respected
	tmpDir := t.TempDir()
	toolDefsPath := filepath.Join(tmpDir, "test-tools.yaml")
	serverConfigPath := filepath.Join(tmpDir, "test-server.yaml")

	// Create a minimal tool definitions file
	toolDefsContent := `kind: MCPToolDefinitions
schemaVersion: "0.2.0"
name: test-server
version: "1.0.0"
tools:
- name: test_tool
  description: "A test tool"
  inputSchema:
    type: object
    properties: {}
  invocation:
    http:
      method: GET
      url: http://localhost:8080/test
`
	err := os.WriteFile(toolDefsPath, []byte(toolDefsContent), 0644)
	require.NoError(t, err)

	// Create server config with custom logging
	serverConfigContent := `kind: MCPServerConfig
schemaVersion: "0.2.0"
runtime:
  transportProtocol: streamablehttp
  streamableHttpConfig:
    port: 0
  loggingConfig:
    level: warn
    encoding: json
`
	err = os.WriteFile(serverConfigPath, []byte(serverConfigContent), 0644)
	require.NoError(t, err)

	// Parse configs
	toolDefsFile, err := definitions.ParseMCPFile(toolDefsPath)
	require.NoError(t, err)

	serverConfigFile, err := serverconfig.ParseMCPFile(serverConfigPath)
	require.NoError(t, err)

	mcpServer := &mcpserver.MCPServer{
		MCPToolDefinitions: toolDefsFile.MCPToolDefinitions,
		MCPServerConfig:    serverConfigFile.MCPServerConfig,
	}

	// Verify custom logger is used
	logger := mcpServer.Runtime.GetBaseLogger()
	assert.NotNil(t, logger)
	// With warn level, info should be disabled
	assert.False(t, logger.Core().Enabled(zapcore.InfoLevel))
	assert.True(t, logger.Core().Enabled(zapcore.WarnLevel))
}

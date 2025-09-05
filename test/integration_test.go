package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	// 1. Create a mock HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/123", r.URL.Path)
		if _, err := fmt.Fprintln(w, `{"status": "ok"}`); err != nil {
			t.Fatalf("failed to write response in server: %v", err)
		}
	}))
	defer httpServer.Close()

	// 2. Define the MCP file in YAML
	mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
servers:
  - name: test-server
    version: "1.0"
    tools:
      - name: get_user
        description: "Get user by ID"
        inputSchema:
          type: object
          properties:
            userId:
              type: string
          required:
            - userId
        outputSchema:
          type: object
          properties:
            status:
              type: string
        invocation:
          http:
            url: "%s/users/{userId}"
            method: "GET"
`, httpServer.URL)

	// 3. Write the MCP file to a temporary file
	tmpfile, err := os.CreateTemp("", "mcp-*.yaml")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Errorf("failed to remove temporary file %s: %v", tmpfile.Name(), err)
		}
	}()

	_, err = tmpfile.WriteString(mcpYAML)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	// 4. Parse the MCP file
	mcpConfig, err := mcpfile.ParseMCPFile(tmpfile.Name())
	require.NoError(t, err)

	// 5. Create and start the MCP server
	mcpServer := mcpserver.MakeServer(mcpConfig.Servers[0])

	httpMCPServer := server.NewStreamableHTTPServer(mcpServer)
	go func() {
		if err := httpMCPServer.Start(":8008"); err != nil && err != http.ErrServerClosed {
			require.Failf(t, "unexpected mcp server error: %s", err.Error())
		}
		fmt.Printf("shutdown the mcp server\n")
	}()

	defer func() {
		fmt.Printf("shutting down the mcp server\n")
		err = httpMCPServer.Shutdown(context.Background())
		require.NoError(t, err)
	}()

	// 6. Create an MCP client
	mcpServerURL := "http://localhost:8008/mcp"
	streamableHttpTransport, err := transport.NewStreamableHTTP(mcpServerURL)
	require.NoError(t, err)
	client := mcpclient.NewClient(
		streamableHttpTransport,
	)
	require.NoError(t, err)

	defer func() {
		fmt.Printf("sending client close request\n")
		err := client.Close()
		fmt.Printf("closed client\n")
		require.NoError(t, err, "closing the client should not fail")
	}()

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test client",
				Version: "0.0.1",
			},
		},
	}
	_, err = client.Initialize(context.Background(), initRequest)
	require.NoError(t, err, "client should connect to the server")

	// 7. Call the tool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolCall := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_user",
			Arguments: map[string]any{
				"userId": "123",
			},
		},
	}

	res, err := client.CallTool(ctx, toolCall)
	require.NoError(t, err)

	// 8. Assert the results
	require.NotNil(t, res)
	require.IsType(t, res.Content[0], mcp.TextContent{})

	textResult, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)

	assert.JSONEq(t, `{"status": "ok"}`, textResult.Text)
}

func TestIntegrationCLI(t *testing.T) {
	// 1. Define the MCP file in YAML
	mcpYAML := `
mcpFileVersion: 0.0.1
servers:
  - name: test-server
    version: "1.0"
    tools:
      - name: list_files
        description: "List files in a directory"
        inputSchema:
          type: object
          properties:
            path:
              type: string
          required:
            - path
        outputSchema:
          type: object
          properties:
            stdout:
              type: string
        invocation:
          cli:
            command: "ls -a {path}"
`

	// 2. Write the MCP file to a temporary file
	tmpfile, err := os.CreateTemp("", "mcp-*.yaml")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Errorf("failed to remove temporary file %s: %v", tmpfile.Name(), err)
		}
	}()

	_, err = tmpfile.WriteString(mcpYAML)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	// 3. Parse the MCP file
	mcpConfig, err := mcpfile.ParseMCPFile(tmpfile.Name())
	require.NoError(t, err)

	// 4. Create and start the MCP server
	mcpServer := mcpserver.MakeServer(mcpConfig.Servers[0])

	httpMCPServer := server.NewStreamableHTTPServer(mcpServer)
	go func() {
		if err := httpMCPServer.Start(":8009"); err != nil && err != http.ErrServerClosed {
			require.Failf(t, "unexpected mcp server error: %s", err.Error())
		}
		fmt.Printf("shutdown the mcp server\n")
	}()

	defer func() {
		fmt.Printf("shutting down the mcp server\n")
		err = httpMCPServer.Shutdown(context.Background())
		require.NoError(t, err)
	}()

	// 5. Create an MCP client
	mcpServerURL := "http://localhost:8009/mcp"
	streamableHttpTransport, err := transport.NewStreamableHTTP(mcpServerURL)
	require.NoError(t, err)
	client := mcpclient.NewClient(
		streamableHttpTransport,
	)
	require.NoError(t, err)

	defer func() {
		fmt.Printf("sending client close request\n")
		err := client.Close()
		fmt.Printf("closed client\n")
		require.NoError(t, err, "closing the client should not fail")
	}()

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test client",
				Version: "0.0.1",
			},
		},
	}
	_, err = client.Initialize(context.Background(), initRequest)
	require.NoError(t, err, "client should connect to the server")

	// 6. Call the tool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolCall := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_files",
			Arguments: map[string]any{
				"path": ".",
			},
		},
	}

	res, err := client.CallTool(ctx, toolCall)
	require.NoError(t, err)

	// 7. Assert the results
	require.NotNil(t, res)
	require.IsType(t, res.Content[0], mcp.TextContent{})

	textResult, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)

	assert.Contains(t, textResult.Text, "integration_test.go")
}

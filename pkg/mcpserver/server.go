package mcpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/oauth"
)

func MakeServer(mcpServer *mcpfile.MCPServer) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer(
		mcpServer.Name,
		mcpServer.Version,
		server.WithToolCapabilities(true),
		server.WithToolFilter(filterAuthorizedTools(mcpServer)),
	)

	for _, t := range mcpServer.Tools {
		s.AddTool(
			mcp.NewTool(
				t.Name,
				t.GetMCPToolOpts()...,
			),
			createAuthorizedToolHandler(t),
		)
	}

	return s
}

func RunServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	s := MakeServer(mcpServerConfig)

	switch strings.ToLower(mcpServerConfig.Runtime.TransportProtocol) {
	case mcpfile.TransportProtocolStreamableHttp:
		// Create a root mux to handle different endpoints
		mux := http.NewServeMux()

		// Set up MCP server under /mcp (or whatever is under BasePath)
		mcpServer := server.NewStreamableHTTPServer(s)
		mux.Handle(mcpServerConfig.Runtime.StreamableHTTPConfig.BasePath, oauth.Middleware(mcpServerConfig)(mcpServer))

		// Set up OAuth protected resource metadata endpoint under / if needed
		if mcpServerConfig.Runtime.StreamableHTTPConfig.Auth != nil {
			mux.HandleFunc(oauth.ProtectedResourceMetadataEndpoint, oauth.ProtectedResourceMetadataHandler(mcpServerConfig))
		}

		// Use custom server with the mux
		srv := &http.Server{
			Addr:    fmt.Sprintf(":%d", mcpServerConfig.Runtime.StreamableHTTPConfig.Port),
			Handler: mux,
		}

		fmt.Printf("starting listen on :%d\n", mcpServerConfig.Runtime.StreamableHTTPConfig.Port)

		// Channel to capture server errors
		errCh := make(chan error, 1)
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()

		// Wait for context cancellation or server error
		select {
		case <-ctx.Done():
			fmt.Println("shutting down server...")
			return srv.Shutdown(context.Background())
		case err := <-errCh:
			return err
		}
	case mcpfile.TransportProtocolStdio:
		stdioServer := server.NewStdioServer(s)
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	default:
		return fmt.Errorf("tried running invalid transport protocol")
	}
}

// RunServers runs all servers defined in the MCP file
func RunServers(ctx context.Context, mcpFilePath string) error {
	mcpConfig, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse mcp file: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(mcpConfig.Servers))

	for _, s := range mcpConfig.Servers {
		go func(server *mcpfile.MCPServer) {
			defer wg.Done()
			err := RunServer(ctx, server)
			if err != nil {
				log.Printf("error running server: %s", err.Error())
			}
		}(s)
	}

	wg.Wait()
	return nil
}

// checkToolAuthorization verifies if user has required scopes for a tool
func checkToolAuthorization(ctx context.Context, requiredScopes []string) error {
	if len(requiredScopes) == 0 {
		return nil // No scopes required
	}

	userClaims := oauth.GetClaimsFromContext(ctx)
	if userClaims == nil {
		return fmt.Errorf("no authentication context found")
	}

	// Split the scope string into individual scopes
	userScopes := strings.Split(userClaims.Scope, " ")

	// Check if user has all required scopes
	for _, requiredScope := range requiredScopes {
		if !slices.Contains(userScopes, requiredScope) {
			return fmt.Errorf("missing required scope '%s'", requiredScope)
		}
	}

	return nil
}

// createAuthorizedToolHandler wraps a tool handler with authorization checks
func createAuthorizedToolHandler(tool *mcpfile.Tool) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Check if user has required scopes for this tool
		if err := checkToolAuthorization(ctx, tool.RequiredScopes); err != nil {
			return nil, fmt.Errorf("forbidden: %s for tool '%s'", err.Error(), tool.Name)
		}

		// User has sufficient permissions, proceed with tool execution
		return tool.HandleRequest(ctx, req)
	}
}

func filterAuthorizedTools(mcpServerConfig *mcpfile.MCPServer) server.ToolFilterFunc {
	return func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		var allowedTools []mcp.Tool

		for _, tool := range tools {
			for _, toolConfig := range mcpServerConfig.Tools {
				if tool.Name == toolConfig.Name {
					if err := checkToolAuthorization(ctx, toolConfig.RequiredScopes); err != nil {
						fmt.Printf("user missed required scope to view %s tool: %s\n", tool.Name, err.Error())
					} else {
						fmt.Printf("user has all required scopes (%s) to view %s tool\n", strings.Join(toolConfig.RequiredScopes, ", "), tool.Name)
						allowedTools = append(allowedTools, tool)
					}
				}
			}
		}

		return allowedTools
	}
}

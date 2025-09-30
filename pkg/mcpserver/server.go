package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/cli"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/oauth"
)

func MakeServer(mcpServer *mcpfile.MCPServer) (*mcp.Server, error) {
	// Validate the server configuration before creating the server
	if err := mcpServer.Validate(invocation.InvocationValidator); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	return makeServerWithoutValidation(mcpServer)
}

// makeServerWithoutValidation creates a server without performing validation
// This is used internally when validation has already been performed
func makeServerWithoutValidation(mcpServer *mcpfile.MCPServer) (*mcp.Server, error) {
	return makeServerWithTools(mcpServer, mcpServer.Tools)
}

func RunServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	// Validate the server configuration before running
	if err := mcpServerConfig.Validate(invocation.InvocationValidator); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	switch strings.ToLower(mcpServerConfig.Runtime.TransportProtocol) {
	case mcpfile.TransportProtocolStreamableHttp:
		return runStreamableHttpServer(ctx, mcpServerConfig)
	case mcpfile.TransportProtocolStdio:
		return runStdioServer(ctx, mcpServerConfig)
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

	for _, s := range mcpConfig.Servers {
		err = s.Validate(invocation.InvocationValidator)
		if err != nil {
			return fmt.Errorf("mcp file contains an invalid server: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errChn := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(len(mcpConfig.Servers))

	for _, s := range mcpConfig.Servers {
		go func(server *mcpfile.MCPServer) {
			defer wg.Done()
			err := RunServer(ctx, server)
			if err != nil {
				select {
				case errChn <- fmt.Errorf("error running server: %s", err.Error()):
				default:
				}
			}
		}(s)
	}

	var firstErr error
	select {
	case firstErr = <-errChn:
		cancel()
	case <-ctx.Done():
	}

	wg.Wait()
	return firstErr
}

func runStreamableHttpServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	sm := NewServerManager(mcpServerConfig)
	// Create a root mux to handle different endpoints
	mux := http.NewServeMux()

	// Set up MCP server under /mcp (or whatever is under BasePath)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		s, err := sm.ServerFromContext(r.Context())
		if err != nil {
			return nil
		}

		return s
	}, &mcp.StreamableHTTPOptions{
		Stateless: mcpServerConfig.Runtime.StreamableHTTPConfig.Stateless,
	})

	oauthHandler := oauth.Middleware(mcpServerConfig)(handler)

	mux.Handle(mcpServerConfig.Runtime.StreamableHTTPConfig.BasePath, oauthHandler)

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
		if mcpServerConfig.Runtime.StreamableHTTPConfig.TLS != nil {
			if err := srv.ListenAndServeTLS(
				mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.CertFile,
				mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.KeyFile,
			); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
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
}

func runStdioServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	s, err := makeServerWithoutValidation(mcpServerConfig)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return s.Run(ctx, &mcp.StdioTransport{})
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
func createAuthorizedToolHandler(tool *mcpfile.Tool) (mcp.ToolHandler, error) {
	invoker, err := invocation.CreateInvoker(tool)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoker for tool %s: %w", tool.Name, err)
	}

	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Check if user has required scopes for this tool
		if err := checkToolAuthorization(ctx, tool.RequiredScopes); err != nil {
			return utils.McpTextError("forbidden: %s fpr tool '%s'", err.Error(), tool.Name), fmt.Errorf("forbidden: %s for tool '%s'", err.Error(), tool.Name)
		}

		return invoker.Invoke(ctx, req)
	}, nil
}

// makeServerWithTools makes a server using the server metadata in mcpServer but with the tools specified in tools
// this is useful for creating servers with filtered tool lists
func makeServerWithTools(mcpServer *mcpfile.MCPServer, tools []*mcpfile.Tool) (*mcp.Server, error) {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    mcpServer.Name,
		Version: mcpServer.Version,
	}, &mcp.ServerOptions{
		HasTools: len(mcpServer.Tools) > 0,
	})

	var toolErr error
	for _, t := range tools {
		handler, err := createAuthorizedToolHandler(t)
		if err != nil {
			toolErr = errors.Join(toolErr, err)
			continue
		}

		tool := &mcp.Tool{
			Name:        t.Name,
			Description: t.Description,
			Title:       t.Title,
			InputSchema: t.InputSchema,
		}

		// Only set OutputSchema if it's not nil to avoid typed nil issues
		if t.OutputSchema != nil {
			tool.OutputSchema = t.OutputSchema
		}

		s.AddTool(tool, handler)
	}

	return s, toolErr
}

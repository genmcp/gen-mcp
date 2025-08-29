package mcpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
		server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
			// TODO: remove - just for logging ATM
			claims := oauth.GetClaimsFromContext(ctx)
			if claims != nil {
				fmt.Printf("claims: %+v\n", claims)
			} else {
				fmt.Println("claims not found")
			}

			return tools
		}),
	)

	for _, t := range mcpServer.Tools {
		s.AddTool(
			mcp.NewTool(
				t.Name,
				t.GetMCPToolOpts()...,
			),
			t.HandleRequest,
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

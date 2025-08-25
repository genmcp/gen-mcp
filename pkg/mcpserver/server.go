package mcpserver

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func MakeServer(mcpServer *mcpfile.MCPServer) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer(
		mcpServer.Name,
		mcpServer.Version,
		server.WithToolCapabilities(true),
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

func RunServer(mcpServer *mcpfile.MCPServer) error {
	s := MakeServer(mcpServer)

	switch strings.ToLower(mcpServer.Runtime.TransportProtocol) {
	case mcpfile.TransportProtocolStreamableHttp:
		httpServer := server.NewStreamableHTTPServer(s, server.WithEndpointPath(mcpServer.Runtime.StreamableHTTPConfig.BasePath))
		fmt.Printf("starting listen on :%d\n", mcpServer.Runtime.StreamableHTTPConfig.Port)
		if err := httpServer.Start(fmt.Sprintf(":%d", mcpServer.Runtime.StreamableHTTPConfig.Port)); err != nil {
			return err
		}

		return nil
	case mcpfile.TransportProtocolStdio:
		if err := server.ServeStdio(s); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("tried running invalid transport protocol")
	}
}

// RunServers runs all servers defined in the MCP file
func RunServers(mcpFilePath string) error {
	mcpConfig, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse mcp file: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(mcpConfig.Servers))

	for _, s := range mcpConfig.Servers {
		go func(server *mcpfile.MCPServer) {
			defer wg.Done()
			err := RunServer(server)
			if err != nil {
				log.Printf("error running server: %s", err.Error())
			}
		}(s)
	}

	wg.Wait()
	return nil
}

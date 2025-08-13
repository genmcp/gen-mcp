package mcpserver

import (
	"fmt"
	"strings"

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
		httpServer := server.NewStreamableHTTPServer(s)
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

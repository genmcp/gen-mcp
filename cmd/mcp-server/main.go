package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Cali0707/AutoMCP/pkg/mcpfile"
	"github.com/Cali0707/AutoMCP/pkg/mcpserver"
	"github.com/mark3labs/mcp-go/server"
)
 
func main() {
	mcpFilePath := os.Getenv("MCP_FILE_PATH")

	mcp, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		log.Panicf("failed to parse mcp file: %s", err.Error())
	}

	servers := make([]*server.MCPServer, 0, len(mcp.Servers))
	for _, s := range mcp.Servers {
		servers = append(servers, mcpserver.MakeServer(s))
	}

	httpServer := server.NewStreamableHTTPServer(servers[0])
	if err := httpServer.Start(":8080"); err != nil {
		log.Fatal(err)
	}
	// TODO: run servers somehow
}
 

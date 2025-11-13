package main

import (
	"context"
	"fmt"
	"os"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
)

func main() {
	mcpFilePath := os.Getenv("MCP_FILE_PATH")

	mcpFile, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		fmt.Printf("failed to parse MCP file: %s\n", err)
		os.Exit(1)
	}

	if err := mcpserver.RunServer(context.Background(), &mcpFile.MCPServer); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/genmcp/gen-mcp/pkg/mcpserver"
)

func main() {
	mcpFilePath := os.Getenv("MCP_FILE_PATH")

	if err := mcpserver.RunServers(context.Background(), mcpFilePath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

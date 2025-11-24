package main

import (
	"context"
	"fmt"
	"os"

	"github.com/genmcp/gen-mcp/pkg/runtime"
)

func main() {
	toolDefinitionsPath := os.Getenv("MCP_FILE_PATH")
	serverConfigPath := os.Getenv("MCP_SERVER_CONFIG_PATH")

	if toolDefinitionsPath == "" {
		fmt.Println("MCP_FILE_PATH environment variable is required")
		os.Exit(1)
	}

	if serverConfigPath == "" {
		fmt.Println("MCP_SERVER_CONFIG_PATH environment variable is required")
		os.Exit(1)
	}

	if err := runtime.RunServer(context.Background(), toolDefinitionsPath, serverConfigPath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

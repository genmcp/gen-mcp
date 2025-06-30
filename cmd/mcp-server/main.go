package main

import (
	"log"
	"os"

	"github.com/Cali0707/AutoMCP/pkg/mcpfile"
	"github.com/Cali0707/AutoMCP/pkg/mcpserver"

)
 
func main() {
	mcpFilePath := os.Getenv("MCP_FILE_PATH")

	mcp, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		log.Panicf("failed to parse mcp file: %s", err.Error())
	}

	for _, s := range mcp.Servers {
		mcpserver.MakeServer(s)
	}

	// TODO: run servers somehow
}
 

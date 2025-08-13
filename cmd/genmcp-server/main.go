package main

import (
	"log"
	"os"
	"sync"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
)

func main() {
	mcpFilePath := os.Getenv("MCP_FILE_PATH")

	mcp, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		log.Panicf("failed to parse mcp file: %s", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(len(mcp.Servers))

	for _, s := range mcp.Servers {
		go func() {
			err := mcpserver.RunServer(s)
			if err != nil {
				log.Printf("error running server: %s", err.Error())
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

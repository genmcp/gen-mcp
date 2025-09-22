package utils

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func McpTextError(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
		IsError: true,
	}
}

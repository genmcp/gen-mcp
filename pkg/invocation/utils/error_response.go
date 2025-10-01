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

func McpPromptTextError(format string, args ...any) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: fmt.Sprintf(format, args...),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: fmt.Sprintf(format, args...)},
			},
		},
	}
}

package invocation

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Invoker interface {
	Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

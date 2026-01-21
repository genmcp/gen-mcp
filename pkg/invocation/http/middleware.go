package http

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WithHTTPClientMiddleware creates an MCP middleware that injects a configured
// HTTP client into the request context. This client is used by HTTP invokers
// for outbound requests and can be configured with custom CA certificates.
// If client is nil, http.DefaultClient will be used.
func WithHTTPClientMiddleware(client *http.Client) mcp.Middleware {
	// Ensure we never store nil - use DefaultClient as fallback
	if client == nil {
		client = http.DefaultClient
	}
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			ctx = WithHTTPClient(ctx, client)
			return next(ctx, method, req)
		}
	}
}

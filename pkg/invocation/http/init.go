package http

import "github.com/genmcp/gen-mcp/pkg/invocation"

const (
	InvocationType = "http"
)

func init() {
	invocation.RegisterFactory(InvocationType, &InvokerFactory{})
}

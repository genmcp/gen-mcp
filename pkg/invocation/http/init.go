package http

import "github.com/genmcp/gen-mcp/pkg/invocation"

const (
	InvocationType = "http"
)

func init() {
	invocation.RegisterParser(InvocationType, &Parser{})
	invocation.RegisterFactory(InvocationType, &InvokerFactory{})
}

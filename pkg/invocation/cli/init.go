package cli

import "github.com/genmcp/gen-mcp/pkg/invocation"

const (
	InvocationType = "cli"
)

func init() {
	invocation.RegisterFactory(InvocationType, &InvokerFactory{})
}

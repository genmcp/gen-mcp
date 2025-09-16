package invocation

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func CreateInvoker(tool *mcpfile.Tool) (Invoker, error) {
	config, err := ParseInvocation(tool.InvocationType, tool.InvocationData, tool)
	if err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	factory, exists := globalRegistry.factories[tool.InvocationType]
	if !exists {
		return nil, fmt.Errorf("no invoker factory for type: '%s'", tool.InvocationType)
	}

	return factory.CreateInvoker(config, tool.ResolvedInputSchema)
}

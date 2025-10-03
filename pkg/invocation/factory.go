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

func CreatePromptInvoker(prompt *mcpfile.Prompt) (Invoker, error) {
	config, err := ParsePromptInvocation(prompt.InvocationType, prompt.InvocationData, prompt)
	if err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	factory, exists := globalRegistry.factories[prompt.InvocationType]
	if !exists {
		return nil, fmt.Errorf("no invoker factory for type: '%s'", prompt.InvocationType)
	}

	return factory.CreateInvoker(config, prompt.ResolvedInputSchema)
}

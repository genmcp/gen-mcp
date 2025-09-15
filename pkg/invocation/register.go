package invocation

import (
	"encoding/json"
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

type Registry struct {
	parsers   map[string]InvocationConfigParser
	factories map[string]InvokerFactory
}

var globalRegistry = &Registry{
	parsers:   make(map[string]InvocationConfigParser),
	factories: make(map[string]InvokerFactory),
}

func RegisterParser(invocationType string, parser InvocationConfigParser) {
	globalRegistry.parsers[invocationType] = parser
}

func RegisterFactory(invocationType string, factory InvokerFactory) {
	globalRegistry.factories[invocationType] = factory
}

func ParseInvocation(invocationType string, data json.RawMessage, tool *mcpfile.Tool) (InvocationConfig, error) {
	parser, exists := globalRegistry.parsers[invocationType]
	if !exists {
		return nil, fmt.Errorf("unknown invocation type: '%s'", invocationType)
	}

	return parser.Parse(data, tool)
}

func InvocationValidator(invocationType string, data json.RawMessage, tool *mcpfile.Tool) error {
	config, err := ParseInvocation(invocationType, data, tool)
	if err != nil {
		return fmt.Errorf("failed to parse invocation: %w", err)
	}

	return config.Validate()
}

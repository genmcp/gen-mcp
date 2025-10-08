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

func ParsePromptInvocation(invocationType string, data json.RawMessage, prompt *mcpfile.Prompt) (InvocationConfig, error) {
	parser, exists := globalRegistry.parsers[invocationType]
	if !exists {
		return nil, fmt.Errorf("unknown invocation type: '%s'", invocationType)
	}

	return parser.ParsePrompt(data, prompt)
}

func ParseResourceInvocation(invocationType string, data json.RawMessage, resource *mcpfile.Resource) (InvocationConfig, error) {
	parser, exists := globalRegistry.parsers[invocationType]
	if !exists {
		return nil, fmt.Errorf("unknown invocation type: '%s'", invocationType)
	}

	return parser.ParseResource(data, resource)
}

func ParseResourceTemplateInvocation(invocationType string, data json.RawMessage, resourceTemplate *mcpfile.ResourceTemplate) (InvocationConfig, error) {
	parser, exists := globalRegistry.parsers[invocationType]
	if !exists {
		return nil, fmt.Errorf("unknown invocation type: '%s'", invocationType)
	}

	return parser.ParseResourceTemplate(data, resourceTemplate)
}

func InvocationValidator(invocationType string, data json.RawMessage, primitive mcpfile.Primitive) error {
	switch p := primitive.(type) {
	case *mcpfile.Tool:
		config, err := ParseInvocation(invocationType, data, p)
		if err != nil {
			return fmt.Errorf("failed to parse invocation: %w", err)
		}
		return config.Validate()
	case *mcpfile.Prompt:
		config, err := ParsePromptInvocation(invocationType, data, p)
		if err != nil {
			return fmt.Errorf("failed to parse invocation: %w", err)
		}
		return config.Validate()
	case *mcpfile.Resource:
		config, err := ParseResourceInvocation(invocationType, data, p)
		if err != nil {
			return fmt.Errorf("failed to parse invocation: %w", err)
		}
		return config.Validate()
	case *mcpfile.ResourceTemplate:
		config, err := ParseResourceTemplateInvocation(invocationType, data, p)
		if err != nil {
			return fmt.Errorf("failed to parse invocation: %w", err)
		}
		return config.Validate()
	default:
		return fmt.Errorf("unsupported primitive type %T", primitive)
	}
}

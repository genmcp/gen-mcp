package invocation

import (
	"fmt"
)

func CreateInvoker(primitive Primitive) (Invoker, error) {
	config := primitive.GetInvocationConfig()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	invocationType := primitive.GetInvocationType()
	factory, exists := globalRegistry.factories[invocationType]
	if !exists {
		return nil, fmt.Errorf("no invoker factory for type: '%s'", invocationType)
	}

	return factory.CreateInvoker(config, primitive)
}

func CreatePromptInvoker(primitive Primitive) (Invoker, error) {
	return CreateInvoker(primitive)
}

func CreateResourceInvoker(primitive Primitive) (Invoker, error) {
	return CreateInvoker(primitive)
}

func CreateResourceTemplateInvoker(primitive Primitive) (Invoker, error) {
	return CreateInvoker(primitive)
}

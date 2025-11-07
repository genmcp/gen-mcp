package invocation

import "fmt"

type Registry struct {
	factories map[string]InvokerFactory
}

var globalRegistry = &Registry{
	factories: make(map[string]InvokerFactory),
}

func RegisterFactory(invocationType string, factory InvokerFactory) {
	globalRegistry.factories[invocationType] = factory
}

func InvocationValidator(primitive Primitive) error {
	config := primitive.GetInvocationConfig()
	if config == nil {
		return fmt.Errorf("invocation config is nil")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	invocationType := primitive.GetInvocationType()
	factory, exists := globalRegistry.factories[invocationType]
	if !exists {
		return fmt.Errorf("unknown invocation type: '%s'", invocationType)
	}

	_, err := factory.CreateInvoker(config, primitive)
	return err
}

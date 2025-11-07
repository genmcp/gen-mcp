package extends

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

func init() {
	invocation.RegisterFactory(InvocationType, &ExtendsFactory{})
}

type ExtendsFactory struct{}

func (f *ExtendsFactory) NewConfig() invocation.InvocationConfig {
	return &ExtendsConfig{}
}

func (f *ExtendsFactory) CreateInvoker(config invocation.InvocationConfig, primitive invocation.Primitive) (invocation.Invoker, error) {
	cfg, ok := config.(*ExtendsConfig)
	if !ok {
		return nil, fmt.Errorf("invalid ExtendsConfig for extends invoker factory")
	}

	resolved, err := cfg.resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extends invocation config: %w", err)
	}

	factory, ok := invocation.GetFactory(resolved.Type)
	if !ok {
		return nil, fmt.Errorf("failed to get matching factory for invocation type '%s'", resolved.Type)
	}

	return factory.CreateInvoker(resolved.Config, primitive)
}

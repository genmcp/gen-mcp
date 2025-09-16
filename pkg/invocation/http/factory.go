package http

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/google/jsonschema-go/jsonschema"
)

type InvokerFactory struct{}

func (f *InvokerFactory) CreateInvoker(config invocation.InvocationConfig, schema *jsonschema.Resolved) (invocation.Invoker, error) {
	hic, ok := config.(*HttpInvocationConfig)
	if !ok {
		return nil, fmt.Errorf("invalid InvocationConfig type for http invoker factory")
	}

	invoker := &HttpInvoker{
		PathTemplate: hic.PathTemplate,
		PathIndeces:  hic.PathIndices,
		Method:       hic.Method,
		InputSchema:  schema,
	}

	return invoker, nil
}

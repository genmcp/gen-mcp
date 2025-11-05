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

	if hic.ParsedTemplate == nil {
		return nil, fmt.Errorf("parsed template is nil - parser may not have been run correctly")
	}

	invoker := &HttpInvoker{
		ParsedTemplate: hic.ParsedTemplate,
		Method:         hic.Method,
		InputSchema:    schema,
		URITemplate:    hic.URITemplate,
	}

	return invoker, nil
}

package cli

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/google/jsonschema-go/jsonschema"
)

type InvokerFactory struct{}

func (f *InvokerFactory) CreateInvoker(config invocation.InvocationConfig, schema *jsonschema.Resolved) (invocation.Invoker, error) {
	cic, ok := config.(*CliInvocationConfig)
	if !ok {
		return nil, fmt.Errorf("invalid InvocationConfig for cli invoker factory")
	}

	if cic.ParsedTemplate == nil {
		return nil, fmt.Errorf("parsed template is nil - parser may not have been run correctly")
	}

	return &CliInvoker{
		ParsedTemplate: cic.ParsedTemplate,
		InputSchema:    schema,
		URITemplate:    cic.URITemplate,
	}, nil
}

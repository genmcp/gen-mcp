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

	formatters := make(map[string]Formatter)
	for k, v := range cic.TemplateVariables {
		formatters[k] = v
	}

	for k := range cic.ParameterIndices {
		_, ok := formatters[k]
		if !ok {
			formatter, err := NewDummyFormatter(k, schema.Schema())
			if err != nil {
				return nil, fmt.Errorf("failed to create formatter for parameter: %w", err)
			}

			formatters[k] = formatter
		}
	}

	return &CliInvoker{
		CommandTemplate:    cic.Command,
		ArgumentIndices:    cic.ParameterIndices,
		ArgumentFormatters: formatters,
		InputSchema:        schema,
	}, nil

}

type DummyFormatter struct {
	formatString string
}

func NewDummyFormatter(paramName string, schema *jsonschema.Schema) (*DummyFormatter, error) {
	formatString, err := formatStringForParam(paramName, schema)
	if err != nil {
		return nil, err
	}

	return &DummyFormatter{
		formatString: formatString,
	}, nil
}

func (df *DummyFormatter) FormatValue(v any) string {
	return fmt.Sprintf(df.formatString, v)
}

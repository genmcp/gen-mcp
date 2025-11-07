package cli

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/yosida95/uritemplate/v3"
)

type InvokerFactory struct{}

func (f *InvokerFactory) NewConfig() invocation.InvocationConfig {
	return &CliInvocationConfig{}
}

func (f *InvokerFactory) CreateInvoker(config invocation.InvocationConfig, primitive invocation.Primitive) (invocation.Invoker, error) {
	cic, ok := config.(*CliInvocationConfig)
	if !ok {
		return nil, fmt.Errorf("invalid InvocationConfig for cli invoker factory")
	}

	// Create source factories for template parsing
	sources := template.CreateHeadersSourceFactory()

	formatters := make(map[string]template.VariableFormatter)
	for tvName, tv := range cic.TemplateVariables {
		formatter, err := template.NewTemplateFormatter(tv.Template, primitive.GetInputSchema(), tv.OmitIfFalse, sources)
		if err != nil {
			return nil, fmt.Errorf("failed to create template formatter for '%s': %w", tvName, err)
		}
		formatters[tvName] = formatter
	}

	parsedTemplate, err := template.ParseTemplate(cic.Command, template.TemplateParserOptions{
		InputSchema: primitive.GetInputSchema(),
		Formatters:  formatters,
		Sources:     sources,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse command template: %w", err)
	}

	if primitive.PrimitiveType() == "resource" {
		if len(parsedTemplate.Variables) > 0 {
			return nil, fmt.Errorf("static resource command cannot contain template variables")
		}
	}

	uriTemplate := primitive.GetURITemplate()
	if uriTemplate != "" {
		_, err = uritemplate.New(uriTemplate)
		if err != nil {
			return nil, fmt.Errorf("invalid URI template '%s': %w", uriTemplate, err)
		}
	}

	return &CliInvoker{
		ParsedTemplate: parsedTemplate,
		InputSchema:    primitive.GetResolvedInputSchema(),
		URITemplate:    uriTemplate,
	}, nil
}

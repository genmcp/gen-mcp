package http

import (
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/yosida95/uritemplate/v3"
)

type InvokerFactory struct{}

func (f *InvokerFactory) NewConfig() invocation.InvocationConfig {
	return &HttpInvocationConfig{}
}

func (f *InvokerFactory) CreateInvoker(config invocation.InvocationConfig, primitive invocation.Primitive) (invocation.Invoker, error) {
	hic, ok := config.(*HttpInvocationConfig)
	if !ok {
		return nil, fmt.Errorf("invalid InvocationConfig type for http invoker factory")
	}

	hic.Method = strings.ToUpper(hic.Method)

	// Create source factories for template parsing
	sources := template.CreateHeadersSourceFactory()

	parsedTemplate, err := template.ParseTemplate(hic.URL, template.TemplateParserOptions{
		InputSchema: primitive.GetInputSchema(),
		Sources:     sources,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL template: %w", err)
	}

	if primitive.PrimitiveType() == "resource" {
		if len(parsedTemplate.Variables) > 0 {
			return nil, fmt.Errorf("static resource URL cannot contain path parameters")
		}
	}

	uriTemplate := primitive.GetURITemplate()
	if uriTemplate != "" {
		_, err = uritemplate.New(uriTemplate)
		if err != nil {
			return nil, fmt.Errorf("invalid URI template '%s': %w", uriTemplate, err)
		}
	}

	// Parse header templates
	headerTemplates := make(map[string]*template.ParsedTemplate)
	for headerName, headerTemplate := range hic.Headers {
		pt, err := template.ParseTemplate(headerTemplate, template.TemplateParserOptions{
			InputSchema: primitive.GetInputSchema(),
			Sources:     sources,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to parse header template for '%s': %w", headerName, err)
		}
		headerTemplates[headerName] = pt
	}

	invoker := &HttpInvoker{
		ParsedTemplate:  parsedTemplate,
		HeaderTemplates: headerTemplates,
		Method:          hic.Method,
		InputSchema:     primitive.GetResolvedInputSchema(),
		URITemplate:     uriTemplate,
	}

	return invoker, nil
}

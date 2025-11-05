package http

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/yosida95/uritemplate/v3"
)

type Parser struct{}

var _ invocation.InvocationConfigParser = &Parser{}

type primitiveAdapter struct {
	InputSchema *jsonschema.Schema
}

func (p *Parser) Parse(data json.RawMessage, tool *mcpfile.Tool) (invocation.InvocationConfig, error) {
	return p.parsePrimitive(data, primitiveAdapter{InputSchema: tool.InputSchema})
}

func (p *Parser) ParsePrompt(data json.RawMessage, prompt *mcpfile.Prompt) (invocation.InvocationConfig, error) {
	return p.parsePrimitive(data, primitiveAdapter{InputSchema: prompt.InputSchema})
}

func (p *Parser) ParseResource(data json.RawMessage, resource *mcpfile.Resource) (invocation.InvocationConfig, error) {
	config, err := p.parsePrimitive(data, primitiveAdapter{InputSchema: resource.InputSchema})
	if err != nil {
		return nil, err
	}

	// Validate that static resources don't have path parameters that would never be filled
	if hic, ok := config.(*HttpInvocationConfig); ok {
		if len(hic.ParsedTemplate.Variables) > 0 {
			return nil, fmt.Errorf("static resource URL cannot contain path parameters")
		}
	}

	return config, nil
}

func (p *Parser) ParseResourceTemplate(data json.RawMessage, resourceTemplate *mcpfile.ResourceTemplate) (invocation.InvocationConfig, error) {
	config, err := p.parsePrimitive(data, primitiveAdapter{InputSchema: resourceTemplate.InputSchema})
	if err != nil {
		return nil, err
	}

	// Validate URI template syntax early during parsing
	_, err = uritemplate.New(resourceTemplate.URITemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid URI template '%s': %w", resourceTemplate.URITemplate, err)
	}

	// Set the URI template for resource templates
	if hic, ok := config.(*HttpInvocationConfig); ok {
		hic.URITemplate = resourceTemplate.URITemplate
	}

	return config, nil
}

func (p *Parser) parsePrimitive(data json.RawMessage, primitive primitiveAdapter) (invocation.InvocationConfig, error) {
	hic := &HttpInvocationConfig{}

	err := json.Unmarshal(data, hic)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal http invocation: %w", err)
	}

	// Normalize method to uppercase
	hic.Method = strings.ToUpper(hic.Method)

	// Parse the URL template using the template package
	parsedTemplate, err := template.ParseTemplate(hic.URL, template.TemplateParserOptions{
		InputSchema: primitive.InputSchema,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL template: %w", err)
	}

	hic.ParsedTemplate = parsedTemplate

	return hic, nil
}

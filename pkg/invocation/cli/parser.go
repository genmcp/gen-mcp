package cli

import (
	"encoding/json"
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/yosida95/uritemplate/v3"
)

type Parser struct{}

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

	// Validate that static resources don't have template variables that would never be filled
	if cic, ok := config.(*CliInvocationConfig); ok {
		if len(cic.ParsedTemplate.Variables) > 0 {
			return nil, fmt.Errorf("static resource command cannot contain template variables")
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
	if cic, ok := config.(*CliInvocationConfig); ok {
		cic.URITemplate = resourceTemplate.URITemplate
	}

	return config, nil
}

func (p *Parser) parsePrimitive(data json.RawMessage, primitive primitiveAdapter) (invocation.InvocationConfig, error) {
	config := &CliInvocationConfig{}

	err := json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// Convert TemplateVariables to formatters for the template package
	formatters := make(map[string]template.VariableFormatter)
	for tvName, tv := range config.TemplateVariables {
		formatter, err := template.NewTemplateFormatter(tv.Template, primitive.InputSchema, tv.OmitIfFalse)
		if err != nil {
			return nil, fmt.Errorf("failed to create template formatter for '%s': %w", tvName, err)
		}
		formatters[tvName] = formatter
	}

	// Parse the command template using the template package
	parsedTemplate, err := template.ParseTemplate(config.Command, template.TemplateParserOptions{
		InputSchema: primitive.InputSchema,
		Formatters:  formatters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse command template: %w", err)
	}

	config.ParsedTemplate = parsedTemplate

	return config, nil
}


package http

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
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
		if len(hic.PathIndices) > 0 {
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
	type Doppleganger HttpInvocationConfig

	hic := &HttpInvocationConfig{}

	tmp := struct {
		URL string `json:"url"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(hic),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal http invocation: %w", err)
	}

	hic.Method = strings.ToUpper(hic.Method)

	// iterate ofer the (possible) templated URL string and:
	// 1. collect any path paramters + their indices
	// 2. replace each path parameter with the correct string formatter

	chunks := []string{}
	paramIndices := make(map[string]int)
	paramIdx := 0
	var chunk strings.Builder
	for i := 0; i < len(tmp.URL); {
		if tmp.URL[i] == '{' {
			offset := strings.Index(tmp.URL[i:], "}") + i
			if offset-i == -1 {
				return nil, fmt.Errorf("unterminated path parameter found in URL")
			}

			paramName := tmp.URL[i+1 : offset]

			paramIndices[paramName] = paramIdx

			paramIdx++

			formatVar, err := utils.FormatStringForParam(paramName, primitive.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("failed to parse invocation url: %w", err)
			}

			chunks = append(chunks, chunk.String(), formatVar)
			chunk.Reset()

			i = offset + 1
			continue
		} else if tmp.URL[i] == '}' {
			return nil, fmt.Errorf("no opening bracket for a closing bracket in URL")
		}

		chunk.WriteByte(tmp.URL[i])
		i++
	}

	chunks = append(chunks, chunk.String())
	hic.PathTemplate = strings.Join(chunks, "")

	hic.PathIndices = paramIndices

	return hic, nil
}

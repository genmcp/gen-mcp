package http

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
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

func formatStringForParam(paramName string, schema *jsonschema.Schema) (string, error) {
	schema, err := lookupParam(paramName, schema)
	if err != nil {
		return "", err
	}

	switch schema.Type {
	case invocation.JsonSchemaTypeArray, invocation.JsonSchemaTypeNull, invocation.JsonSchemaTypeObject:
		return "%v", nil
	case invocation.JsonSchemaTypeBoolean:
		return "%b", nil
	case invocation.JsonSchemaTypeInteger:
		return "%d", nil
	case invocation.JsonSchemaTypeNumber:
		return "%f", nil
	case invocation.JsonSchemaTypeString:
		return "%s", nil
	default:
		return "", fmt.Errorf("unknown json schema type: '%s'", schema.Type)
	}
}

func lookupParam(paramName string, schema *jsonschema.Schema) (*jsonschema.Schema, error) {
	path := strings.Split(paramName, ".")
	currentSchema := schema
	var ok bool

	for _, p := range path {
		currentSchema, ok = currentSchema.Properties[p]
		if !ok {
			return nil, fmt.Errorf("http invocation has %s path parameter, but there is no corresponding property defined on the input schema", paramName)
		}
	}

	return currentSchema, nil
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

			formatVar, err := formatStringForParam(paramName, primitive.InputSchema)
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

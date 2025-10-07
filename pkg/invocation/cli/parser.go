package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
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
	return p.parsePrimitive(data, primitiveAdapter{InputSchema: resource.InputSchema})
}

func (p *Parser) ParseResourceTemplate(data json.RawMessage, resourceTemplate *mcpfile.ResourceTemplate) (invocation.InvocationConfig, error) {
	config, err := p.parsePrimitive(data, primitiveAdapter{InputSchema: resourceTemplate.InputSchema})
	if err != nil {
		return nil, err
	}

	// Set the URI template for resource templates
	if cic, ok := config.(*CliInvocationConfig); ok {
		cic.URITemplate = resourceTemplate.URITemplate
	}

	return config, nil
}

func (p *Parser) parsePrimitive(data json.RawMessage, primitive primitiveAdapter) (invocation.InvocationConfig, error) {
	type Doppleganger CliInvocationConfig

	config := &CliInvocationConfig{}

	tmp := struct {
		Command           string                     `json:"command"`
		TemplateVariables map[string]json.RawMessage `json:"templateVariables,omitempty"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(config),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	templateVariables := make(map[string]*TemplateVariable)
	for tvName, tvRaw := range tmp.TemplateVariables {
		tv, err := parseTemplateVariable(tvRaw, primitive)
		if err != nil {
			return nil, err
		}

		templateVariables[tvName] = tv
	}

	chunks := []string{}
	paramIndices := make(map[string]int)
	paramIdx := 0
	var chunk strings.Builder
	for i := 0; i < len(tmp.Command); {
		if tmp.Command[i] == '{' {
			offset := strings.Index(tmp.Command[i:], "}") + i
			if offset-1 == -1 {
				return nil, fmt.Errorf("unterminated parameter found in command")
			}

			paramName := tmp.Command[i+1 : offset]
			paramIndices[paramName] = paramIdx

			paramIdx++

			chunks = append(chunks, chunk.String(), "%s")
			chunk.Reset()

			i = offset + 1
			continue
		} else if tmp.Command[i] == '}' {
			return nil, fmt.Errorf("no opening bracket for a closing bracket in command")
		}

		chunk.WriteByte(tmp.Command[i])
		i++
	}

	chunks = append(chunks, chunk.String())
	config.Command = strings.Join(chunks, "")

	config.ParameterIndices = paramIndices
	config.TemplateVariables = templateVariables

	return config, nil
}

func parseTemplateVariable(data json.RawMessage, primitive primitiveAdapter) (*TemplateVariable, error) {
	type Doppleganger TemplateVariable

	tv := &TemplateVariable{}

	tmp := struct {
		Format string `json:"format"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(tv),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	openCount := strings.Count(tmp.Format, "{")
	closeCount := strings.Count(tmp.Format, "}")

	if openCount > 1 {
		return nil, fmt.Errorf("invalid number of parameters in template variable. Expected <= 1, received %d", openCount)
	}

	if openCount != closeCount {
		return nil, fmt.Errorf("number of opening brackets should match the number of closing brackets")
	}

	varStart := strings.Index(tmp.Format, "{")
	if varStart == -1 {
		tv.shouldFormat = false
		tv.Template = tmp.Format
		return tv, nil
	}

	varEnd := strings.Index(tmp.Format, "}")
	if varEnd == -1 {
		return nil, fmt.Errorf("unterminated parameter found in template variable format")
	} else if varEnd < varStart {
		return nil, fmt.Errorf("invalid parameter brackets found in template variable format (closing bracket before opening bracket)")
	}

	paramName := tmp.Format[varStart+1 : varEnd]

	formatString, err := utils.FormatStringForParam(paramName, primitive.InputSchema)
	if err != nil {
		return nil, err
	}

	tv.shouldFormat = true
	tv.Template = tmp.Format[:varStart] + formatString + tmp.Format[varEnd+1:]

	return tv, nil
}

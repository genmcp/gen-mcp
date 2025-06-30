package mcpfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	MCPFileVersion        = "0.0.1"
	JsonSchemaTypeArray   = "array"
	JsonSchemaTypeBoolean = "boolean"
	JsonSchemaTypeInteger = "integer"
	JsonSchemaTypeNumber  = "number"
	JsonSchemaTypeNull    = "null"
	JsonSchemaTypeObject  = "object"
	JsonSchemaTypeString  = "string"
)

type JsonSchema struct {
	Type                 string                 `json:"type"`                           // can be array, boolean, integer, number, null, object, or string
	Items                *JsonSchema            `json:"items,omitempty"`                // schema for items of an array
	Properties           map[string]*JsonSchema `json:"properties,omitempty"`           // properties of an object
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"` // allow extra properties for type object
	Required             []string               `json:"required,omitempty"`             // required properties for an object
	Description          string                 `json:"description,omitempty"`          // optional human readable description of the item
}

type Tool struct {
	Name         string      `json:"name"`                   // name of the tool
	Title        string      `json:"title,omitempty"`        // optional human readable name of the tool, for client display
	Description  string      `json:"description"`            // description of the tool
	InputSchema  *JsonSchema `json:"inputSchema"`            // input schema to call the tool
	OutputSchema *JsonSchema `json:"outputSchema,omitempty"` // optional output schema of the tool
	URL          url.URL     `json:"url"`                    // url for where the MCP server should call the tool
}

type MCPServer struct {
	Name    string  `json:"name"`            // name of the server
	Version string  `json:"version"`         // version of the server
	Tools   []*Tool `json:"tools,omitempty"` // set of tools available to the server
}

type MCPFile struct {
	FileVersion string       `json:"mcpFileVersion"`
	Servers     []*MCPServer `json:"servers,omitempty"`
}

func (js *JsonSchema) GetMCPToolOpt(name string, required bool) mcp.ToolOption {
	opts := make([]mcp.PropertyOption, 0, 2)
	if required {
		opts = append(opts, mcp.Required())
	}
	if js.Description != "" {
		opts = append(opts, mcp.Description(js.Description))
	}

	switch js.Type {
	case JsonSchemaTypeArray:
		return mcp.WithArray(name, opts...)
	case JsonSchemaTypeBoolean:
		return mcp.WithBoolean(name, opts...)
	case JsonSchemaTypeInteger:
		return mcp.WithNumber(name, opts...) // TODO: replace this with WithInt when https://github.com/mark3labs/mcp-go/pull/458 merges
	case JsonSchemaTypeNull:
		return nil
	case JsonSchemaTypeNumber:
		return mcp.WithNumber(name, opts...)
	case JsonSchemaTypeObject:
		// copy to a new map as go map types are distinct, even thought *JsonSchema is assignable to any
		newMap := make(map[string]any)
		for k, v := range js.Properties {
			newMap[k] = v
		}

		opts = append(opts, mcp.Properties(newMap))
		return mcp.WithObject(name, opts...)
	case JsonSchemaTypeString:
		return mcp.WithString(name, opts...)

	default:
		return nil
	}

}

func (t *Tool) GetMCPToolOpts() []mcp.ToolOption {
	opts := make([]mcp.ToolOption, 0, 8)

	if t.Description != "" {
		opts = append(opts, mcp.WithDescription(t.Description))
	}

	for propName, prop := range t.InputSchema.Properties {
		opts = append(opts, prop.GetMCPToolOpt(propName, slices.Contains(t.InputSchema.Required, propName)))
	}

	return opts
}

func (t *Tool) HandleRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fmt.Printf("received tool request\n")
	args := req.GetRawArguments()

	argsMap, ok := args.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("arguments were not a valid object"), nil
	}

	fmt.Printf("received arguments: %+v\n", args)

	// TODO: validation of all the properties in the request

	jsonData, err := json.Marshal(argsMap)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("could not marshal arguments to a json request body: %s", err.Error())), nil
	}

	request, err := http.NewRequest("POST", t.URL.String(), bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("request to tool endpoint failed: %s", err.Error())), nil
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	fmt.Printf("received response: %s\n", string(body))
	return mcp.NewToolResultText(string(body)), nil
}

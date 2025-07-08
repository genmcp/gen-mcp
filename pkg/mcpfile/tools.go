package mcpfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

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

func (h *HttpInvocation) HandleRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	request, err := http.NewRequest(h.Method, h.URL.String(), bytes.NewBuffer(jsonData))
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

func (t *Tool) HandleRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.Invocation.HandleRequest(ctx, req)
}

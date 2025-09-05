package mcpfile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os/exec"
	"slices"
	"strings"

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

func (t *Tool) parseToolCallParameter(name string, req mcp.CallToolRequest) (any, error) {
	param := t.InputSchema.Properties[name]
	switch param.Type {
	case JsonSchemaTypeArray:
		switch param.Items.Type {
		}
		return nil, nil
	case JsonSchemaTypeBoolean:
		return req.RequireBool(name)
	case JsonSchemaTypeInteger:
		return req.RequireInt(name)
	case JsonSchemaTypeNull:
		return nil, nil // TODO: think if we really want to support this
	case JsonSchemaTypeNumber:
		return req.RequireFloat(name)
	case JsonSchemaTypeObject:
		// TODO: figure out this recursive parsing stuff
		return nil, nil
	case JsonSchemaTypeString:
		return req.RequireString(name)
	default:
		panic("this should never happen: encountered unknown runtime type")
	}
}

func (h *HttpInvocation) HandleRequest(ctx context.Context, req mcp.CallToolRequest, t *Tool) (*mcp.CallToolResult, error) {
	args := req.GetRawArguments()

	argsMap, ok := args.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("arguments were not a valid object"), nil
	}

	var err error = nil
	pathParamValues := []any{}
	for _, paramName := range h.pathParameters {
		val, parseErr := t.parseToolCallParameter(paramName, req)
		err = errors.Join(err, parseErr)
		pathParamValues = append(pathParamValues, val)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("encountered error while parsing path parameters: %s", err.Error())), nil
	}

	fmt.Println(h.URL)
	url := fmt.Sprintf(h.URL, pathParamValues...)

	remainingArgs := map[string]any{}
	for paramName := range argsMap {
		if slices.Contains(h.pathParameters, paramName) {
			continue
		}
		if _, ok := t.InputSchema.Properties[paramName]; !ok && (t.InputSchema.AdditionalProperties == nil || !*t.InputSchema.AdditionalProperties) {
			continue
		}
		val, parseErr := t.parseToolCallParameter(paramName, req)
		if parseErr != nil {
			err = errors.Join(err, parseErr)
			continue
		}
		remainingArgs[paramName] = val
	}

	// build query string or request body
	var reqBody io.Reader = nil
	if h.Method == http.MethodGet || h.Method == http.MethodDelete || h.Method == http.MethodHead {
		queryParams := neturl.Values{}
		for k, v := range remainingArgs {
			queryParams.Add(k, fmt.Sprintf("%v", v))
		}
		url = fmt.Sprintf("%s?%s", url, queryParams.Encode())
	} else {
		jsonData, err := json.Marshal(remainingArgs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("could not marshal arguments to a json request body: %s", err.Error())), nil
		}

		reqBody = bytes.NewBuffer(jsonData)
	}

	request, err := http.NewRequest(h.Method, url, reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("could not create request: %s", err.Error())), nil
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("request to tool endpoint failed: %s", err.Error())), nil
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	body, _ := io.ReadAll(response.Body)
	return mcp.NewToolResultText(string(body)), nil
}

func (tv *TemplateVariable) formatValue(value any) string {
	if len(tv.formatParameters) > 0 {
		return fmt.Sprintf(tv.Format, value)
	}

	return tv.Format
}

func (c *CliInvocation) HandleRequest(ctx context.Context, req mcp.CallToolRequest, t *Tool) (*mcp.CallToolResult, error) {
	args := req.GetRawArguments()

	argsMap, ok := args.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("arguments were not a valid object"), nil
	}

	var err error = nil
	pathParamValues := []any{}
	for _, paramName := range c.commandParameters {
		val, parseErr := t.parseToolCallParameter(paramName, req)
		if parseErr != nil && !slices.Contains(t.InputSchema.Required, paramName) {
			pathParamValues = append(pathParamValues, "")
			continue
		}

		err = errors.Join(err, parseErr)
		if tv, ok := c.TemplateVariables[paramName]; ok {
			val = tv.formatValue(val)
		}

		pathParamValues = append(pathParamValues, val)
	}

	remainingArgs := []string{}
	for paramName := range argsMap {
		if slices.Contains(c.commandParameters, paramName) {
			continue
		}

		if _, ok := t.InputSchema.Properties[paramName]; !ok && (t.InputSchema.AdditionalProperties == nil || !*t.InputSchema.AdditionalProperties) {
			continue
		}
		val, parseErr := t.parseToolCallParameter(paramName, req)
		if parseErr != nil {
			err = errors.Join(err, parseErr)
			continue
		}

		remainingArgs = append(remainingArgs, fmt.Sprintf("--%s=%v", paramName, val))
	}

	command := fmt.Sprintf(c.Command, pathParamValues...) + " " + strings.Join(remainingArgs, " ")

	cmd := exec.Command("bash", "-c", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("encountered error while calling command: %s. output was: %s", err.Error(), string(output))), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

func (t *Tool) HandleRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return t.Invocation.HandleRequest(ctx, req, t)
}

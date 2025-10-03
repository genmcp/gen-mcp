package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	nethttp "net/http"
	neturl "net/url"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
)

const contentTypeHeader = "Content-Type"

type HttpInvoker struct {
	PathTemplate string               // template string for the request path
	PathIndeces  map[string]int       // map to where each path parameter should go in the path
	Method       string               // Http request method
	InputSchema  *jsonschema.Resolved // InputSchema for the tool
}

var _ invocation.Invoker = &HttpInvoker{}

func (hi *HttpInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ub := &urlBuilder{
		pathTemplate: hi.PathTemplate,
		pathIndeces:  hi.PathIndeces,
		pathValues:   make([]any, len(hi.PathIndeces)),
	}

	hasBody := hi.Method != nethttp.MethodGet && hi.Method != nethttp.MethodDelete && hi.Method != nethttp.MethodHead

	if !hasBody {
		ub.queryParams = neturl.Values{}
		ub.buildQuery = len(hi.PathIndeces) > 0
	}

	dj := &invocation.DynamicJson{
		Builders: []invocation.Builder{ub},
	}

	parsed, err := dj.ParseJson(req.Params.Arguments, hi.InputSchema.Schema())
	if err != nil {
		return utils.McpTextError("failed to parse tool call request: %s", err.Error()), err
	}

	err = hi.InputSchema.Validate(parsed)
	if err != nil {
		return utils.McpTextError("failed to validate parsed tool call request: %s", err.Error()), err
	}

	url, _ := ub.GetResult()

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := json.Marshal(deletePathsFromMap(parsed, slices.Collect(maps.Keys(hi.PathIndeces))))
		if err != nil {
			return utils.McpTextError("failed to marshal http request body: %s", err.Error()), err
		}

		reqBody = bytes.NewBuffer(bodyJson)
	}

	httpReq, err := nethttp.NewRequest(hi.Method, url.(string), reqBody)
	if err != nil {
		return utils.McpTextError("failed to create http request: %s", err.Error()), err
	}
	if hasBody {
		httpReq.Header.Set(contentTypeHeader, "application/json; charset=UTF-8")
	}

	client := &nethttp.Client{}
	response, err := client.Do(httpReq)
	if err != nil {
		return utils.McpTextError("failed to execute http request: %s", err.Error()), err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, _ := io.ReadAll(response.Body)

	res := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(body),
			},
		},
		IsError: response.StatusCode < 200 || response.StatusCode >= 300,
	}

	contentType := response.Header.Get(contentTypeHeader)
	if strings.Contains(contentType, "application/json") {
		var data map[string]any
		err := json.Unmarshal(body, &data)
		if err == nil {
			res.StructuredContent = data
		}
	}

	return res, nil
}

type urlBuilder struct {
	pathTemplate string
	pathIndeces  map[string]int
	pathValues   []any
	queryParams  neturl.Values
	buildQuery   bool
}

var _ invocation.Builder = &urlBuilder{}

func (ub *urlBuilder) SetField(path string, value any) {
	_, ok := ub.pathIndeces[path]
	if ok {
		ub.pathValues[ub.pathIndeces[path]] = value
		return
	}

	if !ub.buildQuery {
		return
	}

	if s, ok := value.(string); ok {
		ub.queryParams.Add(path, s)
	} else {
		ub.queryParams.Add(path, fmt.Sprintf("%v", value))
	}
}

func (ub *urlBuilder) GetResult() (any, error) {
	if ub.buildQuery {
		return fmt.Sprintf(ub.pathTemplate, ub.pathValues...) + "?" + ub.queryParams.Encode(), nil
	}

	return fmt.Sprintf(ub.pathTemplate, ub.pathValues...), nil
}

func deletePathsFromMap(m map[string]any, paths []string) map[string]any {
	for _, path := range paths {
		deletePathFromMap(m, path)
	}

	return m
}

func deletePathFromMap(m map[string]any, path string) {
	keys := strings.Split(path, ".")
	var parentMap map[string]any
	currentMap := m

	for _, key := range keys[:len(keys)-1] {
		if nestedMap, ok := currentMap[key].(map[string]any); ok {
			parentMap = currentMap
			currentMap = nestedMap
		} else {
			return
		}
	}

	key := keys[len(keys)-1]
	delete(currentMap, key)

	if len(currentMap) == 0 && parentMap != nil {
		delete(parentMap, keys[len(keys)-2])
	}
}

func (hi *HttpInvoker) InvokePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	ub := &urlBuilder{
		pathTemplate: hi.PathTemplate,
		pathIndeces:  hi.PathIndeces,
		pathValues:   make([]any, len(hi.PathIndeces)),
	}

	hasBody := hi.Method != nethttp.MethodGet && hi.Method != nethttp.MethodDelete && hi.Method != nethttp.MethodHead

	if !hasBody {
		ub.queryParams = neturl.Values{}
		ub.buildQuery = len(hi.PathIndeces) > 0
	}

	dj := &invocation.DynamicJson{
		Builders: []invocation.Builder{ub},
	}

	args := req.Params.Arguments
	if args == nil {
		args = make(map[string]string)
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		return utils.McpPromptTextError("failed to marshal prompt request arguments: %s", err.Error()), err
	}

	parsed, err := dj.ParseJson(argsBytes, hi.InputSchema.Schema())
	if err != nil {
		return utils.McpPromptTextError("failed to parse prompt request arguments: %s", err.Error()), err
	}

	if err := hi.InputSchema.Validate(parsed); err != nil {
		return utils.McpPromptTextError("failed to validate prompt request arguments: %s", err.Error()), err
	}

	url, _ := ub.GetResult()

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := json.Marshal(deletePathsFromMap(parsed, slices.Collect(maps.Keys(hi.PathIndeces))))
		if err != nil {
			return utils.McpPromptTextError("failed to marshal http request body: %s", err.Error()), err
		}

		reqBody = bytes.NewBuffer(bodyJson)
	}

	httpReq, err := nethttp.NewRequestWithContext(ctx, hi.Method, url.(string), reqBody)
	if err != nil {
		return utils.McpPromptTextError("failed to create http request: %s", err.Error()), err
	}
	if hasBody {
		httpReq.Header.Set(contentTypeHeader, "application/json; charset=UTF-8")
	}

	client := &nethttp.Client{}

	response, err := client.Do(httpReq)
	if err != nil {
		return utils.McpPromptTextError("failed to execute http request: %s", err.Error()), err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, _ := io.ReadAll(response.Body)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return utils.McpPromptTextError("http request failed with status %d", response.StatusCode), fmt.Errorf("http request failed with status %d", response.StatusCode)
	}

	result := &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: string(body)},
			},
		},
	}

	return result, nil
}

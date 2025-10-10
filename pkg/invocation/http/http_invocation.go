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
	"github.com/yosida95/uritemplate/v3"
	"go.uber.org/zap"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
)

const contentTypeHeader = "Content-Type"

type HttpInvoker struct {
	PathTemplate string               // template string for the request path
	PathIndeces  map[string]int       // map to where each path parameter should go in the path
	Method       string               // Http request method
	InputSchema  *jsonschema.Resolved // InputSchema for the tool
	URITemplate  string               // MCP URI template (for resource templates only)
}

var _ invocation.Invoker = &HttpInvoker{}

func (hi *HttpInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger := logging.FromContext(ctx)  // Sent to both server and client
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting HTTP tool invocation")

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
		logger.Error("Failed to parse HTTP tool call request", zap.Error(err))
		return nil, fmt.Errorf("failed to parse tool call request: %w", err)
	}

	err = hi.InputSchema.Validate(parsed)
	if err != nil {
		logger.Error("Failed to validate HTTP tool call request", zap.Error(err))
		return nil, fmt.Errorf("failed to validate tool call request: %w", err)
	}

	url, _ := ub.GetResult()

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := json.Marshal(deletePathsFromMap(parsed, slices.Collect(maps.Keys(hi.PathIndeces))))
		if err != nil {
			logger.Error("Failed to marshal HTTP request body", zap.Error(err))
			return nil, fmt.Errorf("failed to prepare request body: %w", err)
		}

		reqBody = bytes.NewBuffer(bodyJson)
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Debug("Executing HTTP request",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.Bool("has_body", hasBody))

	httpReq, err := nethttp.NewRequest(hi.Method, url.(string), reqBody)
	if err != nil {
		logger.Error("Failed to create HTTP request",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	if hasBody {
		httpReq.Header.Set(contentTypeHeader, "application/json; charset=UTF-8")
	}

	client := &nethttp.Client{}
	response, err := client.Do(httpReq)
	if err != nil {
		baseLogger.Error("HTTP request execution failed",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.Error(err))
		logger.Error("HTTP request execution failed")
		return utils.McpTextError("HTTP request failed: %v", err), nil
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, _ := io.ReadAll(response.Body)

	// Server-side only logging with sensitive HTTP details
	baseLogger.Info("HTTP request completed",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.Int("status_code", response.StatusCode),
		zap.Int("response_length", len(body)))

	logger.Info("HTTP tool invocation completed successfully")

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
		base := fmt.Sprintf(ub.pathTemplate, ub.pathValues...)
		q := ub.queryParams.Encode()
		if q == "" {
			return base, nil
		}
		return base + "?" + q, nil
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
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting HTTP prompt invocation")

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
		logger.Error("Failed to marshal HTTP prompt request arguments", zap.Error(err))
		return nil, fmt.Errorf("failed to prepare prompt request: %w", err)
	}

	parsed, err := dj.ParseJson(argsBytes, hi.InputSchema.Schema())
	if err != nil {
		logger.Error("Failed to parse HTTP prompt request arguments", zap.Error(err))
		return nil, fmt.Errorf("failed to parse prompt request: %w", err)
	}

	if err := hi.InputSchema.Validate(parsed); err != nil {
		logger.Error("Failed to validate HTTP prompt request arguments", zap.Error(err))
		return nil, fmt.Errorf("failed to validate prompt request: %w", err)
	}

	url, _ := ub.GetResult()

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := json.Marshal(deletePathsFromMap(parsed, slices.Collect(maps.Keys(hi.PathIndeces))))
		if err != nil {
			logger.Error("Failed to marshal HTTP prompt request body", zap.Error(err))
			return nil, fmt.Errorf("failed to prepare request body: %w", err)
		}

		reqBody = bytes.NewBuffer(bodyJson)
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Debug("Executing HTTP prompt request",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.Bool("has_body", hasBody))

	httpReq, err := nethttp.NewRequestWithContext(ctx, hi.Method, url.(string), reqBody)
	if err != nil {
		logger.Error("Failed to create HTTP prompt request",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	if hasBody {
		httpReq.Header.Set(contentTypeHeader, "application/json; charset=UTF-8")
	}

	client := &nethttp.Client{}

	response, err := client.Do(httpReq)
	if err != nil {
		baseLogger.Error("HTTP prompt request execution failed",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.Error(err))
		logger.Error("HTTP prompt request execution failed")
		return utils.McpPromptTextError("HTTP request failed: %v", err), nil
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to read HTTP prompt response body", zap.Error(readErr))
		// Log generic error for client
		logger.Error("HTTP prompt response reading failed", zap.String("error", "response reading error"))
		return utils.McpPromptTextError("failed to read http response body"), nil
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Info("HTTP prompt request completed",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.Int("status_code", response.StatusCode),
		zap.Int("response_length", len(body)))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Log detailed error server-side only
		baseLogger.Error("HTTP prompt request failed with error status",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.Int("status_code", response.StatusCode))
		// Log generic error for client
		logger.Error("HTTP prompt request failed with error status", zap.String("error", "http error status"))
		return utils.McpPromptTextError("http request failed"), nil
	}

	logger.Info("HTTP prompt invocation completed successfully")

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

func (hi *HttpInvoker) InvokeResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting HTTP resource invocation", zap.String("uri", req.Params.URI))

	ub := &urlBuilder{
		pathTemplate: hi.PathTemplate,
		pathIndeces:  hi.PathIndeces,
		pathValues:   make([]any, len(hi.PathIndeces)),
	}

	ub.queryParams = neturl.Values{}
	ub.buildQuery = len(hi.PathIndeces) > 0

	url, _ := ub.GetResult()

	// Server-side only logging with sensitive HTTP details
	baseLogger.Debug("Executing HTTP resource request",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.String("uri", req.Params.URI))

	var reqBody io.Reader = nil
	httpReq, err := nethttp.NewRequestWithContext(ctx, hi.Method, url.(string), reqBody)
	if err != nil {
		// Log detailed error with sensitive URL server-side only
		baseLogger.Error("Failed to create HTTP resource request",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log safe details to client
		logger.Error("Failed to create HTTP resource request",
			zap.String("method", hi.Method),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	client := &nethttp.Client{}

	response, err := client.Do(httpReq)
	if err != nil {
		// Server-side only logging with sensitive URL details
		baseLogger.Error("HTTP resource request execution failed",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log safe details to client
		logger.Error("HTTP resource request execution failed",
			zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		logger.Error("Failed to read HTTP resource response body",
			zap.String("uri", req.Params.URI),
			zap.Error(readErr))
		return nil, fmt.Errorf("failed to read http response body: %w", readErr)
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Info("HTTP resource request completed",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.String("uri", req.Params.URI),
		zap.Int("status_code", response.StatusCode),
		zap.Int("response_length", len(body)))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Server-side only logging with sensitive URL details
		baseLogger.Error("HTTP resource request failed with error status",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Int("status_code", response.StatusCode))
		// Log safe details to client
		logger.Error("HTTP resource request failed with error status",
			zap.String("uri", req.Params.URI),
			zap.Int("status_code", response.StatusCode))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	logger.Info("HTTP resource invocation completed successfully", zap.String("uri", req.Params.URI))

	mimeType := response.Header.Get(contentTypeHeader)
	if mimeType == "" {
		mimeType = "text/plain"
	}

	result := &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: mimeType,
				Text:     string(body),
			},
		},
	}

	return result, nil
}

func (hi *HttpInvoker) InvokeResourceTemplate(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting HTTP resource template invocation",
		zap.String("uri", req.Params.URI))

	ub := &urlBuilder{
		pathTemplate: hi.PathTemplate,
		pathIndeces:  hi.PathIndeces,
		pathValues:   make([]any, len(hi.PathIndeces)),
	}

	ub.queryParams = neturl.Values{}
	ub.buildQuery = true

	// URI template syntax is validated during parsing, so we can safely use it here
	argsMap := make(map[string]any)
	uriTmpl, _ := uritemplate.New(hi.URITemplate)

	// Match the incoming URI against the template to extract argument values
	matches := uriTmpl.Match(req.Params.URI)
	if matches == nil {
		logger.Error("URI does not match HTTP resource template",
			zap.String("uri", req.Params.URI),
			zap.String("template", hi.URITemplate))
		return nil, fmt.Errorf("URI does not match template")
	}

	// Convert uritemplate.Values to map[string]any
	for _, paramName := range uriTmpl.Varnames() {
		if val := matches.Get(paramName); val.Valid() {
			argsMap[paramName] = val.String()
		} else {
			logger.Error("Missing required parameter in resource template",
				zap.String("parameter", paramName),
				zap.String("uri", req.Params.URI),
				zap.String("template", hi.URITemplate))
			return nil, fmt.Errorf("missing required parameter: %s", paramName)
		}
	}

	argsBytes, err := json.Marshal(argsMap)
	if err != nil {
		logger.Error("Failed to marshal HTTP resource template arguments",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to prepare arguments: %w", err)
	}

	dj := &invocation.DynamicJson{
		Builders: []invocation.Builder{ub},
	}

	parsed, err := dj.ParseJson(argsBytes, hi.InputSchema.Schema())
	if err != nil {
		logger.Error("Failed to parse HTTP resource template request",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse resource template request: %w", err)
	}

	if err := hi.InputSchema.Validate(parsed); err != nil {
		logger.Error("Failed to validate HTTP resource template request",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to validate resource template request: %w", err)
	}

	url, _ := ub.GetResult()

	// Server-side only logging with sensitive HTTP details
	baseLogger.Debug("Executing HTTP resource template request",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.String("uri", req.Params.URI),
		zap.String("template", hi.URITemplate))

	httpReq, err := nethttp.NewRequestWithContext(ctx, hi.Method, url.(string), nil)
	if err != nil {
		// Log detailed error with sensitive URL server-side only
		baseLogger.Error("Failed to create HTTP resource template request",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log safe details to client
		logger.Error("Failed to create HTTP resource template request",
			zap.String("method", hi.Method),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	client := &nethttp.Client{}
	response, err := client.Do(httpReq)
	if err != nil {
		// Server-side only logging with sensitive URL details
		baseLogger.Error("HTTP resource template request execution failed",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log safe details to client
		logger.Error("HTTP resource template request execution failed",
			zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		logger.Error("Failed to read HTTP resource template response body",
			zap.String("uri", req.Params.URI),
			zap.Error(readErr))
		return nil, fmt.Errorf("failed to read http response body: %w", readErr)
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Info("HTTP resource template request completed",
		zap.String("method", hi.Method),
		zap.String("url", url.(string)),
		zap.String("uri", req.Params.URI),
		zap.Int("status_code", response.StatusCode),
		zap.Int("response_length", len(body)))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Server-side only logging with sensitive URL details
		baseLogger.Error("HTTP resource template request failed with error status",
			zap.String("method", hi.Method),
			zap.String("url", url.(string)),
			zap.String("uri", req.Params.URI),
			zap.Int("status_code", response.StatusCode))
		// Log safe details to client
		logger.Error("HTTP resource template request failed with error status",
			zap.String("uri", req.Params.URI),
			zap.Int("status_code", response.StatusCode))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	logger.Info("HTTP resource template invocation completed successfully", zap.String("uri", req.Params.URI))

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: response.Header.Get(contentTypeHeader),
				Text:     string(body),
			},
		},
	}, nil
}

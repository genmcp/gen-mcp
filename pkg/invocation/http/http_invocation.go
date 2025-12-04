package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	neturl "net/url"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yosida95/uritemplate/v3"
	"go.uber.org/zap"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"github.com/genmcp/gen-mcp/pkg/template"
)

const contentTypeHeader = "Content-Type"

type HttpInvoker struct {
	ParsedTemplate  *template.ParsedTemplate            // Parsed template for the URL path
	HeaderTemplates map[string]*template.ParsedTemplate // Parsed templates for the headers
	Method          string                              // Http request method
	InputSchema     *jsonschema.Resolved                // InputSchema for the tool
	URITemplate     string                              // MCP URI template (for resource templates only)
}

var _ invocation.Invoker = &HttpInvoker{}

func (hi *HttpInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting HTTP tool invocation")

	hasBody := hi.Method != nethttp.MethodGet && hi.Method != nethttp.MethodDelete && hi.Method != nethttp.MethodHead

	// Extract incoming headers from request
	var incomingHeaders nethttp.Header
	if req.Extra != nil {
		incomingHeaders = req.Extra.Header
	}

	url, headers, parsed, err := hi.buildRequestComponents(ctx, req.Params.Arguments, !hasBody, incomingHeaders)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := hi.prepareRequestBody(parsed)
		if err != nil {
			logger.Error("Failed to marshal HTTP request body", zap.Error(err))
			return nil, fmt.Errorf("failed to prepare request body: %w", err)
		}
		reqBody = bytes.NewBuffer(bodyJson)
	}

	response, body, err := hi.executeHTTPRequest(ctx, hi.Method, url, reqBody, hasBody, headers, nil)
	if err != nil {
		return utils.McpTextError("HTTP request failed: %v", err), nil
	}

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

func (hi *HttpInvoker) InvokePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting HTTP prompt invocation")

	hasBody := hi.Method != nethttp.MethodGet && hi.Method != nethttp.MethodDelete && hi.Method != nethttp.MethodHead
	buildQuery := !hasBody && len(hi.ParsedTemplate.Variables) > 0

	args := req.Params.Arguments
	if args == nil {
		args = make(map[string]string)
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		logger.Error("Failed to marshal HTTP prompt request arguments", zap.Error(err))
		return nil, fmt.Errorf("failed to prepare prompt request: %w", err)
	}

	// Extract incoming headers from request
	var incomingHeaders nethttp.Header
	if req.Extra != nil {
		incomingHeaders = req.Extra.Header
	}

	url, headers, parsed, err := hi.buildRequestComponents(ctx, argsBytes, buildQuery, incomingHeaders)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if hasBody {
		bodyJson, err := hi.prepareRequestBody(parsed)
		if err != nil {
			logger.Error("Failed to marshal HTTP prompt request body", zap.Error(err))
			return nil, fmt.Errorf("failed to prepare request body: %w", err)
		}
		reqBody = bytes.NewBuffer(bodyJson)
	}

	response, body, err := hi.executeHTTPRequest(ctx, hi.Method, url, reqBody, hasBody, headers, nil)
	if err != nil {
		return utils.McpPromptTextError("HTTP request failed: %v", err), nil
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
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
	logger.Debug("Starting HTTP resource invocation", zap.String("uri", req.Params.URI))

	// For static resources, the template should have no variables
	// We can use the template directly as the URL
	url := hi.ParsedTemplate.Template

	// Build headers if any are configured
	var headers nethttp.Header
	if len(hi.HeaderTemplates) > 0 {
		// Extract incoming headers from request
		var incomingHeaders nethttp.Header
		if req.Extra != nil {
			incomingHeaders = req.Extra.Header
		}

		hb, err := newHeaderBuilder(hi.HeaderTemplates)
		if err != nil {
			logger.Error("Failed to create header builder", zap.String("uri", req.Params.URI), zap.Error(err))
			return nil, fmt.Errorf("failed to create header builder: %w", err)
		}

		if incomingHeaders != nil {
			headerResolver := template.NewHttpHeaderResolver(incomingHeaders)
			hb.SetSourceResolver("headers", headerResolver)
		}

		headersResult, err := hb.GetResult()
		if err != nil {
			logger.Error("Failed to build headers", zap.String("uri", req.Params.URI), zap.Error(err))
			return nil, fmt.Errorf("failed to build headers: %w", err)
		}
		headers = headersResult.(nethttp.Header)
	} else {
		headers = make(nethttp.Header)
	}

	response, body, err := hi.executeHTTPRequest(ctx, hi.Method, url, nil, false, headers, map[string]string{"uri": req.Params.URI})
	if err != nil {
		logger.Error("HTTP resource request execution failed", zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
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
	logger.Debug("Starting HTTP resource template invocation", zap.String("uri", req.Params.URI))

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

	// Extract incoming headers from request
	var incomingHeaders nethttp.Header
	if req.Extra != nil {
		incomingHeaders = req.Extra.Header
	}

	url, headers, _, err := hi.buildRequestComponents(ctx, argsBytes, true, incomingHeaders)
	if err != nil {
		return nil, err
	}

	response, body, err := hi.executeHTTPRequest(ctx, hi.Method, url, nil, false, headers, map[string]string{
		"uri":      req.Params.URI,
		"template": hi.URITemplate,
	})
	if err != nil {
		logger.Error("HTTP resource template request execution failed", zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
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

// executeHTTPRequest handles the common HTTP request/response cycle.
// It centralizes request creation, execution, response reading, and logging.
// Returns the response and body bytes. The response body has already been read and closed,
// so callers should use the returned []byte instead of accessing response.Body.
func (hi *HttpInvoker) executeHTTPRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
	hasBody bool,
	headers nethttp.Header,
	contextInfo map[string]string, // additional context for logging (e.g., "uri", "template")
) (*nethttp.Response, []byte, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx)

	// Build log fields with sensitive HTTP details
	logFields := []zap.Field{
		zap.String("method", method),
		zap.String("url", url),
	}
	if hasBody {
		logFields = append(logFields, zap.Bool("has_body", true))
	}
	for k, v := range contextInfo {
		logFields = append(logFields, zap.String(k, v))
	}

	baseLogger.Debug("Executing HTTP request", logFields...)

	httpReq, err := nethttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		baseLogger.Error("Failed to create HTTP request", append(logFields, zap.Error(err))...)
		logger.Error("Failed to create HTTP request", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header = headers

	if hasBody {
		httpReq.Header.Set(contentTypeHeader, "application/json; charset=UTF-8")
	}

	client := &nethttp.Client{}
	response, err := client.Do(httpReq)
	if err != nil {
		baseLogger.Error("HTTP request execution failed", append(logFields, zap.Error(err))...)
		logger.Error("HTTP request execution failed")
		return nil, nil, err
	}
	defer func() {
		if cerr := response.Body.Close(); cerr != nil {
			baseLogger.Warn("Failed to close HTTP response body", append(logFields, zap.Error(cerr))...)
		}
	}()

	responseBody, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		baseLogger.Error("Failed to read HTTP response body", append(logFields, zap.Error(readErr))...)
		logger.Error("Failed to read HTTP response body")
		return nil, nil, readErr
	}

	// Server-side only logging with sensitive HTTP details
	baseLogger.Info("HTTP request completed", append(logFields,
		zap.Int("status_code", response.StatusCode),
		zap.Int("response_length", len(responseBody)))...)

	return response, responseBody, nil
}

// prepareRequestBody creates a JSON body from the parsed arguments,
// excluding any variables that are used in the URL template.
func (hi *HttpInvoker) prepareRequestBody(parsed map[string]any) ([]byte, error) {
	varNames := make([]string, 0, len(hi.ParsedTemplate.Variables))
	for _, v := range hi.ParsedTemplate.Variables {
		varNames = append(varNames, v.Name)
	}

	return json.Marshal(deletePathsFromMap(parsed, varNames))
}

// buildRequestComponents builds the URL and headers from request arguments and incoming headers.
// It handles setting up source resolvers for both URL and header templates.
func (hi *HttpInvoker) buildRequestComponents(
	ctx context.Context,
	argsBytes []byte,
	buildQuery bool,
	incomingHeaders nethttp.Header,
) (string, nethttp.Header, map[string]any, error) {
	logger := logging.FromContext(ctx)

	// Create URL builder
	ub, err := hi.newUrlBuilder(buildQuery)
	if err != nil {
		logger.Error("Failed to create URL builder", zap.Error(err))
		return "", nil, nil, fmt.Errorf("failed to create URL builder: %w", err)
	}

	// Create header builder
	var hb *headerBuilder
	if len(hi.HeaderTemplates) > 0 {
		hb, err = newHeaderBuilder(hi.HeaderTemplates)
		if err != nil {
			logger.Error("Failed to create header builder", zap.Error(err))
			return "", nil, nil, fmt.Errorf("failed to create header builder: %w", err)
		}
	}

	// Set up source resolver for incoming headers if provided
	if incomingHeaders != nil {
		headerResolver := template.NewHttpHeaderResolver(incomingHeaders)
		ub.SetSourceResolver("headers", headerResolver)
		if hb != nil {
			hb.SetSourceResolver("headers", headerResolver)
		}
	}

	// Parse and validate arguments
	builders := []invocation.Builder{ub}
	if hb != nil {
		builders = append(builders, hb)
	}

	dj := &invocation.DynamicJson{
		Builders: builders,
	}

	parsed, err := dj.ParseJson(argsBytes, hi.InputSchema.Schema())
	if err != nil {
		logger.Error("Failed to parse request arguments", zap.Error(err))
		return "", nil, nil, fmt.Errorf("failed to parse request: %w", err)
	}

	if err := hi.InputSchema.Validate(parsed); err != nil {
		logger.Error("Failed to validate request arguments", zap.Error(err))
		return "", nil, nil, fmt.Errorf("failed to validate request: %w", err)
	}

	// Get results
	url, _ := ub.GetResult()

	var headers nethttp.Header
	if hb != nil {
		headersResult, err := hb.GetResult()
		if err != nil {
			logger.Error("Failed to build headers", zap.Error(err))
			return "", nil, nil, fmt.Errorf("failed to build headers: %w", err)
		}
		headers = headersResult.(nethttp.Header)
	} else {
		headers = make(nethttp.Header)
	}

	return url.(string), headers, parsed, nil
}

// newUrlBuilder creates a new urlBuilder from the parsed template.
// A new builder is created for each invocation to avoid sharing state.
func (hi *HttpInvoker) newUrlBuilder(buildQuery bool) (*urlBuilder, error) {
	// Create a new TemplateBuilder for this invocation
	templateBuilder, err := template.NewTemplateBuilder(hi.ParsedTemplate, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create template builder: %w", err)
	}

	// Create variable names set for routing
	templateVarNames := make(map[string]bool)
	for _, varName := range templateBuilder.VariableNames() {
		templateVarNames[varName] = true
	}

	ub := &urlBuilder{
		templateBuilder:  templateBuilder,
		templateVarNames: templateVarNames,
		buildQuery:       buildQuery,
	}

	if buildQuery {
		ub.queryParams = neturl.Values{}
	}

	return ub, nil
}

type urlBuilder struct {
	templateBuilder  *template.TemplateBuilder
	templateVarNames map[string]bool // Set of variable names the template cares about
	queryParams      neturl.Values
	buildQuery       bool
}

var _ invocation.Builder = &urlBuilder{}

func (ub *urlBuilder) SetField(path string, value any) {
	// If this is a variable that the template cares about, propagate to the template
	if ub.templateVarNames[path] {
		ub.templateBuilder.SetField(path, value)
		return
	}

	// Otherwise, if we're building query params, add it to query
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
	// Get the formatted URL from the template
	templateResult, err := ub.templateBuilder.GetResult()
	if err != nil {
		return nil, err
	}

	base := templateResult.(string)

	if ub.buildQuery {
		q := ub.queryParams.Encode()
		if q == "" {
			return base, nil
		}
		return base + "?" + q, nil
	}

	return base, nil
}

// SetSourceResolver sets the resolver for the URL template to access source data (e.g., headers).
func (ub *urlBuilder) SetSourceResolver(sourceName string, resolver template.SourceResolver) {
	ub.templateBuilder.SetSourceResolver(sourceName, resolver)
}

// headerBuilder manages templates for multiple HTTP headers.
// It routes field values to the appropriate header templates based on variable dependencies.
type headerBuilder struct {
	headers          map[string]*template.TemplateBuilder // Map of header name to its template builder
	headerVarIndices map[string][]string                  // Map of variable name to list of header names that need it
}

// newHeaderBuilder creates a headerBuilder from parsed header templates.
func newHeaderBuilder(headerTemplates map[string]*template.ParsedTemplate) (*headerBuilder, error) {
	headers := make(map[string]*template.TemplateBuilder, len(headerTemplates))
	headerVarIndices := make(map[string][]string)

	for headerName, pt := range headerTemplates {
		tb, err := template.NewTemplateBuilder(pt, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create template builder for header '%s': %w", headerName, err)
		}
		headers[headerName] = tb

		// Build index of which headers need which variables
		for _, varName := range tb.VariableNames() {
			headerVarIndices[varName] = append(headerVarIndices[varName], headerName)
		}
	}

	return &headerBuilder{
		headers:          headers,
		headerVarIndices: headerVarIndices,
	}, nil
}

var _ invocation.Builder = &headerBuilder{}

func (hb *headerBuilder) SetField(path string, value any) {
	headerNames, ok := hb.headerVarIndices[path]
	if !ok {
		return
	}

	// Set the field on all headers that need this variable
	for _, headerName := range headerNames {
		hb.headers[headerName].SetField(path, value)
	}
}

func (hb *headerBuilder) GetResult() (any, error) {
	result := make(nethttp.Header, len(hb.headers))

	for headerName, tb := range hb.headers {
		value, err := tb.GetResult()
		if err != nil {
			return nil, fmt.Errorf("failed to build header '%s': %w", headerName, err)
		}
		result.Set(headerName, value.(string))
	}

	return result, nil
}

// SetSourceResolver sets the resolver for all header templates using the specified source.
func (hb *headerBuilder) SetSourceResolver(sourceName string, resolver template.SourceResolver) {
	for _, tb := range hb.headers {
		tb.SetSourceResolver(sourceName, resolver)
	}
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

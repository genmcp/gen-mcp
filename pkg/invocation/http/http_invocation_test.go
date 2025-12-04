package http

import (
	"context"
	"encoding/json"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHttpInvoker creates an HttpInvoker for testing from a URL template
func testHttpInvoker(t *testing.T, urlTemplate string, headerTemplates map[string]string, schema *jsonschema.Resolved, method string, uriTemplate string) HttpInvoker {
	t.Helper()

	sources := template.CreateHeadersSourceFactory()

	parsedTemplate, err := template.ParseTemplate(urlTemplate, template.TemplateParserOptions{
		InputSchema: schema.Schema(),
		Sources:     sources,
	})
	require.NoError(t, err, "failed to parse URL template")

	parsedHeaders := make(map[string]*template.ParsedTemplate)
	for headerName, headerTemplate := range headerTemplates {
		pt, err := template.ParseTemplate(headerTemplate, template.TemplateParserOptions{
			InputSchema: schema.Schema(),
			Sources:     sources,
		})
		require.NoError(t, err, "failed to parse header template for '%s'", headerName)
		parsedHeaders[headerName] = pt
	}

	return HttpInvoker{
		ParsedTemplate:  parsedTemplate,
		HeaderTemplates: parsedHeaders,
		InputSchema:     schema,
		Method:          method,
		URITemplate:     uriTemplate,
	}
}

var (
	resolvedEmpty, _    = (&jsonschema.Schema{Type: invocation.JsonSchemaTypeObject}).Resolve(nil)
	resolvedWithPath, _ = (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"path": {
				Type: invocation.JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"part1": {Type: invocation.JsonSchemaTypeInteger},
					"part2": {Type: invocation.JsonSchemaTypeString},
				},
			},
			"limit":  {Type: invocation.JsonSchemaTypeInteger},
			"search": {Type: invocation.JsonSchemaTypeString},
		},
	}).Resolve(nil)
)

func TestHttpInvocation(t *testing.T) {
	tt := []struct {
		name              string
		responseCode      int
		responseBody      func() []byte
		urlTemplate       string
		headerTemplates   map[string]string
		schema            *jsonschema.Resolved
		method            string
		request           *mcp.CallToolRequest
		expectedResult    *mcp.CallToolResult
		expectedReqMethod string
		expectedPath      string
		expectedQuery     neturl.Values
		expectedBody      map[string]any
		expectedHeaders   nethttp.Header
		expectError       bool
	}{
		{
			name:         "simple GET request",
			responseCode: 200,
			responseBody: func() []byte { return []byte("hello, world!") },
			urlTemplate:  "/hello",
			schema:       resolvedEmpty,
			method:       "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "hello, world!",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/hello",
		},
		{
			name:         "GET request with multiple path params and query values",
			responseCode: 200,
			responseBody: func() []byte { return []byte("hello, world!") },
			urlTemplate:  "/hello/{path.part1}/{path.part2}",
			schema:       resolvedWithPath,
			method:       "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"path\": {\"part1\": 1, \"part2\": \"world\"}, \"limit\": 50, \"search\": \"hello\"}"),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "hello, world!",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery: map[string][]string{
				"limit":  {"50"},
				"search": {"hello"},
			},
			expectedPath: "/hello/1/world",
		},
		{
			name:         "POST request with multiple path params and body",
			responseCode: 200,
			responseBody: func() []byte { return []byte("hello, world!") },
			urlTemplate:  "/hello/{path.part1}/{path.part2}",
			schema:       resolvedWithPath,
			method:       "POST",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"path\": {\"part1\": 1, \"part2\": \"world\"}, \"limit\": 50, \"search\": \"hello\"}"),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "hello, world!",
					},
				},
			},
			expectedReqMethod: "POST",
			expectedQuery:     make(neturl.Values),
			expectedBody: map[string]any{
				"limit":  float64(50), // by default the json.Unmarshal we do in tests parses all numbers as f64
				"search": "hello",
			},
			expectedPath: "/hello/1/world",
		},
		{
			name:         "GET request with static header",
			responseCode: 200,
			responseBody: func() []byte { return []byte("authenticated!") },
			urlTemplate:  "/secure",
			headerTemplates: map[string]string{
				"Authorization": "Bearer secret-token",
				"X-Custom":      "static-value",
			},
			schema: resolvedEmpty,
			method: "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "authenticated!",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/secure",
			expectedHeaders: nethttp.Header{
				"Authorization": []string{"Bearer secret-token"},
				"X-Custom":      []string{"static-value"},
			},
		},
		{
			name:         "POST request with header from request params",
			responseCode: 200,
			responseBody: func() []byte { return []byte("created!") },
			urlTemplate:  "/items",
			headerTemplates: map[string]string{
				"X-User-Id": "{path.part1}",
			},
			schema: resolvedWithPath,
			method: "POST",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte(`{"path": {"part1": 42, "part2": "test"}}`),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "created!",
					},
				},
			},
			expectedReqMethod: "POST",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/items",
			expectedBody: map[string]any{
				"path": map[string]any{
					"part1": float64(42),
					"part2": "test",
				},
			},
			expectedHeaders: nethttp.Header{
				"X-User-Id": []string{"42"},
			},
		},
		{
			name:         "GET request with header from incoming request headers",
			responseCode: 200,
			responseBody: func() []byte { return []byte("proxied!") },
			urlTemplate:  "/proxy",
			headerTemplates: map[string]string{
				"X-Forwarded-Auth": "{headers.Authorization}",
				"X-Request-Id":     "{headers.X-Request-Id}",
			},
			schema: resolvedEmpty,
			method: "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
				Extra: &mcp.RequestExtra{
					Header: nethttp.Header{
						"Authorization": []string{"Bearer incoming-token"},
						"X-Request-Id":  []string{"req-123"},
					},
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "proxied!",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/proxy",
			expectedHeaders: nethttp.Header{
				"X-Forwarded-Auth": []string{"Bearer incoming-token"},
				"X-Request-Id":     []string{"req-123"},
			},
		},
		{
			name:         "GET request with header source in URL template",
			responseCode: 200,
			responseBody: func() []byte { return []byte("url from header!") },
			urlTemplate:  "/users/{headers.X-User-Name}",
			schema:       resolvedEmpty,
			method:       "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
				Extra: &mcp.RequestExtra{
					Header: nethttp.Header{
						"X-User-Name": []string{"alice"},
					},
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "url from header!",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/users/alice",
		},
		{
			name:         "GET request with query params and no template variables",
			responseCode: 200,
			responseBody: func() []byte { return []byte("search results") },
			urlTemplate:  "/search",
			schema:       resolvedWithPath,
			method:       "GET",
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte(`{"limit": 10, "search": "test query"}`),
				},
			},
			expectedResult: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "search results",
					},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery: map[string][]string{
				"limit":  {"10"},
				"search": {"test query"},
			},
			expectedPath: "/search",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedQuery neturl.Values
			var receivedBody map[string]any
			var receievedMethod string
			var receivedPath string
			var receivedHeaders nethttp.Header
			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receievedMethod = r.Method
				receivedQuery = r.URL.Query()
				receivedPath = r.URL.Path
				receivedHeaders = r.Header.Clone()
				if r.ContentLength > 0 {
					defer func() {
						_ = r.Body.Close()
					}()
					bodyBytes, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading request body should not fail")

					err = json.Unmarshal(bodyBytes, &receivedBody)
					assert.NoError(t, err, "unmarshalling request body should not fail")
				}

				w.WriteHeader(tc.responseCode)
				_, err := w.Write(tc.responseBody())
				assert.NoError(t, err, "writing response should not fail")
			}))
			defer s.Close()

			httpInvoker := testHttpInvoker(t, s.URL+tc.urlTemplate, tc.headerTemplates, tc.schema, tc.method, "")

			res, err := httpInvoker.Invoke(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors, expect Go error
				assert.Error(t, err, "http invocation should return Go error for validation/parsing failures")
				assert.Nil(t, res, "should not get result when there's a Go error")
			} else {
				// For successful operations and HTTP execution errors, expect MCP result
				assert.NoError(t, err, "http invocation should not return Go error")
				assert.NotNil(t, res, "should get a result")
			}

			assert.Equal(t, tc.expectedReqMethod, receievedMethod, "http invocation should use correct request method")
			assert.Equal(t, tc.expectedQuery, receivedQuery, "http url query should match")
			assert.Equal(t, tc.expectedBody, receivedBody, "http body should match")
			assert.Equal(t, tc.expectedResult, res, "mcp tool call result should match")
			assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")

			if tc.expectedHeaders != nil {
				for headerName, expectedValues := range tc.expectedHeaders {
					assert.Equal(t, expectedValues, receivedHeaders[headerName], "header %s should match", headerName)
				}
			}
		})
	}
}

func TestHttpPromptInvocation(t *testing.T) {
	tt := []struct {
		name              string
		responseCode      int
		responseBody      func() []byte
		urlTemplate       string
		schema            *jsonschema.Resolved
		method            string
		request           *mcp.GetPromptRequest
		expectedReqMethod string
		expectedPath      string
		expectedQuery     neturl.Values
		expectedBody      map[string]any
		expectError       bool
	}{
		{
			name:         "simple GET prompt request",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"messages":[{"role":"user","content":{"type":"text","text":"Analyze this data"}}]}`)
			},
			urlTemplate: "/prompts/analyze",
			schema:      resolvedEmpty,
			method:      "GET",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "analyze",
					Arguments: map[string]string{},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/prompts/analyze",
		},
		{
			name:         "POST prompt request with arguments",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"messages":[{"role":"user","content":{"text":"Analyze features for Q4 planning"}}]}`)
			},
			urlTemplate: "/prompts/feature-analysis",
			schema:      resolvedEmpty,
			method:      "POST",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "feature-analysis",
					Arguments: map[string]string{},
				},
			},
			expectedReqMethod: "POST",
			expectedQuery:     make(neturl.Values),
			expectedBody:      map[string]any{},
			expectedPath:      "/prompts/feature-analysis",
		},
		{
			name:         "GET prompt request with query params",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"messages":[{"role":"user","content":{"text":"Custom analysis prompt"}}]}`)
			},
			urlTemplate: "/prompts/custom",
			schema:      resolvedEmpty,
			method:      "GET",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "custom-prompt",
					Arguments: map[string]string{},
				},
			},
			expectedReqMethod: "GET",
			expectedQuery:     make(neturl.Values),
			expectedPath:      "/prompts/custom",
		},
		{
			name:         "PUT prompt request with body",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"messages":[{"role":"assistant","content":{"text":"Updated prompt response"}}]}`)
			},
			urlTemplate: "/prompts/update",
			schema:      resolvedEmpty,
			method:      "PUT",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "update-prompt",
					Arguments: map[string]string{},
				},
			},
			expectedReqMethod: "PUT",
			expectedQuery:     make(neturl.Values),
			expectedBody:      map[string]any{},
			expectedPath:      "/prompts/update",
		},
		{
			name:         "prompt request with HTTP error response",
			responseCode: 400,
			responseBody: func() []byte {
				return []byte(`{"error": "Invalid request"}`)
			},
			urlTemplate: "/prompts/error",
			schema:      resolvedEmpty,
			method:      "POST",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "error-prompt",
					Arguments: map[string]string{},
				},
			},
			expectError:       true,
			expectedReqMethod: "POST",
			expectedQuery:     make(neturl.Values),
			expectedBody:      map[string]any{},
			expectedPath:      "/prompts/error",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedQuery neturl.Values
			var receivedBody map[string]any
			var receivedMethod string
			var receivedPath string
			var receivedContentType string

			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receivedMethod = r.Method
				receivedQuery = r.URL.Query()
				receivedPath = r.URL.Path
				receivedContentType = r.Header.Get("Content-Type")

				if r.ContentLength > 0 {
					defer func() {
						_ = r.Body.Close()
					}()
					bodyBytes, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading request body should not fail")

					err = json.Unmarshal(bodyBytes, &receivedBody)
					assert.NoError(t, err, "unmarshalling request body should not fail")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseCode)
				_, err := w.Write(tc.responseBody())
				assert.NoError(t, err, "writing response should not fail")
			}))
			defer s.Close()

			httpInvoker := testHttpInvoker(t, s.URL+tc.urlTemplate, nil, tc.schema, tc.method, "")

			res, err := httpInvoker.InvokePrompt(context.Background(), tc.request)
			if tc.expectError {
				// For HTTP execution errors, expect MCP error result
				assert.NoError(t, err, "http prompt invocation should not return Go error for HTTP execution errors")
				assert.NotNil(t, res, "should get MCP error result")
				assert.NotEmpty(t, res.Description, "MCP error result should have error description")
			} else {
				assert.NoError(t, err, "http prompt invocation should not have an error")
				assert.NotNil(t, res, "should get a response")
				assert.NotNil(t, res.Messages, "should have messages")
				assert.Empty(t, res.Description, "successful result should not have error description")
			}

			assert.Equal(t, tc.expectedReqMethod, receivedMethod, "http invocation should use correct request method")
			assert.Equal(t, tc.expectedQuery, receivedQuery, "http url query should match")
			assert.Equal(t, tc.expectedBody, receivedBody, "http body should match")
			assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")

			// Verify Content-Type header is set for requests with body
			hasBody := tc.method != "GET" && tc.method != "DELETE" && tc.method != "HEAD"
			if hasBody {
				assert.Equal(t, "application/json; charset=UTF-8", receivedContentType, "Content-Type header should be set for requests with body")
			}
		})
	}
}

func TestHttpPromptInvocationErrors(t *testing.T) {
	tt := []struct {
		name        string
		urlTemplate string
		schema      *jsonschema.Resolved
		method      string
		request     *mcp.GetPromptRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid JSON in arguments",
			urlTemplate: "/prompts/test",
			schema:      resolvedEmpty,
			method:      "POST",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "test",
					// This will cause JSON marshal to fail when we try to convert to []byte
					Arguments: map[string]string{},
				},
			},
			expectError: false, // This actually won't error because map[string]string is valid JSON
		},
		{
			name:        "schema validation failure",
			urlTemplate: "/prompts/test",
			schema:      resolvedWithPath, // Requires specific structure
			method:      "POST",
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "test",
					Arguments: map[string]string{
						"invalid": "field", // Doesn't match required schema
					},
				},
			},
			expectError: true,
			errorMsg:    "extraneous field found in json",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"messages":[]}`))
			}))
			defer s.Close()

			httpInvoker := testHttpInvoker(t, s.URL+tc.urlTemplate, nil, tc.schema, tc.method, "")

			_, err := httpInvoker.InvokePrompt(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "should have error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "should not have error")
			}
		})
	}
}

func TestHttpResourceInvocation(t *testing.T) {
	tt := []struct {
		name              string
		responseCode      int
		responseBody      func() []byte
		contentType       string
		urlTemplate       string
		schema            *jsonschema.Resolved
		method            string
		request           *mcp.ReadResourceRequest
		expectedReqMethod string
		expectedPath      string
		expectError       bool
	}{
		{
			name:         "simple GET resource request",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte("Resource content here")
			},
			contentType: "text/plain",
			urlTemplate: "/resources/log.txt",
			schema:      resolvedEmpty,
			method:      "GET",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "file://log.txt",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/resources/log.txt",
		},
		{
			name:         "GET resource with JSON content",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"data": "value"}`)
			},
			contentType: "application/json",
			urlTemplate: "/resources/data.json",
			schema:      resolvedEmpty,
			method:      "GET",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "file://data.json",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/resources/data.json",
		},
		{
			name:         "resource request with HTTP error",
			responseCode: 404,
			responseBody: func() []byte {
				return []byte("Not found")
			},
			contentType: "text/plain",
			urlTemplate: "/resources/missing.txt",
			schema:      resolvedEmpty,
			method:      "GET",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "file://missing.txt",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/resources/missing.txt",
			expectError:       true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedMethod string
			var receivedPath string

			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receivedMethod = r.Method
				receivedPath = r.URL.Path

				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.responseCode)
				_, err := w.Write(tc.responseBody())
				assert.NoError(t, err, "writing response should not fail")
			}))
			defer s.Close()

			httpInvoker := testHttpInvoker(t, s.URL+tc.urlTemplate, nil, tc.schema, tc.method, "")

			res, err := httpInvoker.InvokeResource(context.Background(), tc.request)
			if tc.expectError {
				// For HTTP execution errors, expect Go error (ResourceNotFound)
				assert.Error(t, err, "http resource invocation should return Go error for HTTP execution errors")
				assert.Nil(t, res, "should not get result when there's a Go error")
			} else {
				assert.NoError(t, err, "http resource invocation should not have an error")
				assert.NotNil(t, res, "should get a response")
				assert.NotNil(t, res.Contents, "should have contents")
				assert.Equal(t, 1, len(res.Contents), "should have one content item")
				assert.Equal(t, tc.request.Params.URI, res.Contents[0].URI, "URI should match request")
				assert.Equal(t, tc.contentType, res.Contents[0].MIMEType, "MIME type should match response header")
				assert.Equal(t, string(tc.responseBody()), res.Contents[0].Text, "text content should match")
			}

			assert.Equal(t, tc.expectedReqMethod, receivedMethod, "http invocation should use correct request method")
			assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")
		})
	}
}

func TestHttpResourceTemplateInvocation(t *testing.T) {
	resolvedWithCityDate, _ := (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"city": {Type: invocation.JsonSchemaTypeString},
			"date": {Type: invocation.JsonSchemaTypeString},
		},
		Required: []string{"city", "date"},
	}).Resolve(nil)

	resolvedWithUserPost, _ := (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"username": {Type: invocation.JsonSchemaTypeString},
			"postId":   {Type: invocation.JsonSchemaTypeString},
		},
		Required: []string{"username", "postId"},
	}).Resolve(nil)

	tt := []struct {
		name              string
		responseCode      int
		responseBody      func() []byte
		contentType       string
		urlTemplate       string
		schema            *jsonschema.Resolved
		method            string
		uriTemplate       string
		request           *mcp.ReadResourceRequest
		expectedReqMethod string
		expectedPath      string
		expectedQuery     neturl.Values
		expectError       bool
		errorMsg          string
	}{
		{
			name:         "resource template with URI params as query",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"temperature": 72, "conditions": "sunny"}`)
			},
			contentType: "application/json",
			urlTemplate: "/weather",
			schema:      resolvedWithCityDate,
			method:      "GET",
			uriTemplate: "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London/2025-10-07",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/weather",
			expectedQuery: map[string][]string{
				"city": {"London"},
				"date": {"2025-10-07"},
			},
		},
		{
			name:         "resource template with path params in URL",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte(`{"user": "data"}`)
			},
			contentType: "application/json",
			urlTemplate: "/users/{username}/posts/{postId}",
			schema:      resolvedWithUserPost,
			method:      "GET",
			uriTemplate: "app://users/{username}/posts/{postId}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "app://users/alice/posts/123",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/users/alice/posts/123",
			expectedQuery:     make(neturl.Values),
		},
		{
			name:         "resource template with invalid URI",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte("")
			},
			contentType: "text/plain",
			urlTemplate: "/weather",
			schema:      resolvedWithCityDate,
			method:      "GET",
			uriTemplate: "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://other/path",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name:         "resource template with missing required param",
			responseCode: 200,
			responseBody: func() []byte {
				return []byte("")
			},
			contentType: "text/plain",
			urlTemplate: "/weather",
			schema:      resolvedWithCityDate,
			method:      "GET",
			uriTemplate: "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name:         "resource template with HTTP error response",
			responseCode: 500,
			responseBody: func() []byte {
				return []byte("Internal server error")
			},
			contentType: "text/plain",
			urlTemplate: "/weather",
			schema:      resolvedWithCityDate,
			method:      "GET",
			uriTemplate: "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London/2025-10-07",
				},
			},
			expectedReqMethod: "GET",
			expectedPath:      "/weather",
			expectError:       true, // HTTP execution errors return Go errors (ResourceNotFound)
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedMethod string
			var receivedPath string
			var receivedQuery neturl.Values

			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receivedMethod = r.Method
				receivedPath = r.URL.Path
				receivedQuery = r.URL.Query()

				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.responseCode)
				_, err := w.Write(tc.responseBody())
				assert.NoError(t, err, "writing response should not fail")
			}))
			defer s.Close()

			httpInvoker := testHttpInvoker(t, s.URL+tc.urlTemplate, nil, tc.schema, tc.method, tc.uriTemplate)

			res, err := httpInvoker.InvokeResourceTemplate(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors and HTTP execution errors, expect Go error
				assert.Error(t, err, "http resource template invocation should return Go error for failures")
				assert.Nil(t, res, "should not get result when there's a Go error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				// For successful operations
				assert.NoError(t, err, "http resource template invocation should not have an error")
				assert.NotNil(t, res, "should get a response")
				assert.NotNil(t, res.Contents, "should have contents")
				assert.Equal(t, 1, len(res.Contents), "should have one content item")
				assert.Equal(t, tc.request.Params.URI, res.Contents[0].URI, "URI should match request")
				assert.Equal(t, tc.contentType, res.Contents[0].MIMEType, "MIME type should match response header")
				assert.Equal(t, string(tc.responseBody()), res.Contents[0].Text, "text content should match")

				assert.Equal(t, tc.expectedReqMethod, receivedMethod, "http invocation should use correct request method")
				assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")
				assert.Equal(t, tc.expectedQuery, receivedQuery, "http url query should match")
			}
		})
	}
}

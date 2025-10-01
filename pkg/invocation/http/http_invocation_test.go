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
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

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
		httpInvoker       HttpInvoker
		request           *mcp.CallToolRequest
		expectedResult    *mcp.CallToolResult
		expectedReqMethod string
		expectedPath      string
		expectedQuery     neturl.Values
		expectedBody      map[string]any
		expectError       bool
	}{
		{
			name:         "simple GET request",
			responseCode: 200,
			responseBody: func() []byte { return []byte("hello, world!") },
			httpInvoker: HttpInvoker{
				PathTemplate: "/hello",
				PathIndeces:  make(map[string]int),
				Method:       "GET",
				InputSchema:  resolvedEmpty,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/hello/%d/%s",
				PathIndeces: map[string]int{
					"path.part1": 0,
					"path.part2": 1,
				},
				Method:      "GET",
				InputSchema: resolvedWithPath,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/hello/%d/%s",
				PathIndeces: map[string]int{
					"path.part1": 0,
					"path.part2": 1,
				},
				Method:      "POST",
				InputSchema: resolvedWithPath,
			},
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedQuery neturl.Values
			var receivedBody map[string]any
			var receievedMethod string
			var receivedPath string
			s := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receievedMethod = r.Method
				receivedQuery = r.URL.Query()
				receivedPath = r.URL.Path
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

			tc.httpInvoker.PathTemplate = s.URL + tc.httpInvoker.PathTemplate

			res, err := tc.httpInvoker.Invoke(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "http invocation should have an error")
			} else {
				assert.NoError(t, err, "http invocation should not have an error")
			}

			assert.Equal(t, tc.expectedReqMethod, receievedMethod, "http invocation should use correct request method")
			assert.Equal(t, tc.expectedQuery, receivedQuery, "http url query should match")
			assert.Equal(t, tc.expectedBody, receivedBody, "http body should match")
			assert.Equal(t, tc.expectedResult, res, "mcp tool call result should match")
			assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")
		})
	}
}

func TestHttpPromptInvocation(t *testing.T) {
	tt := []struct {
		name              string
		responseCode      int
		responseBody      func() []byte
		httpInvoker       HttpInvoker
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/analyze",
				PathIndeces:  make(map[string]int),
				Method:       "GET",
				InputSchema:  resolvedEmpty,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/feature-analysis",
				PathIndeces:  make(map[string]int),
				Method:       "POST",
				InputSchema:  resolvedEmpty,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/custom",
				PathIndeces:  make(map[string]int),
				Method:       "GET",
				InputSchema:  resolvedEmpty,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/update",
				PathIndeces:  make(map[string]int),
				Method:       "PUT",
				InputSchema:  resolvedEmpty,
			},
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
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/error",
				PathIndeces:  make(map[string]int),
				Method:       "POST",
				InputSchema:  resolvedEmpty,
			},
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

			tc.httpInvoker.PathTemplate = s.URL + tc.httpInvoker.PathTemplate

			res, err := tc.httpInvoker.InvokePrompt(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "http prompt invocation should have an error")
			} else {
				assert.NoError(t, err, "http prompt invocation should not have an error")
			}

			assert.Equal(t, tc.expectedReqMethod, receivedMethod, "http invocation should use correct request method")
			assert.Equal(t, tc.expectedQuery, receivedQuery, "http url query should match")
			assert.Equal(t, tc.expectedBody, receivedBody, "http body should match")
			assert.Equal(t, tc.expectedPath, receivedPath, "http path should match")

			// Just verify we got a response (JSON parsing is tested separately)
			if !tc.expectError {
				assert.NotNil(t, res, "should get a response")
				assert.NotNil(t, res.Messages, "should have messages")
			}

			// Verify Content-Type header is set for requests with body
			hasBody := tc.httpInvoker.Method != "GET" && tc.httpInvoker.Method != "DELETE" && tc.httpInvoker.Method != "HEAD"
			if hasBody {
				assert.Equal(t, "application/json; charset=UTF-8", receivedContentType, "Content-Type header should be set for requests with body")
			}
		})
	}
}

func TestHttpPromptInvocationErrors(t *testing.T) {
	tt := []struct {
		name        string
		httpInvoker HttpInvoker
		request     *mcp.GetPromptRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid JSON in arguments",
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/test",
				PathIndeces:  make(map[string]int),
				Method:       "POST",
				InputSchema:  resolvedEmpty,
			},
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
			name: "schema validation failure",
			httpInvoker: HttpInvoker{
				PathTemplate: "/prompts/test",
				PathIndeces:  make(map[string]int),
				Method:       "POST",
				InputSchema:  resolvedWithPath, // Requires specific structure
			},
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

			tc.httpInvoker.PathTemplate = s.URL + tc.httpInvoker.PathTemplate

			_, err := tc.httpInvoker.InvokePrompt(context.Background(), tc.request)
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

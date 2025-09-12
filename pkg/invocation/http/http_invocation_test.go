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

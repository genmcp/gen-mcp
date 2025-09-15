package http

import (
	"encoding/json"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestParser_Parse(t *testing.T) {
	tt := []struct {
		name           string
		inputData      json.RawMessage
		tool           *mcpfile.Tool
		expectedConfig *HttpInvocationConfig
		expectError    bool
	}{
		{
			name:      "simple URL without parameters",
			inputData: json.RawMessage(`{"url": "/api/users", "method": "get"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectedConfig: &HttpInvocationConfig{
				PathTemplate: "/api/users",
				PathIndices:  map[string]int{},
				Method:       "GET",
			},
			expectError: false,
		},
		{
			name:      "URL with path parameters",
			inputData: json.RawMessage(`{"url": "/api/users/{id}/posts/{postId}", "method": "post"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"id": {Type: invocation.JsonSchemaTypeInteger},
						"postId": {Type: invocation.JsonSchemaTypeString},
					},
				},
			},
			expectedConfig: &HttpInvocationConfig{
				PathTemplate: "/api/users/%d/posts/%s",
				PathIndices: map[string]int{
					"id":     0,
					"postId": 1,
				},
				Method: "POST",
			},
			expectError: false,
		},
		{
			name:      "invalid JSON",
			inputData: json.RawMessage(`{"url": "/api/users", "method": "get"`), // missing closing brace
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectedConfig: nil,
			expectError:    true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := &Parser{}
			config, err := parser.Parse(tc.inputData, tc.tool)

			if tc.expectError {
				assert.Error(t, err, "parser should return an error")
				assert.Nil(t, config, "config should be nil on error")
			} else {
				assert.NoError(t, err, "parser should not return an error")
				assert.Equal(t, tc.expectedConfig, config, "parsed config should match expected")
			}
		})
	}
}
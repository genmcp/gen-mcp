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
		name             string
		inputData        json.RawMessage
		tool             *mcpfile.Tool
		expectError      bool
		expectedURL      string
		expectedMethod   string
		expectedVarCount int
		expectedVarNames []string
	}{
		{
			name:      "simple URL without parameters",
			inputData: json.RawMessage(`{"url": "/api/users", "method": "get"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError:      false,
			expectedURL:      "/api/users",
			expectedMethod:   "GET",
			expectedVarCount: 0,
			expectedVarNames: []string{},
		},
		{
			name:      "URL with path parameters",
			inputData: json.RawMessage(`{"url": "/api/users/{id}/posts/{postId}", "method": "post"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"id":     {Type: invocation.JsonSchemaTypeInteger},
						"postId": {Type: invocation.JsonSchemaTypeString},
					},
				},
			},
			expectError:      false,
			expectedURL:      "/api/users/{id}/posts/{postId}",
			expectedMethod:   "POST",
			expectedVarCount: 2,
			expectedVarNames: []string{"id", "postId"},
		},
		{
			name:      "invalid JSON",
			inputData: json.RawMessage(`{"url": "/api/users", "method": "get"`), // missing closing brace
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError: true,
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
				assert.NotNil(t, config)
				hic := config.(*HttpInvocationConfig)
				assert.Equal(t, tc.expectedURL, hic.URL)
				assert.Equal(t, tc.expectedMethod, hic.Method)
				assert.NotNil(t, hic.ParsedTemplate)
				assert.Len(t, hic.ParsedTemplate.Variables, tc.expectedVarCount)
				for i, expectedName := range tc.expectedVarNames {
					assert.Equal(t, expectedName, hic.ParsedTemplate.Variables[i].Name)
				}
			}
		})
	}
}

func TestParser_ParseResource(t *testing.T) {
	tt := []struct {
		name             string
		inputData        json.RawMessage
		resource         *mcpfile.Resource
		expectError      bool
		errorContains    string
		expectedURL      string
		expectedMethod   string
		expectedVarCount int
	}{
		{
			name:      "valid static resource without path parameters",
			inputData: json.RawMessage(`{"url": "http://localhost:8080/status", "method": "get"}`),
			resource: &mcpfile.Resource{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError:      false,
			expectedURL:      "http://localhost:8080/status",
			expectedMethod:   "GET",
			expectedVarCount: 0,
		},
		{
			name:      "invalid static resource with path parameters",
			inputData: json.RawMessage(`{"url": "http://localhost:8080/users/{id}", "method": "get"}`),
			resource: &mcpfile.Resource{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"id": {Type: invocation.JsonSchemaTypeInteger},
					},
				},
			},
			expectError:   true,
			errorContains: "static resource URL cannot contain path parameters",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := &Parser{}
			config, err := parser.ParseResource(tc.inputData, tc.resource)

			if tc.expectError {
				assert.Error(t, err, "parser should return an error")
				assert.Nil(t, config, "config should be nil on error")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains, "error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "parser should not return an error")
				assert.NotNil(t, config)
				hic := config.(*HttpInvocationConfig)
				assert.Equal(t, tc.expectedURL, hic.URL)
				assert.Equal(t, tc.expectedMethod, hic.Method)
				assert.NotNil(t, hic.ParsedTemplate)
				assert.Len(t, hic.ParsedTemplate.Variables, tc.expectedVarCount)
			}
		})
	}
}

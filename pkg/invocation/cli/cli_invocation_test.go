package cli

import (
	"context"
	"net/http"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCliInvoker creates a CliInvoker for testing from a command template
func testCliInvoker(t *testing.T, commandTemplate string, schema *jsonschema.Resolved, uriTemplate string) CliInvoker {
	t.Helper()

	sources := template.CreateHeadersSourceFactory()

	parsedTemplate, err := template.ParseTemplate(commandTemplate, template.TemplateParserOptions{
		InputSchema: schema.Schema(),
		Sources:     sources,
	})
	require.NoError(t, err, "failed to parse command template")

	return CliInvoker{
		ParsedTemplate: parsedTemplate,
		InputSchema:    schema,
		URITemplate:    uriTemplate,
	}
}

var (
	resolvedEmpty, _      = (&jsonschema.Schema{Type: invocation.JsonSchemaTypeObject}).Resolve(nil)
	resolvedWithValues, _ = (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"all": {Type: invocation.JsonSchemaTypeBoolean},
		},
	}).Resolve(nil)
	resolvedWithPath, _ = (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"path":      {Type: invocation.JsonSchemaTypeString},
			"recursive": {Type: invocation.JsonSchemaTypeBoolean},
		},
	}).Resolve(nil)
	resolvedWithCount, _ = (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"lines": {Type: invocation.JsonSchemaTypeInteger},
			"file":  {Type: invocation.JsonSchemaTypeString},
		},
	}).Resolve(nil)
	resolvedWithAdditional, _ = (&jsonschema.Schema{
		Type:                 invocation.JsonSchemaTypeObject,
		AdditionalProperties: &jsonschema.Schema{Type: invocation.JsonSchemaTypeString},
	}).Resolve(nil)
)

func TestCliInvocation(t *testing.T) {
	tt := []struct {
		name            string
		commandTemplate string
		schema          *jsonschema.Resolved
		request         *mcp.CallToolRequest
		expectedResult  func(t *testing.T, result *mcp.CallToolResult)
		expectError     bool
	}{
		{
			name:            "simple echo command",
			commandTemplate: "echo 'hello, world!'",
			schema:          resolvedEmpty,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Equal(t, "hello, world!\n", textContent.Text)
			},
		},
		{
			name:            "ls command with path parameter",
			commandTemplate: "ls {path}",
			schema:          resolvedWithPath,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"path\": \"/tmp\"}"),
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				// Just check that we got some output (ls /tmp should return something)
				assert.NotEmpty(t, textContent.Text)
			},
		},
		{
			name:            "echo command with extra args",
			commandTemplate: "echo 'base message'",
			schema:          resolvedWithValues,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"all\": true}"),
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				// The command should be "echo 'base message' --all=true"
				assert.Contains(t, textContent.Text, "base message")
			},
		},
		{
			name:            "head command with multiple parameters",
			commandTemplate: "head -{lines} {file}",
			schema:          resolvedWithCount,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"lines\": 5, \"file\": \"/etc/passwd\"}"),
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				// Should get the first 5 lines of /etc/passwd
				assert.NotEmpty(t, textContent.Text)
			},
		},
		{
			name:            "invalid command should fail",
			commandTemplate: "nonexistentcommand12345",
			schema:          resolvedEmpty,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.True(t, result.IsError, "cli invocation should return MCP error result for execution failure")
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "Command execution failed")
			},
		},
		{
			name:            "command with header from incoming request headers",
			commandTemplate: "echo 'User: {headers.X-User-Name}, ID: {headers.X-Request-Id}'",
			schema:          resolvedEmpty,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"X-User-Name":  []string{"alice"},
						"X-Request-Id": []string{"req-123"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Equal(t, "User: alice, ID: req-123\n", textContent.Text)
			},
		},
		{
			name:            "command with header and path parameter",
			commandTemplate: "echo 'Path: {path}, Auth: {headers.Authorization}'",
			schema:          resolvedWithPath,
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{\"path\": \"/tmp\"}"),
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"Authorization": []string{"Bearer token-123"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.CallToolResult) {
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(*mcp.TextContent)
				assert.Equal(t, "Path: /tmp, Auth: Bearer token-123\n", textContent.Text)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invoker := testCliInvoker(t, tc.commandTemplate, tc.schema, "")
			res, err := invoker.Invoke(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors, expect Go error
				assert.Error(t, err, "cli invocation should return a Go error for validation/parsing failures")
				assert.Nil(t, res, "cli invocation should not return a result when there's a Go error")
			} else {
				// For successful executions and execution errors, expect MCP result
				assert.NoError(t, err, "cli invocation should not return a Go error")
				assert.NotNil(t, res, "cli invocation should return a result")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				} else {
					assert.False(t, res.IsError, "cli invocation should not have an error result")
				}
			}
		})
	}
}

func TestCliPromptInvocation(t *testing.T) {
	tt := []struct {
		name            string
		commandTemplate string
		schema          *jsonschema.Resolved
		request         *mcp.GetPromptRequest
		expectedResult  func(t *testing.T, result *mcp.GetPromptResult)
		expectError     bool
	}{
		{
			name:            "simple echo prompt",
			commandTemplate: "echo 'Generate analysis for prompt'",
			schema:          resolvedEmpty,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "analysis",
					Arguments: map[string]string{},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "Generate analysis for prompt\n", textContent.Text)
			},
		},
		{
			name:            "prompt with arguments",
			commandTemplate: "echo 'Analyzing path: {path}'",
			schema:          resolvedWithPath,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "path-analysis",
					Arguments: map[string]string{
						"path": "/tmp",
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "Analyzing path: /tmp\n", textContent.Text)
			},
		},
		{
			name:            "prompt with extra arguments",
			commandTemplate: "echo 'Base prompt'",
			schema:          resolvedWithAdditional,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "detailed-prompt",
					Arguments: map[string]string{
						"verbose": "true",
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "Base prompt")
				assert.Contains(t, textContent.Text, "--verbose=true")
			},
		},
		{
			name:            "prompt with date command",
			commandTemplate: "date '+Current time: %Y-%m-%d %H:%M:%S'",
			schema:          resolvedEmpty,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "timestamp",
					Arguments: map[string]string{},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "Current time:")
			},
		},
		{
			name:            "prompt with nil arguments",
			commandTemplate: "echo 'Prompt with no args'",
			schema:          resolvedEmpty,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "no-args",
					Arguments: nil,
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "Prompt with no args\n", textContent.Text)
			},
		},
		{
			name:            "prompt with validation error - missing required field",
			commandTemplate: "echo 'Analysis: {path}'",
			schema: func() *jsonschema.Resolved {
				schema := &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"path": {Type: invocation.JsonSchemaTypeString},
					},
					Required: []string{"path"}, // path is required
				}
				resolved, _ := schema.Resolve(nil)
				return resolved
			}(),
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "missing-field",
					Arguments: map[string]string{}, // Empty - missing required "path"
				},
			},
			expectError: true,
		},
		{
			name:            "prompt with multiple string arguments",
			commandTemplate: "echo 'User: {user}, Topic: {topic}'",
			schema: func() *jsonschema.Resolved {
				schema := &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"user":  {Type: invocation.JsonSchemaTypeString},
						"topic": {Type: invocation.JsonSchemaTypeString},
					},
				}
				resolved, _ := schema.Resolve(nil)
				return resolved
			}(),
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "multi-arg",
					Arguments: map[string]string{
						"user":  "mcp",
						"topic": "coding",
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "User: mcp, Topic: coding\n", textContent.Text)
			},
		},
		{
			name:            "invalid command should fail",
			commandTemplate: "nonexistentcommand54321",
			schema:          resolvedEmpty,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "invalid",
					Arguments: map[string]string{},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.NotEmpty(t, result.Description, "cli prompt invocation should have an error description for execution failure")
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Contains(t, textContent.Text, "Command execution failed")
			},
		},
		{
			name:            "prompt with header from incoming request headers",
			commandTemplate: "echo 'Auth: {headers.Authorization}'",
			schema:          resolvedEmpty,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "auth-prompt",
					Arguments: map[string]string{},
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"Authorization": []string{"Bearer prompt-token"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "Auth: Bearer prompt-token\n", textContent.Text)
			},
		},
		{
			name:            "prompt with header and argument",
			commandTemplate: "echo 'Path: {path}, User: {headers.X-User-Name}'",
			schema:          resolvedWithPath,
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "path-user-prompt",
					Arguments: map[string]string{
						"path": "/data",
					},
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"X-User-Name": []string{"bob"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Len(t, result.Messages, 1)
				assert.Equal(t, mcp.Role("assistant"), result.Messages[0].Role)
				textContent := result.Messages[0].Content.(*mcp.TextContent)
				assert.Equal(t, "Path: /data, User: bob\n", textContent.Text)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invoker := testCliInvoker(t, tc.commandTemplate, tc.schema, "")
			res, err := invoker.InvokePrompt(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors, expect Go error
				assert.Error(t, err, "cli prompt invocation should return a Go error for validation/parsing failures")
				assert.Nil(t, res, "cli prompt invocation should not return a result when there's a Go error")
			} else {
				// For successful executions and execution errors, expect MCP result
				assert.NoError(t, err, "cli prompt invocation should not return a Go error")
				assert.NotNil(t, res, "cli prompt invocation should return a result")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				} else {
					assert.Empty(t, res.Description, "cli prompt invocation should not have an error description")
				}
			}
		})
	}
}

func TestCliResourceInvocation(t *testing.T) {
	tt := []struct {
		name            string
		commandTemplate string
		schema          *jsonschema.Resolved
		request         *mcp.ReadResourceRequest
		expectedResult  func(t *testing.T, result *mcp.ReadResourceResult)
		expectError     bool
	}{
		{
			name:            "simple cat command",
			commandTemplate: "echo 'Resource content from command'",
			schema:          resolvedEmpty,
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "file://test.txt",
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "file://test.txt", result.Contents[0].URI)
				assert.Equal(t, "Resource content from command\n", result.Contents[0].Text)
			},
		},
		{
			name:            "date command as resource",
			commandTemplate: "date '+%Y-%m-%d'",
			schema:          resolvedEmpty,
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "system://date",
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "system://date", result.Contents[0].URI)
				assert.NotEmpty(t, result.Contents[0].Text)
			},
		},
		{
			name:            "invalid command should fail",
			commandTemplate: "nonexistentresourcecmd98765",
			schema:          resolvedEmpty,
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "file://missing.txt",
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invoker := testCliInvoker(t, tc.commandTemplate, tc.schema, "")
			res, err := invoker.InvokeResource(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors, expect Go error
				assert.Error(t, err, "cli resource invocation should return a Go error for validation/parsing failures")
				assert.Nil(t, res, "cli resource invocation should not return a result when there's a Go error")
			} else {
				// For successful executions and execution errors, expect MCP result
				assert.NoError(t, err, "cli resource invocation should not return a Go error")
				assert.NotNil(t, res, "cli resource invocation should return a result")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				}
			}
		})
	}
}

func TestCliResourceTemplateInvocation(t *testing.T) {
	resolvedWithCityDate, _ := (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"city": {Type: invocation.JsonSchemaTypeString},
			"date": {Type: invocation.JsonSchemaTypeString},
		},
		Required: []string{"city", "date"},
	}).Resolve(nil)

	resolvedWithUserFile, _ := (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"user": {Type: invocation.JsonSchemaTypeString},
			"file": {Type: invocation.JsonSchemaTypeString},
		},
		Required: []string{"user", "file"},
	}).Resolve(nil)

	tt := []struct {
		name            string
		commandTemplate string
		schema          *jsonschema.Resolved
		uriTemplate     string
		request         *mcp.ReadResourceRequest
		expectedResult  func(t *testing.T, result *mcp.ReadResourceResult)
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "resource template with URI params",
			commandTemplate: "echo 'Weather for {city} on {date}'",
			schema:          resolvedWithCityDate,
			uriTemplate:     "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London/2025-10-07",
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "weather://forecast/London/2025-10-07", result.Contents[0].URI)
				assert.Equal(t, "Weather for London on 2025-10-07\n", result.Contents[0].Text)
			},
		},
		{
			name:            "resource template with extra args",
			commandTemplate: "echo 'Base command'",
			schema:          resolvedWithUserFile,
			uriTemplate:     "app://users/{user}/files/{file}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "app://users/alice/files/doc.txt",
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "app://users/alice/files/doc.txt", result.Contents[0].URI)
				assert.Contains(t, result.Contents[0].Text, "Base command")
				assert.Contains(t, result.Contents[0].Text, "--user=alice")
				assert.Contains(t, result.Contents[0].Text, "--file=doc.txt")
			},
		},
		{
			name:            "resource template with invalid URI",
			commandTemplate: "echo 'Weather data'",
			schema:          resolvedWithCityDate,
			uriTemplate:     "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://other/path",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name:            "resource template with missing required param",
			commandTemplate: "echo 'Weather data'",
			schema:          resolvedWithCityDate,
			uriTemplate:     "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name:            "resource template with command failure",
			commandTemplate: "nonexistentcmdforresource456 {city}",
			schema: func() *jsonschema.Resolved {
				schema := &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"city": {Type: invocation.JsonSchemaTypeString},
					},
					Required: []string{"city"},
				}
				resolved, _ := schema.Resolve(nil)
				return resolved
			}(),
			uriTemplate: "weather://city/{city}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://city/Paris",
				},
			},
			expectError: true,
		},
		{
			name:            "resource template with header from incoming request headers",
			commandTemplate: "echo 'City: {city}, Auth: {headers.Authorization}'",
			schema: func() *jsonschema.Resolved {
				schema := &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"city": {Type: invocation.JsonSchemaTypeString},
					},
					Required: []string{"city"},
				}
				resolved, _ := schema.Resolve(nil)
				return resolved
			}(),
			uriTemplate: "weather://city/{city}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://city/Tokyo",
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"Authorization": []string{"Bearer resource-token"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "weather://city/Tokyo", result.Contents[0].URI)
				assert.Contains(t, result.Contents[0].Text, "City: Tokyo")
				assert.Contains(t, result.Contents[0].Text, "Auth: Bearer resource-token")
			},
		},
		{
			name:            "resource template with URI params and headers",
			commandTemplate: "echo 'City: {city}, Date: {date}, User: {headers.X-User-Name}'",
			schema:          resolvedWithCityDate,
			uriTemplate:     "weather://forecast/{city}/{date}",
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/Paris/2025-11-08",
				},
				Extra: &mcp.RequestExtra{
					Header: http.Header{
						"X-User-Name": []string{"charlie"},
					},
				},
			},
			expectedResult: func(t *testing.T, result *mcp.ReadResourceResult) {
				assert.NotNil(t, result)
				assert.Len(t, result.Contents, 1)
				assert.Equal(t, "weather://forecast/Paris/2025-11-08", result.Contents[0].URI)
				assert.Equal(t, "City: Paris, Date: 2025-11-08, User: charlie\n", result.Contents[0].Text)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invoker := testCliInvoker(t, tc.commandTemplate, tc.schema, tc.uriTemplate)
			res, err := invoker.InvokeResourceTemplate(context.Background(), tc.request)
			if tc.expectError {
				// For validation/parsing errors, expect Go error
				assert.Error(t, err, "cli resource template invocation should return a Go error for validation/parsing failures")
				assert.Nil(t, res, "cli resource template invocation should not return a result when there's a Go error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				// For successful executions and execution errors, expect MCP result
				assert.NoError(t, err, "cli resource template invocation should not return a Go error")
				assert.NotNil(t, res, "cli resource template invocation should return a result")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				}
			}
		})
	}
}

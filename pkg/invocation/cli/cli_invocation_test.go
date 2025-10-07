package cli

import (
	"context"
	"strconv"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// Test formatter implementations
type stringFormatter struct{}

func (f stringFormatter) FormatValue(v any) string { return v.(string) }

type intFormatter struct{}

func (f intFormatter) FormatValue(v any) string { return strconv.Itoa(v.(int)) }

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
		name           string
		cliInvoker     CliInvoker
		request        *mcp.CallToolRequest
		expectedResult func(t *testing.T, result *mcp.CallToolResult)
		expectError    bool
	}{
		{
			name: "simple echo command",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'hello, world!'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "ls command with path parameter",
			cliInvoker: CliInvoker{
				CommandTemplate: "ls %s",
				ArgumentIndices: map[string]int{
					"path": 0,
				},
				ArgumentFormatters: map[string]Formatter{
					"path": stringFormatter{},
				},
				InputSchema: resolvedWithPath,
			},
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
			name: "echo command with extra args",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'base message'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedWithValues,
			},
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
			name: "head command with multiple parameters",
			cliInvoker: CliInvoker{
				CommandTemplate: "head -%s %s",
				ArgumentIndices: map[string]int{
					"lines": 0,
					"file":  1,
				},
				ArgumentFormatters: map[string]Formatter{
					"lines": intFormatter{},
					"file":  stringFormatter{},
				},
				InputSchema: resolvedWithCount,
			},
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
			name: "invalid command should fail",
			cliInvoker: CliInvoker{
				CommandTemplate:    "nonexistentcommand12345",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
			request: &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Arguments: []byte("{}"),
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := tc.cliInvoker.Invoke(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "cli invocation should have an error")
			} else {
				assert.NoError(t, err, "cli invocation should not have an error")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				}
			}
		})
	}
}

func TestCliPromptInvocation(t *testing.T) {
	tt := []struct {
		name           string
		cliInvoker     CliInvoker
		request        *mcp.GetPromptRequest
		expectedResult func(t *testing.T, result *mcp.GetPromptResult)
		expectError    bool
	}{
		{
			name: "simple echo prompt",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'Generate analysis for prompt'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "prompt with arguments",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Analyzing path: %s'",
				ArgumentIndices: map[string]int{
					"path": 0,
				},
				ArgumentFormatters: map[string]Formatter{
					"path": stringFormatter{},
				},
				InputSchema: resolvedWithPath,
			},
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
			name: "prompt with extra arguments",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'Base prompt'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedWithAdditional,
			},
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
			name: "prompt with date command",
			cliInvoker: CliInvoker{
				CommandTemplate:    "date '+Current time: %Y-%m-%d %H:%M:%S'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "prompt with nil arguments",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'Prompt with no args'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "prompt with validation error - missing required field",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Analysis: %s'",
				ArgumentIndices: map[string]int{
					"path": 0,
				},
				ArgumentFormatters: map[string]Formatter{
					"path": stringFormatter{},
				},
				InputSchema: func() *jsonschema.Resolved {
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
			},
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "missing-field",
					Arguments: map[string]string{}, // Empty - missing required "path"
				},
			},
			expectError: true,
		},
		{
			name: "prompt with multiple string arguments",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'User: %s, Topic: %s'",
				ArgumentIndices: map[string]int{
					"user":  0,
					"topic": 1,
				},
				ArgumentFormatters: map[string]Formatter{
					"user":  stringFormatter{},
					"topic": stringFormatter{},
				},
				InputSchema: func() *jsonschema.Resolved {
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
			},
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
			name: "invalid command should fail",
			cliInvoker: CliInvoker{
				CommandTemplate:    "nonexistentcommand54321",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
			request: &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "invalid",
					Arguments: map[string]string{},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := tc.cliInvoker.InvokePrompt(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "cli prompt invocation should have an error")
			} else {
				assert.NoError(t, err, "cli prompt invocation should not have an error")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				}
			}
		})
	}
}

func TestCliResourceInvocation(t *testing.T) {
	tt := []struct {
		name           string
		cliInvoker     CliInvoker
		request        *mcp.ReadResourceRequest
		expectedResult func(t *testing.T, result *mcp.ReadResourceResult)
		expectError    bool
	}{
		{
			name: "simple cat command",
			cliInvoker: CliInvoker{
				CommandTemplate:    "echo 'Resource content from command'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "date command as resource",
			cliInvoker: CliInvoker{
				CommandTemplate:    "date '+%Y-%m-%d'",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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
			name: "invalid command should fail",
			cliInvoker: CliInvoker{
				CommandTemplate:    "nonexistentresourcecmd98765",
				ArgumentIndices:    make(map[string]int),
				ArgumentFormatters: make(map[string]Formatter),
				InputSchema:        resolvedEmpty,
			},
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

			res, err := tc.cliInvoker.InvokeResource(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "cli resource invocation should have an error")
			} else {
				assert.NoError(t, err, "cli resource invocation should not have an error")
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
		name           string
		cliInvoker     CliInvoker
		request        *mcp.ReadResourceRequest
		expectedResult func(t *testing.T, result *mcp.ReadResourceResult)
		expectError    bool
		errorMsg       string
	}{
		{
			name: "resource template with URI params",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Weather for %s on %s'",
				ArgumentIndices: map[string]int{
					"city": 0,
					"date": 1,
				},
				ArgumentFormatters: map[string]Formatter{
					"city": stringFormatter{},
					"date": stringFormatter{},
				},
				InputSchema: resolvedWithCityDate,
				URITemplate: "weather://forecast/{city}/{date}",
			},
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
			name: "resource template with extra args",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Base command'",
				ArgumentIndices: make(map[string]int),
				ArgumentFormatters: map[string]Formatter{
					"user": stringFormatter{},
					"file": stringFormatter{},
				},
				InputSchema: resolvedWithUserFile,
				URITemplate: "app://users/{user}/files/{file}",
			},
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
			name: "resource template with invalid URI",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Weather data'",
				ArgumentIndices: make(map[string]int),
				ArgumentFormatters: map[string]Formatter{
					"city": stringFormatter{},
					"date": stringFormatter{},
				},
				InputSchema: resolvedWithCityDate,
				URITemplate: "weather://forecast/{city}/{date}",
			},
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://other/path",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name: "resource template with missing required param",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Weather data'",
				ArgumentIndices: make(map[string]int),
				ArgumentFormatters: map[string]Formatter{
					"city": stringFormatter{},
					"date": stringFormatter{},
				},
				InputSchema: resolvedWithCityDate,
				URITemplate: "weather://forecast/{city}/{date}",
			},
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://forecast/London",
				},
			},
			expectError: true,
			errorMsg:    "does not match template",
		},
		{
			name: "resource template with empty URI template",
			cliInvoker: CliInvoker{
				CommandTemplate: "echo 'Data'",
				ArgumentIndices: make(map[string]int),
				ArgumentFormatters: map[string]Formatter{
					"city": stringFormatter{},
				},
				InputSchema: resolvedWithCityDate,
				URITemplate: "",
			},
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "app://data",
				},
			},
			expectError: true,
			errorMsg:    "URI template not configured",
		},
		{
			name: "resource template with command failure",
			cliInvoker: CliInvoker{
				CommandTemplate: "nonexistentcmdforresource456 %s",
				ArgumentIndices: map[string]int{
					"city": 0,
				},
				ArgumentFormatters: map[string]Formatter{
					"city": stringFormatter{},
				},
				InputSchema: func() *jsonschema.Resolved {
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
				URITemplate: "weather://city/{city}",
			},
			request: &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "weather://city/Paris",
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := tc.cliInvoker.InvokeResourceTemplate(context.Background(), tc.request)
			if tc.expectError {
				assert.Error(t, err, "cli resource template invocation should have an error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "cli resource template invocation should not have an error")
				if tc.expectedResult != nil {
					tc.expectedResult(t, res)
				}
			}
		})
	}
}

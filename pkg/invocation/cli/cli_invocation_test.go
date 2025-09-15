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
			"path": {Type: invocation.JsonSchemaTypeString},
			"recursive": {Type: invocation.JsonSchemaTypeBoolean},
		},
	}).Resolve(nil)
	resolvedWithCount, _ = (&jsonschema.Schema{
		Type: invocation.JsonSchemaTypeObject,
		Properties: map[string]*jsonschema.Schema{
			"lines": {Type: invocation.JsonSchemaTypeInteger},
			"file": {Type: invocation.JsonSchemaTypeString},
		},
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

package cli

import (
	"encoding/json"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	tt := []struct {
		name        string
		inputData   json.RawMessage
		tool        *mcpfile.Tool
		expectError bool
		validate    func(t *testing.T, config *CliInvocationConfig)
	}{
		{
			name:      "simple command without parameters",
			inputData: json.RawMessage(`{"command": "echo hello"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError: false,
			validate: func(t *testing.T, config *CliInvocationConfig) {
				assert.Equal(t, "echo hello", config.Command)
				require.NotNil(t, config.ParsedTemplate)
				assert.Equal(t, "echo hello", config.ParsedTemplate.Template)
				assert.Empty(t, config.ParsedTemplate.Variables)
			},
		},
		{
			name:      "command with parameters",
			inputData: json.RawMessage(`{"command": "echo {message} {count}"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"message": {Type: invocation.JsonSchemaTypeString},
						"count":   {Type: invocation.JsonSchemaTypeInteger},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, config *CliInvocationConfig) {
				assert.Equal(t, "echo {message} {count}", config.Command)
				require.NotNil(t, config.ParsedTemplate)
				assert.Len(t, config.ParsedTemplate.Variables, 2)
				assert.Equal(t, "message", config.ParsedTemplate.Variables[0].Name)
				assert.Equal(t, "count", config.ParsedTemplate.Variables[1].Name)
			},
		},
		{
			name:      "invalid JSON",
			inputData: json.RawMessage(`{"command": "echo hello"`), // missing closing brace
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError: true,
		},
		{
			name:      "command with template variables",
			inputData: json.RawMessage(`{"command": "curl {url}", "templateVariables": {"verbose": {"format": "--verbose"}}}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"url":     {Type: invocation.JsonSchemaTypeString},
						"verbose": {Type: invocation.JsonSchemaTypeBoolean},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, config *CliInvocationConfig) {
				assert.Equal(t, "curl {url}", config.Command)
				require.NotNil(t, config.ParsedTemplate)
				assert.Len(t, config.ParsedTemplate.Variables, 1)
				assert.Equal(t, "url", config.ParsedTemplate.Variables[0].Name)
				assert.Contains(t, config.TemplateVariables, "verbose")
				assert.Equal(t, "--verbose", config.TemplateVariables["verbose"].Template)
			},
		},
		{
			name:      "command with formatted template variables",
			inputData: json.RawMessage(`{"command": "wget {url}", "templateVariables": {"output": {"format": "-O {filename}"}}}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"url":      {Type: invocation.JsonSchemaTypeString},
						"filename": {Type: invocation.JsonSchemaTypeString},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, config *CliInvocationConfig) {
				assert.Equal(t, "wget {url}", config.Command)
				require.NotNil(t, config.ParsedTemplate)
				assert.Len(t, config.ParsedTemplate.Variables, 1)
				assert.Equal(t, "url", config.ParsedTemplate.Variables[0].Name)
				assert.Contains(t, config.TemplateVariables, "output")
				assert.Equal(t, "-O {filename}", config.TemplateVariables["output"].Template)
			},
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
				require.NotNil(t, config)
				cliConfig := config.(*CliInvocationConfig)
				if tc.validate != nil {
					tc.validate(t, cliConfig)
				}
			}
		})
	}
}

func TestParser_ParseResource(t *testing.T) {
	tt := []struct {
		name          string
		inputData     json.RawMessage
		resource      *mcpfile.Resource
		expectError   bool
		errorContains string
		validate      func(t *testing.T, config *CliInvocationConfig)
	}{
		{
			name:      "valid static resource without parameters",
			inputData: json.RawMessage(`{"command": "cat /var/log/app.log"}`),
			resource: &mcpfile.Resource{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectError: false,
			validate: func(t *testing.T, config *CliInvocationConfig) {
				assert.Equal(t, "cat /var/log/app.log", config.Command)
				require.NotNil(t, config.ParsedTemplate)
				assert.Empty(t, config.ParsedTemplate.Variables)
			},
		},
		{
			name:      "invalid static resource with template variables",
			inputData: json.RawMessage(`{"command": "cat /var/log/{filename}"}`),
			resource: &mcpfile.Resource{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
					Properties: map[string]*jsonschema.Schema{
						"filename": {Type: invocation.JsonSchemaTypeString},
					},
				},
			},
			expectError:   true,
			errorContains: "static resource command cannot contain template variables",
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
				require.NotNil(t, config)
				cliConfig := config.(*CliInvocationConfig)
				if tc.validate != nil {
					tc.validate(t, cliConfig)
				}
			}
		})
	}
}

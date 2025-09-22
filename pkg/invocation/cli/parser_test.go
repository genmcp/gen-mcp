package cli

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
		expectedConfig *CliInvocationConfig
		expectError    bool
	}{
		{
			name:      "simple command without parameters",
			inputData: json.RawMessage(`{"command": "echo hello"}`),
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectedConfig: &CliInvocationConfig{
				Command:          "echo hello",
				ParameterIndices: map[string]int{},
			},
			expectError: false,
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
			expectedConfig: &CliInvocationConfig{
				Command: "echo %s %s",
				ParameterIndices: map[string]int{
					"message": 0,
					"count":   1,
				},
			},
			expectError: false,
		},
		{
			name:      "invalid JSON",
			inputData: json.RawMessage(`{"command": "echo hello"`), // missing closing brace
			tool: &mcpfile.Tool{
				InputSchema: &jsonschema.Schema{
					Type: invocation.JsonSchemaTypeObject,
				},
			},
			expectedConfig: nil,
			expectError:    true,
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
			expectedConfig: &CliInvocationConfig{
				Command: "curl %s",
				ParameterIndices: map[string]int{
					"url": 0,
				},
				TemplateVariables: map[string]*TemplateVariable{
					"verbose": {
						Template:     "--verbose",
						shouldFormat: false,
					},
				},
			},
			expectError: false,
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
			expectedConfig: &CliInvocationConfig{
				Command: "wget %s",
				ParameterIndices: map[string]int{
					"url": 0,
				},
				TemplateVariables: map[string]*TemplateVariable{
					"output": {
						Template:     "-O %s",
						shouldFormat: true,
					},
				},
			},
			expectError: false,
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
				cliConfig := config.(*CliInvocationConfig)
				assert.Equal(t, tc.expectedConfig.Command, cliConfig.Command, "command should match expected")
				assert.Equal(t, tc.expectedConfig.ParameterIndices, cliConfig.ParameterIndices, "parameter indices should match expected")

				if tc.expectedConfig.TemplateVariables != nil {
					assert.Equal(t, len(tc.expectedConfig.TemplateVariables), len(cliConfig.TemplateVariables), "template variables count should match")
					for key, expectedTV := range tc.expectedConfig.TemplateVariables {
						actualTV := cliConfig.TemplateVariables[key]
						assert.NotNil(t, actualTV, "template variable should exist")
						assert.Equal(t, expectedTV.Template, actualTV.Template, "template should match")
						assert.Equal(t, expectedTV.shouldFormat, actualTV.shouldFormat, "shouldFormat should match")
					}
				}
			}
		})
	}
}

package cli

import (
	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
)

// The structure for CLI invocation configuration.
type CliInvocationData struct {
	// Detailed CLI invocation configuration.
	Http CliInvocationConfig `json:"cli" jsonschema:"required"`
}

// Configuration for executing a command-line tool.
type CliInvocationConfig struct {
	// The command-line string to be executed. It can contain placeholders in the form of '{paramName}' which correspond to parameters defined in the input schema.
	Command string `json:"command" jsonschema:"required"`

	// Defines how input parameters are formatted into the command string.
	// The map key corresponds to the parameter name from the input schema.
	TemplateVariables map[string]*TemplateVariable `json:"templateVariables,omitempty" jsonschema:"optional"`

	// ParsedTemplate contains the parsed command template with variables
	// This field is for internal use and is not part of the JSON schema.
	ParsedTemplate *template.ParsedTemplate `json:"-"`

	// MCP URI template (for resource templates only)
	URITemplate string `json:"-"`
}

var _ invocation.InvocationConfig = &CliInvocationConfig{}

func (c *CliInvocationConfig) Validate() error {
	// Validation is handled during template parsing
	return nil
}

// The formatting for a single parameter in the command template
type TemplateVariable struct {
	// Template is the format string for the variable. It can be a simple string or contain template variables like '{paramName}'.
	// For example, "--user={username}" or "--verbose".
	Template string `json:"format" jsonschema:"required"`

	// OmitIfFalse, if true, causes the template to be omitted entirely if the input
	// value is a boolean `false`. This is useful for optional flags like "--force".
	OmitIfFalse bool `json:"omitIfFalse,omitempty" jsonschema:"optional"`
}

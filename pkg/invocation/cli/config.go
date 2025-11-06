package cli

import (
	"maps"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// Configuration for executing a command-line tool.
// This is a pure data structure with no parsing logic - all struct tags only.
type CliInvocationConfig struct {
	// The command-line string to be executed. It can contain placeholders in the form of '{paramName}' which correspond to parameters defined in the input schema.
	Command string `json:"command" jsonschema:"required"`

	// Defines how input parameters are formatted into the command string.
	// The map key corresponds to the parameter name from the input schema.
	TemplateVariables map[string]*TemplateVariable `json:"templateVariables,omitempty" jsonschema:"optional"`
}

var _ invocation.InvocationConfig = &CliInvocationConfig{}

func (c *CliInvocationConfig) Validate() error {
	// Validation is handled during template parsing
	return nil
}

func (c *CliInvocationConfig) DeepCopy() invocation.InvocationConfig {
	cp := &CliInvocationConfig{
		Command:           c.Command,
		TemplateVariables: make(map[string]*TemplateVariable, len(c.TemplateVariables)),
	}
	maps.Copy(cp.TemplateVariables, c.TemplateVariables)

	return cp
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

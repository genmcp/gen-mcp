package cli

import (
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// The structure for CLI invocation configuration.
type CliInvocationData struct {
	// Detailed CLI invocation configuration.
	Http CliInvocationConfig `json:"cli" jsonschema:"required"`
}

// Configuration for executing a command-line tool.
type CliInvocationConfig struct {
	// The command-line string to be executed. It can contain placeholders in the form of '%' which correspond to parameters defined in the input schema.
	Command string `json:"command" jsonschema:"required"`

	// Defines how input parameters are formatted into the command string.
	// The map key corresponds to the parameter name from the input schema.
	TemplateVariables map[string]*TemplateVariable `json:"templateVariables,omitempty" jsonschema:"optional"`

	// ParameterIndices maps parameter names to their positional index in the command template.
	// This field is for internal use and is not part of the JSON schema.
	ParameterIndices map[string]int `json:"-"`

	URITemplate       string                       `json:"-"` // MCP URI template (for resource templates only)
}

var _ invocation.InvocationConfig = &CliInvocationConfig{}

func (c *CliInvocationConfig) Validate() error {
	validPathIndicesCount := strings.Count(c.Command, "%") == len(c.ParameterIndices)
	if !validPathIndicesCount {
		return fmt.Errorf("parameter indices do not match the number of template variables in the command template. expected %d, received %d", len(c.ParameterIndices), strings.Count(c.Command, "%"))
	}
	return nil
}

// The formatting for a single parameter in the command template
type TemplateVariable struct {
	// Template is the format string for the variable. It can be a simple string or a Go format string containing a verb like '%s', '%d', etc.
	// For example, "--user=%s" or "--verbose".
	Template string `json:"format" jsonschema:"required"`

	// OmitIfFalse, if true, causes the template to be omitted entirely if the input
	// value is a boolean `false`. This is useful for optional flags like "--force".
	OmitIfFalse bool `json:"omitIfFalse,omitempty" jsonschema:"optional"`

	// shouldFormat is an internal flag to indicate if the template contains a formatting verb.
	shouldFormat bool
}

func (tv *TemplateVariable) FormatValue(value any) string {
	if value == nil {
		// Input is nil, so we should omit the template variable
		return ""
	}

	if tv.OmitIfFalse {
		b, ok := value.(bool)
		if ok && !b {
			return ""
		}
	}

	if tv.shouldFormat {
		return fmt.Sprintf(tv.Template, value)
	}

	return tv.Template
}

var _ Formatter = &TemplateVariable{}

package cli

import (
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

type CliInvocationConfig struct {
	Command           string                       `json:"command"`
	TemplateVariables map[string]*TemplateVariable `json:"templateVariables,omitempty"`
	ParameterIndices  map[string]int               `json:"-"`
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

type TemplateVariable struct {
	Template     string `json:"format"`
	OmitIfFalse  bool   `json:"omitIfFalse,omitempty"`
	shouldFormat bool
}

func (tv *TemplateVariable) FormatValue(value any) string {
	if value == nil {
		// Input is nil, so we should omit the template variable
		return ""
	}

	if tv.OmitIfFalse {
		b := value.(bool)
		if !b {
			return ""
		}
	}

	if tv.shouldFormat {
		return fmt.Sprintf(tv.Template, value)
	}

	return tv.Template
}

var _ Formatter = &TemplateVariable{}

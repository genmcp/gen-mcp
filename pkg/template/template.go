package template

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/google/jsonschema-go/jsonschema"
)

type VariableType int

const (
	VariableTypeParam VariableType = iota
	VariableTypeEnv
)

// VariableFormatter formats template variables into their final values.
// It implements a builder pattern where values are set via SetField and
// the final result is retrieved via GetResult.
type VariableFormatter interface {
	SetField(path string, value any)
	GetResult() (any, error)
	FormatString() string
	VariableNames() []string // Returns the list of variable names this formatter needs
}

type Variable struct {
	VariableFormatter
	Name  string // the variable name (e.g. "userId", or "env.API_KEY")
	Type  VariableType
	Index int // Position in the parameter list
}

type ParsedTemplate struct {
	Template        string           // Final template with format specifiers
	Variables       []Variable       // Ordered list of variables - order comes from parse order
	VariableIndices map[string][]int // Map from variable name to all indices where it appears
}

type TemplateParserOptions struct {
	InputSchema *jsonschema.Schema           // used to validate parameters and determine their type
	Formatters  map[string]VariableFormatter // used to specify specific formatting options for specific variables
}

// escapePercent escapes literal % characters in template chunks by replacing % with %%
// so they don't interfere with fmt.Sprintf format verbs.
func escapePercent(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}

func ParseTemplate(template string, opts TemplateParserOptions) (*ParsedTemplate, error) {
	variables := make([]Variable, 0)
	variableIndices := make(map[string][]int)
	paramIdx := 0
	var chunks []string
	var chunk strings.Builder

	for i := 0; i < len(template); {
		// Handle ${VAR} syntax for environment variables
		if i+1 < len(template) && template[i] == '$' && template[i+1] == '{' {
			start := i + 2

			offset := strings.Index(template[start:], "}") + start
			if offset-start == -1 {
				return nil, fmt.Errorf("unterminated environment variable at position %d", i)
			}

			varName := template[start:offset]
			variable, err := createEnvVariable(varName, paramIdx)
			if err != nil {
				return nil, err
			}

			variables = append(variables, *variable)
			variableIndices[variable.Name] = append(variableIndices[variable.Name], paramIdx)
			chunks = append(chunks, escapePercent(chunk.String()), variable.FormatString())
			chunk.Reset()
			paramIdx++
			i = offset + 1
			continue
		}

		// handle {paramName} or {env.VAR} syntax
		if template[i] == '{' {
			start := i + 1

			offset := strings.Index(template[start:], "}") + start
			if offset-start == -1 {
				return nil, fmt.Errorf("unterminated variable at position %d", i)
			}

			varName := template[start:offset]

			var variable *Variable
			var err error
			if envVarName, found := strings.CutPrefix(varName, "env."); found {
				variable, err = createEnvVariable(envVarName, paramIdx)
			} else {
				variable, err = createSchemaVariable(varName, paramIdx, opts)
			}
			if err != nil {
				return nil, err
			}

			variables = append(variables, *variable)
			variableIndices[variable.Name] = append(variableIndices[variable.Name], paramIdx)
			chunks = append(chunks, escapePercent(chunk.String()), variable.FormatString())
			chunk.Reset()
			paramIdx++
			i = offset + 1
			continue
		}

		// handle unmatched closing bracket
		if template[i] == '}' {
			return nil, fmt.Errorf("unmatched closing bracket at position %d", i)
		}

		chunk.WriteByte(template[i])
		i++
	}

	chunks = append(chunks, escapePercent(chunk.String()))

	return &ParsedTemplate{
		Template:        strings.Join(chunks, ""),
		Variables:       variables,
		VariableIndices: variableIndices,
	}, nil
}

// TemplateBuilder builds the final template string by collecting values
// for all variables and rendering them into the template.
type TemplateBuilder struct {
	template          string
	formatters        []VariableFormatter
	indices           map[string][]int
	omitIfFalse       bool
	implicitFormatter *paramFormatter // Used when omitIfFalse=true with 0 variables
}

// NewTemplateBuilder creates a new builder from a parsed template.
func NewTemplateBuilder(pt *ParsedTemplate, omitIfFalse bool) (*TemplateBuilder, error) {
	formatters := make([]VariableFormatter, len(pt.Variables))
	for i, v := range pt.Variables {
		formatters[i] = v.VariableFormatter
	}

	var implicitFormatter *paramFormatter
	if omitIfFalse {
		if len(pt.Variables) > 1 {
			return nil, fmt.Errorf("omitIfFalse can only be used with <= 1 variables")
		}
		if len(pt.Variables) == 1 {
			if _, ok := formatters[0].(*paramFormatter); !ok {
				return nil, fmt.Errorf("omitIfFalse requires a parameter formatter, got %T", formatters[0])
			}
		}
		if len(pt.Variables) == 0 {
			// Create an implicit formatter that accepts any field name
			implicitFormatter = &paramFormatter{
				formatString: "%v",
			}
		}
	}

	// Start with a copy of the parsed template's indices
	indices := make(map[string][]int)
	for k, v := range pt.VariableIndices {
		indices[k] = append([]int(nil), v...) // Create a copy of the slice
	}

	// Augment with variables from nested formatters
	for i, formatter := range formatters {
		for _, varName := range formatter.VariableNames() {
			// Check if this formatter index is already in the list for this variable
			alreadyAdded := false
			for _, existingIdx := range indices[varName] {
				if existingIdx == i {
					alreadyAdded = true
					break
				}
			}
			if !alreadyAdded {
				indices[varName] = append(indices[varName], i)
			}
		}
	}

	return &TemplateBuilder{
		template:          pt.Template,
		formatters:        formatters,
		indices:           indices,
		omitIfFalse:       omitIfFalse,
		implicitFormatter: implicitFormatter,
	}, nil
}

func (tb *TemplateBuilder) SetField(path string, value any) {
	indices, ok := tb.indices[path]
	if !ok {
		// If there's an implicit formatter (omitIfFalse with 0 variables), accept any field
		if tb.implicitFormatter != nil {
			tb.implicitFormatter.SetField(path, value)
		}
		return
	}

	for _, idx := range indices {
		tb.formatters[idx].SetField(path, value)
	}
}

func (tb *TemplateBuilder) GetResult() (any, error) {
	// If omitIfFalse is true, check the appropriate formatter for a false boolean value
	if tb.omitIfFalse {
		var checkFormatter *paramFormatter
		if tb.implicitFormatter != nil {
			// Zero variables case - check implicit formatter
			checkFormatter = tb.implicitFormatter
		} else if len(tb.formatters) == 1 {
			// One variable case - check that formatter
			checkFormatter = tb.formatters[0].(*paramFormatter) // Safe because validated in NewTemplateBuilder
		}

		if checkFormatter != nil {
			if boolVal, ok := checkFormatter.value.(bool); ok && !boolVal {
				return "", nil
			}
		}
	}

	formattedValues := make([]any, len(tb.formatters))

	for i, formatter := range tb.formatters {
		formatted, err := formatter.GetResult()
		if err != nil {
			return nil, fmt.Errorf("failed to format variable at position %d: %w", i, err)
		}
		formattedValues[i] = formatted
	}

	return fmt.Sprintf(tb.template, formattedValues...), nil
}

func (tb *TemplateBuilder) FormatString() string {
	return "%s"
}

func (tb *TemplateBuilder) VariableNames() []string {
	// Collect all variable names from nested formatters
	varNamesSet := make(map[string]bool)
	for _, formatter := range tb.formatters {
		for _, varName := range formatter.VariableNames() {
			varNamesSet[varName] = true
		}
	}

	// Convert to slice
	varNames := make([]string, 0, len(varNamesSet))
	for varName := range varNamesSet {
		varNames = append(varNames, varName)
	}
	return varNames
}

func createEnvVariable(varName string, paramIdx int) (*Variable, error) {
	if varName == "" {
		return nil, fmt.Errorf("environment variable name cannot be empty")
	}

	for _, ch := range varName {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return nil, fmt.Errorf("invalid environment variable name '%s'", varName)
		}
	}

	return &Variable{
		Name:              varName,
		Type:              VariableTypeEnv,
		Index:             paramIdx,
		VariableFormatter: &envVarFormatter{envVarName: varName},
	}, nil
}

func createSchemaVariable(varName string, paramIdx int, opts TemplateParserOptions) (*Variable, error) {
	if varName == "" {
		return nil, fmt.Errorf("paramater name cannot be empty")
	}

	for _, ch := range varName {
		if unicode.IsControl(ch) {
			return nil, fmt.Errorf("invalid paramter name '%s': cannot contain control characters", varName)
		}
	}

	formatter, ok := opts.Formatters[varName]
	if !ok {
		formatString, err := utils.FormatStringForParam(varName, opts.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to create variable for parameter '%s': %w", varName, err)
		}
		formatter = &paramFormatter{
			paramName:    varName,
			formatString: formatString,
		}
	}

	return &Variable{
		Name:              varName,
		Type:              VariableTypeParam,
		Index:             paramIdx,
		VariableFormatter: formatter,
	}, nil
}

type envVarFormatter struct {
	envVarName string
}

func (f *envVarFormatter) SetField(path string, value any) {
}

func (f *envVarFormatter) GetResult() (any, error) {
	val, ok := os.LookupEnv(f.envVarName)
	if !ok {
		return "", fmt.Errorf("environment variable '%s' not set", f.envVarName)
	}
	return val, nil
}

func (f *envVarFormatter) FormatString() string {
	return "%s"
}

func (f *envVarFormatter) VariableNames() []string {
	// Environment variables don't need input fields
	return []string{}
}

type paramFormatter struct {
	paramName    string
	formatString string
	value        any
	hasValue     bool
}

func (f *paramFormatter) SetField(path string, value any) {
	// Empty paramName means accept any field (used for implicit formatters)
	if f.paramName == "" || path == f.paramName {
		f.value = value
		f.hasValue = true
	}
}

func (f *paramFormatter) GetResult() (any, error) {
	if !f.hasValue {
		return nil, fmt.Errorf("parameter '%s' was not provided", f.paramName)
	}
	return f.value, nil
}

func (f *paramFormatter) FormatString() string {
	return f.formatString
}

func (f *paramFormatter) VariableNames() []string {
	if f.paramName == "" {
		return []string{}
	}
	return []string{f.paramName}
}

// NewTemplateFormatter creates a formatter from a template string.
func NewTemplateFormatter(templateStr string, inputSchema *jsonschema.Schema, omitIfFalse bool) (VariableFormatter, error) {
	pt, err := ParseTemplate(templateStr, TemplateParserOptions{
		InputSchema: inputSchema,
	})
	if err != nil {
		return nil, err
	}

	return NewTemplateBuilder(pt, omitIfFalse)
}

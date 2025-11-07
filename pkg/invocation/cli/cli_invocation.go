package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yosida95/uritemplate/v3"
	"go.uber.org/zap"
)

type CliInvoker struct {
	ParsedTemplate *template.ParsedTemplate // Parsed template for the command
	InputSchema    *jsonschema.Resolved     // InputSchema for the tool
	URITemplate    string                   // MCP URI template (for resource templates only)
}

var _ invocation.Invoker = &CliInvoker{}

// newCommandBuilder creates a new commandBuilder from the parsed template.
// A new builder is created for each invocation to avoid sharing state.
func (ci *CliInvoker) newCommandBuilder() (*commandBuilder, error) {
	// Create a new TemplateBuilder for this invocation
	// Note: omitIfFalse is handled by the formatters created during parsing
	templateBuilder, err := template.NewTemplateBuilder(ci.ParsedTemplate, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create template builder: %w", err)
	}

	// Create variable names set for routing
	templateVarNames := make(map[string]bool)
	for _, varName := range templateBuilder.VariableNames() {
		templateVarNames[varName] = true
	}

	return &commandBuilder{
		templateBuilder:  templateBuilder,
		templateVarNames: templateVarNames,
		extraArgs:        make(map[string]any),
	}, nil
}

func (ci *CliInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting CLI tool invocation")

	command, _, err := ci.buildCommandFromArgs(ctx, req.Params.Arguments)
	if err != nil {
		return nil, err
	}

	output, err := ci.executeCommand(ctx, command, nil)
	if err != nil {
		return utils.McpTextError("Command execution failed:\n%s", string(output)), nil
	}

	logger.Info("CLI tool invocation completed successfully")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(output),
			},
		},
	}, nil
}

// executeCommand handles the common command execution cycle.
// It centralizes command creation, execution, output reading, and logging.
// Returns the output bytes and error. Logs sensitive command details to baseLogger only.
func (ci *CliInvoker) executeCommand(
	ctx context.Context,
	command string,
	contextInfo map[string]string, // additional context for logging (e.g., "uri", "template")
) ([]byte, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx)

	// Build log fields with sensitive command details
	logFields := []zap.Field{
		zap.String("command", command),
	}
	for k, v := range contextInfo {
		logFields = append(logFields, zap.String(k, v))
	}

	baseLogger.Debug("Executing CLI command", logFields...)

	cmd := exec.Command("bash", "-c", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		baseLogger.Error("CLI command execution failed", append(logFields,
			zap.String("output", string(output)),
			zap.Error(err))...)
		logger.Error("CLI command execution failed")
		return output, err
	}

	// Server-side only logging with sensitive command details
	baseLogger.Info("CLI command executed successfully", append(logFields,
		zap.Int("output_length", len(output)))...)

	return output, nil
}

// buildCommandFromArgs creates a command builder, parses and validates arguments,
// and returns the built command string along with the parsed argument map.
func (ci *CliInvoker) buildCommandFromArgs(
	ctx context.Context,
	argsBytes []byte,
) (string, map[string]any, error) {
	logger := logging.FromContext(ctx)

	cb, err := ci.newCommandBuilder()
	if err != nil {
		logger.Error("Failed to create command builder", zap.Error(err))
		return "", nil, fmt.Errorf("failed to create command builder: %w", err)
	}

	dj := &invocation.DynamicJson{
		Builders: []invocation.Builder{cb},
	}

	parsed, err := dj.ParseJson(argsBytes, ci.InputSchema.Schema())
	if err != nil {
		logger.Error("Failed to parse request arguments", zap.Error(err))
		return "", nil, fmt.Errorf("failed to parse request: %w", err)
	}

	if err := ci.InputSchema.Validate(parsed); err != nil {
		logger.Error("Failed to validate request arguments", zap.Error(err))
		return "", nil, fmt.Errorf("failed to validate request: %w", err)
	}

	command, err := cb.GetResult()
	if err != nil {
		logger.Error("Failed to build command", zap.Error(err))
		return "", nil, fmt.Errorf("failed to build command: %w", err)
	}

	return command.(string), parsed, nil
}

// buildCommandFromPromptArgs creates a command builder from prompt arguments (map[string]string),
// validates them, and returns the built command string.
func (ci *CliInvoker) buildCommandFromPromptArgs(
	ctx context.Context,
	promptArgs map[string]string,
) (string, error) {
	logger := logging.FromContext(ctx)

	cb, err := ci.newCommandBuilder()
	if err != nil {
		logger.Error("Failed to create command builder", zap.Error(err))
		return "", fmt.Errorf("failed to create command builder: %w", err)
	}

	if promptArgs == nil {
		promptArgs = make(map[string]string)
	}

	// Convert to map[string]any for validation and populate command builder
	argsForValidation := make(map[string]any, len(promptArgs))
	for argName, argValue := range promptArgs {
		cb.SetField(argName, argValue)
		argsForValidation[argName] = argValue
	}

	if err := ci.InputSchema.Validate(argsForValidation); err != nil {
		logger.Error("Failed to validate prompt request arguments", zap.Error(err))
		return "", fmt.Errorf("failed to validate prompt request: %w", err)
	}

	command, err := cb.GetResult()
	if err != nil {
		logger.Error("Failed to build command", zap.Error(err))
		return "", fmt.Errorf("failed to build command: %w", err)
	}

	return command.(string), nil
}

// extractArgsFromURI extracts argument values from a URI by matching it against the URI template.
// Returns a map of argument names to values and the command builder populated with those values.
func (ci *CliInvoker) extractArgsFromURI(
	ctx context.Context,
	uri string,
) (*commandBuilder, map[string]any, error) {
	logger := logging.FromContext(ctx)

	cb, err := ci.newCommandBuilder()
	if err != nil {
		logger.Error("Failed to create command builder", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to create command builder: %w", err)
	}

	// URI template syntax is validated during parsing, so we can safely use it here
	uriTmpl, _ := uritemplate.New(ci.URITemplate)

	// Match the incoming URI against the template to extract argument values
	matches := uriTmpl.Match(uri)
	if matches == nil {
		logger.Error("URI does not match CLI resource template",
			zap.String("uri", uri),
			zap.String("template", ci.URITemplate))
		return nil, nil, fmt.Errorf("URI does not match template")
	}

	// Extract arguments and populate command builder
	argsMap := make(map[string]any)
	for _, paramName := range uriTmpl.Varnames() {
		if val := matches.Get(paramName); val.Valid() {
			argValue := val.String()
			cb.SetField(paramName, argValue)
			argsMap[paramName] = argValue
		} else {
			logger.Error("Missing required parameter in resource template",
				zap.String("parameter", paramName),
				zap.String("uri", uri),
				zap.String("template", ci.URITemplate))
			return nil, nil, fmt.Errorf("missing required parameter: %s", paramName)
		}
	}

	if err := ci.InputSchema.Validate(argsMap); err != nil {
		logger.Error("Failed to validate resource template request",
			zap.String("uri", uri),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to validate resource template request: %w", err)
	}

	return cb, argsMap, nil
}

func (ci *CliInvoker) InvokePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting CLI prompt invocation")

	command, err := ci.buildCommandFromPromptArgs(ctx, req.Params.Arguments)
	if err != nil {
		return nil, err
	}

	output, err := ci.executeCommand(ctx, command, nil)
	if err != nil {
		return utils.McpPromptTextError("Command execution failed:\n%s", string(output)), nil
	}

	logger.Info("CLI prompt invocation completed successfully")

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: string(output)},
			},
		},
	}, nil
}

func (ci *CliInvoker) InvokeResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting CLI resource invocation", zap.String("uri", req.Params.URI))

	// For static resources, the template should have no variables
	// We can use the template directly as the command
	command := ci.ParsedTemplate.Template

	output, err := ci.executeCommand(ctx, command, map[string]string{"uri": req.Params.URI})
	if err != nil {
		logger.Error("CLI resource command execution failed", zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	logger.Info("CLI resource invocation completed successfully", zap.String("uri", req.Params.URI))

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     string(output),
			},
		},
	}, nil
}

func (ci *CliInvoker) InvokeResourceTemplate(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	logger := logging.FromContext(ctx)
	logger.Debug("Starting CLI resource template invocation", zap.String("uri", req.Params.URI))

	cb, _, err := ci.extractArgsFromURI(ctx, req.Params.URI)
	if err != nil {
		return nil, err
	}

	command, err := cb.GetResult()
	if err != nil {
		logger.Error("Failed to build command",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	output, err := ci.executeCommand(ctx, command.(string), map[string]string{
		"uri":      req.Params.URI,
		"template": ci.URITemplate,
	})
	if err != nil {
		logger.Error("CLI resource template command execution failed", zap.String("uri", req.Params.URI))
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	logger.Info("CLI resource template invocation completed successfully", zap.String("uri", req.Params.URI))

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     string(output),
			},
		},
	}, nil
}

type commandBuilder struct {
	templateBuilder  *template.TemplateBuilder
	templateVarNames map[string]bool // Set of variable names the template cares about
	extraArgs        map[string]any
}

var _ invocation.Builder = &commandBuilder{}

func (cb *commandBuilder) SetField(path string, value any) {
	// If this is a variable that the template cares about, propagate to the template
	if cb.templateVarNames[path] {
		cb.templateBuilder.SetField(path, value)
	} else {
		// Otherwise, store it in extra args
		cb.extraArgs[path] = value
	}
}

func (cb *commandBuilder) GetResult() (any, error) {
	// Get the formatted command from the template
	templateResult, err := cb.templateBuilder.GetResult()
	if err != nil {
		return nil, err
	}

	// Build the full command with extra args
	formattedParts := make([]string, 0, len(cb.extraArgs)+1)
	formattedParts = append(formattedParts, templateResult.(string))

	for argName, argVal := range cb.extraArgs {
		formattedParts = append(formattedParts, fmt.Sprintf("--%s=%v", argName, argVal))
	}

	return strings.Join(formattedParts, " "), nil
}

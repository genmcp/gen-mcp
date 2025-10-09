package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yosida95/uritemplate/v3"
	"go.uber.org/zap"
)

type Formatter interface {
	FormatValue(v any) string
}

type CliInvoker struct {
	CommandTemplate    string               // template string for the command to execute
	ArgumentIndices    map[string]int       // map to where each argument should go in the command
	ArgumentFormatters map[string]Formatter // map to the functions to format each variable
	InputSchema        *jsonschema.Resolved // InputSchema for the tool
	URITemplate        string               // MCP URI template (for resource templates only)
}

var _ invocation.Invoker = &CliInvoker{}

func (ci *CliInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting CLI tool invocation")

	cb := &commandBuilder{
		commandTemplate: ci.CommandTemplate,
		argIndices:      ci.ArgumentIndices,
		argFormatters:   ci.ArgumentFormatters,
		argValues:       make([]any, len(ci.ArgumentIndices)),
		extraArgs:       make(map[string]any),
	}

	dj := &invocation.DynamicJson{
		Builders: []invocation.Builder{cb},
	}

	parsed, err := dj.ParseJson(req.Params.Arguments, ci.InputSchema.Schema())
	if err != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to parse CLI tool call request", zap.Error(err))
		// Log generic error for client
		logger.Error("Tool request parsing failed", zap.String("error", "parsing error"))
		return utils.McpTextError("failed to parse tool call request"), nil
	}

	err = ci.InputSchema.Validate(parsed)
	if err != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to validate CLI tool call request", zap.Error(err))
		// Log generic error for client
		logger.Error("Tool request validation failed", zap.String("error", "validation error"))
		return utils.McpTextError("failed to validate tool call request"), nil
	}

	command, _ := cb.GetResult()

	// Server-side only logging with sensitive command details
	baseLogger.Debug("Executing CLI command", zap.String("command", command.(string)))

	cmd := exec.Command("bash", "-c", command.(string))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Server-side only logging with sensitive command details
		baseLogger.Error("CLI command execution failed",
			zap.String("command", command.(string)),
			zap.String("output", string(output)),
			zap.Error(err))
		// Log generic error for client
		logger.Error("CLI command execution failed", zap.String("error", "execution error"))
		return utils.McpTextError("command execution failed"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Info("CLI command executed successfully",
		zap.String("command", command.(string)),
		zap.Int("output_length", len(output)))

	logger.Info("CLI tool invocation completed successfully")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(output),
			},
		},
	}, nil
}

type commandBuilder struct {
	commandTemplate string
	argIndices      map[string]int
	argFormatters   map[string]Formatter
	argValues       []any
	extraArgs       map[string]any
}

func (cb *commandBuilder) SetField(path string, value any) {
	idx, ok := cb.argIndices[path]
	if ok {
		cb.argValues[idx] = value
	} else {
		cb.extraArgs[path] = value
	}
}

func (cb *commandBuilder) GetResult() (any, error) {
	for argName, argIdx := range cb.argIndices {
		cb.argValues[argIdx] = cb.argFormatters[argName].FormatValue(cb.argValues[argIdx])
	}

	formattedParts := make([]string, 0, len(cb.extraArgs)+1)
	formattedParts = append(formattedParts, fmt.Sprintf(cb.commandTemplate, cb.argValues...))
	for argName, argVal := range cb.extraArgs {
		formattedParts = append(formattedParts, fmt.Sprintf("--%s=%v", argName, argVal))
	}

	return strings.Join(formattedParts, " "), nil
}

func (ci *CliInvoker) InvokePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	logger := logging.FromContext(ctx)
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting CLI prompt invocation")

	cb := &commandBuilder{
		commandTemplate: ci.CommandTemplate,
		argIndices:      ci.ArgumentIndices,
		argFormatters:   ci.ArgumentFormatters,
		argValues:       make([]any, len(ci.ArgumentIndices)),
		extraArgs:       make(map[string]any),
	}

	promptArgs := req.Params.Arguments
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
		// Log detailed error server-side only
		baseLogger.Error("Failed to validate CLI prompt request arguments", zap.Error(err))
		// Log generic error for client
		logger.Error("Prompt request validation failed", zap.String("error", "validation error"))
		return utils.McpPromptTextError("failed to validate prompt request"), nil
	}

	command, err := cb.GetResult()
	if err != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to build CLI prompt command", zap.Error(err))
		// Log generic error for client
		logger.Error("Prompt command preparation failed", zap.String("error", "command building error"))
		return utils.McpPromptTextError("failed to build command"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Debug("Executing CLI prompt command", zap.String("command", command.(string)))

	cmd := exec.Command("bash", "-c", command.(string))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Server-side only logging with sensitive command details
		baseLogger.Error("CLI prompt command execution failed",
			zap.String("command", command.(string)),
			zap.String("output", string(output)),
			zap.Error(err))
		// Log generic error for client
		logger.Error("CLI prompt command execution failed", zap.String("error", "execution error"))
		return utils.McpPromptTextError("command execution failed"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Info("CLI prompt command executed successfully",
		zap.String("command", command.(string)),
		zap.Int("output_length", len(output)))

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
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting CLI resource invocation", zap.String("uri", req.Params.URI))

	// Server-side only logging with sensitive command details
	baseLogger.Debug("Executing CLI resource command", zap.String("command", ci.CommandTemplate))

	cmd := exec.Command("bash", "-c", ci.CommandTemplate)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Server-side only logging with sensitive command details
		baseLogger.Error("CLI resource command execution failed",
			zap.String("command", ci.CommandTemplate),
			zap.String("uri", req.Params.URI),
			zap.String("output", string(output)),
			zap.Error(err))
		// Log generic error for client
		logger.Error("CLI resource command execution failed", zap.String("error", "execution error"))
		return utils.McpResourceTextError("command execution failed"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Info("CLI resource command executed successfully",
		zap.String("command", ci.CommandTemplate),
		zap.String("uri", req.Params.URI),
		zap.Int("output_length", len(output)))

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
	baseLogger := logging.BaseFromContext(ctx) // Server-side only for sensitive details

	logger.Debug("Starting CLI resource template invocation", zap.String("uri", req.Params.URI))

	cb := &commandBuilder{
		commandTemplate: ci.CommandTemplate,
		argIndices:      ci.ArgumentIndices,
		argFormatters:   ci.ArgumentFormatters,
		argValues:       make([]any, len(ci.ArgumentIndices)),
		extraArgs:       make(map[string]any),
	}

	//  URI template syntax is validated during parsing, so we can safely use it here
	uriTmpl, _ := uritemplate.New(ci.URITemplate)

	// Match the incoming URI against the template to extract argument values
	matches := uriTmpl.Match(req.Params.URI)
	if matches == nil {
		// Server-side only logging with sensitive template details
		baseLogger.Error("URI does not match CLI resource template",
			zap.String("uri", req.Params.URI),
			zap.String("template", ci.URITemplate))
		// Log generic error for client
		logger.Error("URI does not match template", zap.String("error", "template mismatch"))
		return utils.McpResourceTextError("URI does not match template"), nil
	}

	// Extract arguments and populate command builder
	argsMap := make(map[string]any)
	for _, paramName := range uriTmpl.Varnames() {
		if val := matches.Get(paramName); val.Valid() {
			argValue := val.String()
			cb.SetField(paramName, argValue)
			argsMap[paramName] = argValue
		} else {
			// Log detailed error server-side only
			baseLogger.Error("Missing required parameter in resource template",
				zap.String("parameter", paramName),
				zap.String("uri", req.Params.URI),
				zap.String("template", ci.URITemplate))
			// Log generic error for client
			logger.Error("Missing required parameter in resource template", zap.String("error", "missing parameter"))
			return utils.McpResourceTextError("missing required parameter"), nil
		}
	}

	if err := ci.InputSchema.Validate(argsMap); err != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to validate CLI resource template request",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log generic error for client
		logger.Error("Resource template request validation failed", zap.String("error", "validation error"))
		return utils.McpResourceTextError("failed to validate resource template request"), nil
	}

	command, err := cb.GetResult()
	if err != nil {
		// Log detailed error server-side only
		baseLogger.Error("Failed to build CLI resource template command",
			zap.String("uri", req.Params.URI),
			zap.Error(err))
		// Log generic error for client
		logger.Error("Resource template command preparation failed", zap.String("error", "command building error"))
		return utils.McpResourceTextError("failed to build command"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Debug("Executing CLI resource template command",
		zap.String("command", command.(string)),
		zap.String("uri", req.Params.URI),
		zap.String("template", ci.URITemplate))

	cmd := exec.Command("bash", "-c", command.(string))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Server-side only logging with sensitive command details
		baseLogger.Error("CLI resource template command execution failed",
			zap.String("command", command.(string)),
			zap.String("uri", req.Params.URI),
			zap.String("output", string(output)),
			zap.Error(err))
		// Log generic error for client
		logger.Error("CLI resource template command execution failed", zap.String("error", "execution error"))
		return utils.McpResourceTextError("command execution failed"), nil
	}

	// Server-side only logging with sensitive command details
	baseLogger.Info("CLI resource template command executed successfully",
		zap.String("command", command.(string)),
		zap.String("uri", req.Params.URI),
		zap.Int("output_length", len(output)))

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

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Formatter interface {
	FormatValue(v any) string
}

type CliInvoker struct {
	CommandTemplate    string               // template string for the command to execute
	ArgumentIndices    map[string]int       // map to where each argument should go in the command
	ArgumentFormatters map[string]Formatter // map to the functions to format each variable
	InputSchema        *jsonschema.Resolved // InputSchema for the tool
}

var _ invocation.Invoker = &CliInvoker{}

func (ci *CliInvoker) Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		return utils.McpTextError("failed to parse tool call request: %s", err.Error()), err
	}

	err = ci.InputSchema.Validate(parsed)
	if err != nil {
		return utils.McpTextError("failed to validate parsed tool call request: %s", err.Error()), err
	}

	command, _ := cb.GetResult()

	cmd := exec.Command("bash", "-c", command.(string))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return utils.McpTextError("encountered error while calling command: %s. output was: %s.", err.Error(), string(output)), err
	}

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

	args := req.Params.Arguments
	if args == nil {
		args = make(map[string]string)
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		return utils.McpPromptTextError("failed to marshal prompt request arguments: %s", err.Error()), err
	}

	parsed, err := dj.ParseJson(argsBytes, ci.InputSchema.Schema())
	if err != nil {
		return utils.McpPromptTextError("failed to parse prompt request arguments: %s", err.Error()), err
	}

	if err := ci.InputSchema.Validate(parsed); err != nil {
		return utils.McpPromptTextError("failed to validate prompt request arguments: %s", err.Error()), err
	}

	command, err := cb.GetResult()
	if err != nil {
		return utils.McpPromptTextError("failed to build command: %s", err.Error()), err
	}

	cmd := exec.Command("bash", "-c", command.(string))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return utils.McpPromptTextError("encountered error while calling command: %s. output was: %s.", err.Error(), string(output)), err
	}

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: string(output)},
			},
		},
	}, nil
}

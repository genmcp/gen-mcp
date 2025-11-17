package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/cli"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/google/jsonschema-go/jsonschema"
)

func ExtractCLICommandInfo(cliCommand string, commandItems *[]CommandItem) (bool, error) {
	is_sub_command, err := DetectSubCommand(cliCommand)
	if err != nil {
		return false, err
	}
	// fmt.Println("cliCommand:", cliCommand)
	// fmt.Println("is_sub_command:", is_sub_command)

	if is_sub_command {
		subcommands, err := ExtractSubCommands(cliCommand)
		if err != nil {
			return false, err
		}
		// fmt.Println("subcommands:", subcommands)
		for _, subcommand := range subcommands {
			_, err := ExtractCLICommandInfo(cliCommand+" "+subcommand, commandItems)
			if err != nil {
				return false, err
			}
		}
	} else {
		command, err := ExtractCommand(cliCommand)
		if err != nil {
			return false, err
		}
		// fmt.Println("command:", command)
		*commandItems = append(*commandItems, command)
	}

	return true, nil
}

func ConvertCommandsToMCPFile(commandItems *[]CommandItem) (*mcpserver.MCPServer, error) {
	if commandItems == nil || len(*commandItems) == 0 {
		return nil, fmt.Errorf("no command items provided")
	}

	// Create tools from command items
	tools := make([]*definitions.Tool, 0, len(*commandItems))

	for _, commandItem := range *commandItems {
		tool, err := convertCommandItemToTool(commandItem)
		if err != nil {
			return nil, fmt.Errorf("failed to convert command '%s' to tool: %w", commandItem.Command, err)
		}
		tools = append(tools, tool)
	}

	// Create MCP server
	mcpServer := &mcpserver.MCPServer{
		MCPToolDefinitions: definitions.MCPToolDefinitions{
			Name:    "cli-generated-server",
			Version: "0.0.1",
			Tools:   tools,
		},
		MCPServerConfig: serverconfig.MCPServerConfig{
			Name:    "cli-generated-server",
			Version: "0.0.1",
			Runtime: &serverconfig.ServerRuntime{
				TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
					Port: 7008,
				},
			},
		},
	}

	return mcpServer, nil
}

func convertCommandItemToTool(commandItem CommandItem) (*definitions.Tool, error) {
	// Create input schema for the tool based on arguments
	inputSchema, err := createInputSchemaFromArguments(commandItem.Data.Arguments, commandItem.Data.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to create input schema: %w", err)
	}

	// Create CLI invocation data
	invocationData, err := createCLIInvocationData(commandItem.Command, commandItem.Data.Arguments, commandItem.Data.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to create invocation data: %w", err)
	}

	// Create tool name from command (replace spaces and special chars with underscores)
	toolName := strings.ReplaceAll(strings.ReplaceAll(commandItem.Command, " ", "_"), "-", "_")

	cfg := &cli.CliInvocationConfig{}
	if err := json.Unmarshal(invocationData, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal invocation data: %w", err)
	}

	tool := &definitions.Tool{
		Name:        toolName,
		Title:       commandItem.Command,
		Description: commandItem.Data.Description,
		InputSchema: inputSchema,
		InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
			Type:   cli.InvocationType,
			Config: cfg,
		},
	}

	return tool, nil
}

func createInputSchemaFromArguments(arguments []Argument, options []Option) (*jsonschema.Schema, error) {
	properties := make(map[string]*jsonschema.Schema)
	required := make([]string, 0)

	for _, arg := range arguments {
		// Create a description based on the argument name if no specific description is available
		description := fmt.Sprintf("%s parameter", processSafeName(arg.Name))

		properties[processSafeName(arg.Name)] = &jsonschema.Schema{
			Type:        "string",
			Description: description,
		}

		if !arg.Optional {
			required = append(required, processSafeName(arg.Name))
		}
	}

	for _, option := range options {
		optionType := "string"
		if option.Type == "" {
			optionType = "boolean"
		}
		properties[processSafeName(option.Flag)] = &jsonschema.Schema{
			Type:        optionType,
			Description: option.Description,
		}
	}

	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	return schema, nil
}

func createCLIInvocationData(command string, arguments []Argument, options []Option) (json.RawMessage, error) {
	// Create command template with parameter placeholders
	commandTemplate := command
	for _, arg := range arguments {
		commandTemplate += " {" + processSafeName(arg.Name) + "}"
	}
	for _, option := range options {
		commandTemplate += " {" + processSafeName(option.Flag) + "}"
	}

	templateVariables := createTemplateVariables(arguments, options)

	invocation := map[string]interface{}{
		"command":           commandTemplate,
		"templateVariables": templateVariables,
	}

	data, err := json.Marshal(invocation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CLI invocation data: %w", err)
	}

	return json.RawMessage(data), nil
}

func createTemplateVariables(arguments []Argument, options []Option) map[string]interface{} {
	templateVariables := make(map[string]interface{})
	for _, arg := range arguments {
		templateVariables[processSafeName(arg.Name)] = map[string]interface{}{
			"property":    processSafeName(arg.Name),
			"format":      "\"" + "{" + processSafeName(arg.Name) + "}\"",
			"omitIfFalse": arg.Optional,
		}
	}
	for _, option := range options {
		format := option.Flag
		if option.Type != "" {
			format = format + " {" + processSafeName(option.Flag) + "}"
		}
		templateVariables[processSafeName(option.Flag)] = map[string]interface{}{
			"property":    processSafeName(option.Flag),
			"format":      format,
			"omitIfFalse": option.Type == "",
		}
	}
	return templateVariables
}

func processSafeName(command string) string {
	if after, ok := strings.CutPrefix(command, "--"); ok {
		command = after
	}
	command = strings.TrimSpace(command)
	command = strings.ToLower(command)
	command = strings.ReplaceAll(command, "-", "_")
	command = strings.ReplaceAll(command, " ", "_")
	command = strings.ReplaceAll(command, ":", "_")
	command = strings.ReplaceAll(command, ".", "")
	command = strings.ReplaceAll(command, "*", "")
	command = strings.ReplaceAll(command, "|", "")

	// Remove all non-alphanumeric characters except underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	command = re.ReplaceAllString(command, "")

	return command
}

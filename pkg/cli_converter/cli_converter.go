package cli_converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
)

func ExtractCLICommandInfo(cliCommand string, commandItems *[]CommandItem) (bool, error) {
	is_sub_command, err := DetectSubCommand(cliCommand)
	if err != nil {
		return false, err
	}
	fmt.Println("cliCommand:", cliCommand)
	fmt.Println("is_sub_command:", is_sub_command)

	if is_sub_command {
		subcommands, err := ExtractSubCommands(cliCommand)
		if err != nil {
			return false, err
		}
		fmt.Println("subcommands:", subcommands)
		for _, subcommand := range subcommands {
			ExtractCLICommandInfo(cliCommand+" "+subcommand, commandItems)
		}
	} else {
		command, err := ExtractCommand(cliCommand)
		if err != nil {
			return false, err
		}
		fmt.Println("command:", command)
		*commandItems = append(*commandItems, command)
	}

	return true, nil
}

func ConvertCommandsToMCPFile(commandItems *[]CommandItem) (*mcpfile.MCPFile, error) {
	if commandItems == nil || len(*commandItems) == 0 {
		return nil, fmt.Errorf("no command items provided")
	}

	// Create tools from command items
	tools := make([]*mcpfile.Tool, 0, len(*commandItems))

	for _, commandItem := range *commandItems {
		tool, err := convertCommandItemToTool(commandItem)
		if err != nil {
			return nil, fmt.Errorf("failed to convert command '%s' to tool: %w", commandItem.Command, err)
		}
		tools = append(tools, tool)
	}

	// Create MCP server
	server := &mcpfile.MCPServer{
		Name:    "cli-generated-server",
		Version: "0.0.1",
		Runtime: &mcpfile.ServerRuntime{
			TransportProtocol: mcpfile.TransportProtocolStreamableHttp,
			StreamableHTTPConfig: &mcpfile.StreamableHTTPConfig{
				Port: 7008,
			},
		},
		Tools: tools,
	}

	// Create MCP file
	mcpFile := &mcpfile.MCPFile{
		FileVersion: mcpfile.MCPFileVersion,
		Servers:     []*mcpfile.MCPServer{server},
	}

	return mcpFile, nil
}

func convertCommandItemToTool(commandItem CommandItem) (*mcpfile.Tool, error) {
	// Create input schema for the tool based on arguments
	inputSchema, err := createInputSchemaFromArguments(commandItem.Data.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to create input schema: %w", err)
	}

	// Create CLI invocation data
	invocationData, err := createCLIInvocationData(commandItem.Command, commandItem.Data.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to create invocation data: %w", err)
	}

	// Create tool name from command (replace spaces and special chars with underscores)
	toolName := strings.ReplaceAll(strings.ReplaceAll(commandItem.Command, " ", "_"), "-", "_")

	tool := &mcpfile.Tool{
		Name:           toolName,
		Title:          commandItem.Command,
		Description:    commandItem.Data.Description,
		InputSchema:    inputSchema,
		InvocationData: invocationData,
		InvocationType: mcpfile.InvocationTypeCli,
	}

	return tool, nil
}

func createInputSchemaFromArguments(arguments []Argument) (*jsonschema.Schema, error) {
	properties := make(map[string]*jsonschema.Schema)
	required := make([]string, 0)

	for _, arg := range arguments {
		// Create a description based on the argument name if no specific description is available
		description := fmt.Sprintf("%s parameter", strings.ReplaceAll(arg.Name, "_", " "))

		properties[arg.Name] = &jsonschema.Schema{
			Type:        "string",
			Description: description,
		}

		if !arg.Optional {
			required = append(required, arg.Name)
		}
	}

	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
	}

	return schema, nil
}

func createCLIInvocationData(command string, arguments []Argument) (json.RawMessage, error) {
	// Create command template with parameter placeholders
	commandTemplate := command
	for _, arg := range arguments {
		commandTemplate += " {" + arg.Name + "}"
	}

	templateVariables := createTemplateVariables(arguments)

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

func createTemplateVariables(arguments []Argument) map[string]interface{} {
	templateVariables := make(map[string]interface{})
	for _, arg := range arguments {
		templateVariables[arg.Name] = map[string]interface{}{
			"property":    arg.Name,
			"format":      "\"" + "{" + arg.Name + "}\"",
			"omitIfFalse": arg.Optional,
		}
	}
	return templateVariables
}

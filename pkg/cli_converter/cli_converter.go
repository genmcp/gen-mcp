package cli_converter

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
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
		return &mcpfile.MCPFile{
			FileVersion: mcpfile.MCPFileVersion,
			Servers:     []*mcpfile.MCPServer{},
		}, nil
	}

	// Create tools from command items
	var tools []*mcpfile.Tool
	for _, cmdItem := range *commandItems {
		tool, err := convertCommandItemToTool(&cmdItem)
		if err != nil {
			return nil, fmt.Errorf("failed to convert command '%s' to tool: %w", cmdItem.Command, err)
		}
		tools = append(tools, tool)
	}

	// Create a default MCP server with stdio runtime
	server := &mcpfile.MCPServer{
		Name:    "cli-tools-server",
		Version: "1.0.0",
		Runtime: &mcpfile.ServerRuntime{
			TransportProtocol: mcpfile.TransportProtocolStdio,
			StdioConfig:       &mcpfile.StdioConfig{},
		},
		Tools: tools,
	}

	// Create the MCP file
	mcpFile := &mcpfile.MCPFile{
		FileVersion: mcpfile.MCPFileVersion,
		Servers:     []*mcpfile.MCPServer{server},
	}

	return mcpFile, nil
}

// convertCommandItemToTool converts a single CommandItem to an MCP Tool
func convertCommandItemToTool(cmdItem *CommandItem) (*mcpfile.Tool, error) {
	// Generate tool name from command (replace spaces and special chars with underscores)
	toolName := generateToolName(cmdItem.Command)

	// Create input schema from arguments and options
	inputSchema := createInputSchema(&cmdItem.Data)

	// Create CLI invocation with template variables
	cliInvocation := createCliInvocation(cmdItem.Command, &cmdItem.Data)

	tool := &mcpfile.Tool{
		Name:        toolName,
		Title:       cmdItem.Command,
		Description: cmdItem.Data.Description,
		InputSchema: inputSchema,
		Invocation:  cliInvocation,
	}

	return tool, nil
}

// generateToolName creates a valid tool name from a command string
func generateToolName(command string) string {
	// Replace spaces and special characters with underscores
	toolName := ""
	for _, char := range command {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			toolName += string(char)
		} else if char == ' ' || char == '-' {
			toolName += "_"
		}
	}
	return toolName
}

// createInputSchema generates a JSON schema for the tool input based on arguments and options
func createInputSchema(cmd *Command) *mcpfile.JsonSchema {
	properties := make(map[string]*mcpfile.JsonSchema)
	var required []string

	// Add arguments to schema
	for _, arg := range cmd.Arguments {
		properties[arg.Name] = &mcpfile.JsonSchema{
			Type:        mcpfile.JsonSchemaTypeString,
			Description: fmt.Sprintf("Argument: %s", arg.Name),
		}
		if !arg.Optional {
			required = append(required, arg.Name)
		}
	}

	// Add options to schema
	for _, opt := range cmd.Options {
		// Remove leading dashes from flag name to create property name
		propName := opt.Flag
		if propName[0] == '-' {
			propName = propName[1:]
		}
		if propName[0] == '-' {
			propName = propName[1:]
		}

		properties[propName] = &mcpfile.JsonSchema{
			Type:        mcpfile.JsonSchemaTypeString,
			Description: opt.Description,
		}
	}

	schema := &mcpfile.JsonSchema{
		Type:                 mcpfile.JsonSchemaTypeObject,
		Properties:           properties,
		AdditionalProperties: &[]bool{false}[0], // Set to false
		Required:             required,
		Description:          fmt.Sprintf("Input schema for %s command", cmd.Description),
	}

	return schema
}

// createCliInvocation creates a CLI invocation with proper template variables
func createCliInvocation(command string, cmd *Command) *mcpfile.CliInvocation {
	templateVars := make(map[string]*mcpfile.TemplateVariable)

	// Create template variables for arguments
	for _, arg := range cmd.Arguments {
		templateVars[arg.Name] = &mcpfile.TemplateVariable{
			Property: arg.Name,
			Format:   "{" + arg.Name + "}",
		}
		// Add positional argument placeholder to command using {} syntax
		command += " {}"
	}

	// Create template variables for options
	for _, opt := range cmd.Options {
		propName := opt.Flag
		if len(propName) > 0 && propName[0] == '-' {
			propName = propName[1:]
		}
		if len(propName) > 0 && propName[0] == '-' {
			propName = propName[1:]
		}

		templateVars[propName] = &mcpfile.TemplateVariable{
			Property: propName,
			Format:   fmt.Sprintf("%s {%s}", opt.Flag, propName),
		}
	}

	return &mcpfile.CliInvocation{
		Command:           command,
		TemplateVariables: templateVars,
	}
}

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
	return nil, nil
}

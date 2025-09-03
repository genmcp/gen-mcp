package cli_converter

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func ConvertCliCommandToMcpFile(cliCommand string) (*mcpfile.MCPFile, error) {
	is_sub_command, err := DetectSubCommand(cliCommand)
	if err != nil {
		return nil, err
	}
	fmt.Println("cliCommand:", cliCommand)
	fmt.Println("is_sub_command:", is_sub_command)

	if is_sub_command {
		subcommands, err := ExtractSubCommands(cliCommand)
		if err != nil {
			return nil, err
		}
		fmt.Println("subcommands:", subcommands)
	}

	return nil, nil
}

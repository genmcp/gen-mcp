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
	fmt.Println("is_sub_command:", is_sub_command)

	return nil, nil
}

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Cali0707/AutoMCP/pkg/cli/utils"
	"github.com/Cali0707/AutoMCP/pkg/mcpfile"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&mcpFilePath, "file", "f", "mcpfile.yaml", "mcp file to read from")
	runCmd.Flags().BoolVarP(&detach, "detach", "d", false, "whether to detach when running")
}

var mcpFilePath string
var detach bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a MCP server",
	Run:   executeRunCmd,
}

func executeRunCmd(cobraCmd *cobra.Command, args []string) {
	cmd := exec.Command("automcp-server")
	mcpFilePath, err := filepath.Abs(mcpFilePath)
	if err != nil {
		fmt.Printf("failed to resolve mcp file path: %s\n", err.Error())
		return
	}

	if _, err := os.Stat(mcpFilePath); err != nil {
		fmt.Printf("no file found at mcp file path\n")
		return
	}

	mcpFile, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		fmt.Printf("invalid mcp file: %s\n", err)
	}

	cmd.Env = append(cmd.Environ(), fmt.Sprintf("MCP_FILE_PATH=%s", mcpFilePath))

	for _, s := range mcpFile.Servers {
		if s.Runtime.TransportProtocol == mcpfile.TransportProtocolStdio && detach {
			// TODO: re-enable this logging when we figure out logging w. stdio
			// fmt.Printf("cannot detach when running stdio transport\n")
			detach = false
		}
	}

	if !detach {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("automcp-server failed with %s\n", err.Error())
		}
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start automcp-server: %s\n", err.Error())
	}

	processManager := utils.GetProcessManager()
	err = processManager.SaveProcessId(mcpFilePath, cmd.Process.Pid)
	if err != nil {
		fmt.Printf("failed to save pid for automcp server, to stop the server you will need to manually kill pid %d: %s\n", cmd.Process.Pid, err.Error())
	}

	fmt.Printf("successfully started AutoMCP server...\n")
}

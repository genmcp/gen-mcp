package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cali0707/AutoMCP/pkg/cli/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringVarP(&mcpFilePath, "file", "f", "mcpfile.yaml", "mcp file to read from")
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a MCP server",
	Run:   executeStopCmd,
}

func executeStopCmd(cobraCmd *cobra.Command, args []string) {
	mcpFilePath, err := filepath.Abs(mcpFilePath)
	if err != nil {
		fmt.Printf("failed to resolve mcp file path: %s\n", err.Error())
		return
	}

	if _, err := os.Stat(mcpFilePath); err != nil {
		fmt.Printf("no file found at mcp file path\n")
		return
	}

	processManager := utils.GetProcessManager()
	pid, err := processManager.GetProcessId(mcpFilePath)
	if err != nil {
		fmt.Printf("failed to get pid for automcp server\n")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("failed to find process for pid %d: %s\n", pid, err.Error())
		processManager.DeleteProcessId(mcpFilePath)
		return
	}

	err = proc.Kill()
	if err != nil {
		fmt.Printf("failed to kill automcp process with pid %d: %s\n", pid, err.Error())
		return
	}

	fmt.Printf("successfully started AutoMCP server...\n")
}

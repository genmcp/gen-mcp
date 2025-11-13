package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringVarP(&stopMCPFilePath, "file", "f", "mcpfile.yaml", "mcp file to read from")
}

var stopMCPFilePath string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a MCP server",
	Run:   executeStopCmd,
}

func executeStopCmd(cobraCmd *cobra.Command, args []string) {
	mcpFilePath, err := filepath.Abs(stopMCPFilePath)
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
		fmt.Printf("failed to get pid for genmcp server\n")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("failed to find process for pid %d: %s\n", pid, err.Error())
		if err := processManager.DeleteProcessId(mcpFilePath); err != nil {
			fmt.Printf("failed to delete process id: %s\n", err.Error())
		}
		return
	}

	err = proc.Kill()
	if err != nil {
		fmt.Printf("failed to kill genmcp process with pid %d: %s\n", pid, err.Error())
		return
	}

	if err := processManager.DeleteProcessId(mcpFilePath); err != nil {
		fmt.Printf("failed to delete process id: %s\n", err.Error())
	}

	fmt.Printf("successfully stopped gen-mcp server...\n")
}

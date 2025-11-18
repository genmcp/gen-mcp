package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/runtime"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	// TODO: change it to mcpfile
	runCmd.Flags().StringVarP(&runToolDefinitionsPath, "f", "f", "mcpfile.yaml", "the path to the tool definitions file")
	// TODO: rename
	runCmd.Flags().StringVarP(&runServerConfigPath, "server-config", "s", "mcpfile-server.yaml", "the path to the server config file")
	runCmd.Flags().BoolVarP(&detach, "detach", "d", false, "whether to detach when running")
}

var runToolDefinitionsPath string
var runServerConfigPath string
var detach bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a MCP server",
	Run:   executeRunCmd,
}

func executeRunCmd(cobraCmd *cobra.Command, args []string) {
	toolDefinitionsPath, err := filepath.Abs(runToolDefinitionsPath)
	if err != nil {
		fmt.Printf("failed to resolve tool definitions file path: %s\n", err.Error())
		return
	}

	serverConfigPath, err := filepath.Abs(runServerConfigPath)
	if err != nil {
		fmt.Printf("failed to resolve server config file path: %s\n", err.Error())
		return
	}

	if _, err := os.Stat(toolDefinitionsPath); err != nil {
		fmt.Printf("no file found at tool definitions path: %s\n", toolDefinitionsPath)
		return
	}

	if _, err := os.Stat(serverConfigPath); err != nil {
		fmt.Printf("no file found at server config path: %s\n", serverConfigPath)
		return
	}

	// Parse and validate tool definitions file
	_, err = definitions.ParseMCPFile(toolDefinitionsPath)
	if err != nil {
		fmt.Printf("invalid tool definitions file: %s\n", err)
		return
	}

	// Parse and validate server config file
	serverConfigFile, err := serverconfig.ParseMCPFile(serverConfigPath)
	if err != nil {
		fmt.Printf("invalid server config file: %s\n", err)
		return
	}

	// Check transport protocol for detach validation
	if serverConfigFile.Runtime != nil && serverConfigFile.Runtime.TransportProtocol == serverconfig.TransportProtocolStdio && detach {
		// TODO: re-enable this logging when we figure out logging w. stdio
		// fmt.Printf("cannot detach when running stdio transport\n")
		detach = false
	}

	// Use tool definitions path as the identifier for process management (for backward compatibility)
	processIdentifier := toolDefinitionsPath

	if !detach {
		// Run servers directly in the current process
		err := runtime.RunServers(context.Background(), toolDefinitionsPath, serverConfigPath)
		if err != nil {
			fmt.Printf("genmcp-server failed with %s\n", err.Error())
		}
		return
	}

	// Detached mode: spawn the same command without --detach flag
	cmd := exec.Command(os.Args[0], "run", "-t", toolDefinitionsPath, "-s", serverConfigPath)
	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start genmcp-server: %s\n", err.Error())
		return
	}

	// Save PID for stop command (using tool definitions path as identifier)
	processManager := utils.GetProcessManager()
	err = processManager.SaveProcessId(processIdentifier, cmd.Process.Pid)
	if err != nil {
		fmt.Printf("failed to save pid for genmcp server, to stop the server you will need to manually kill pid %d: %s\n", cmd.Process.Pid, err.Error())
	}

	fmt.Printf("successfully started gen-mcp server...\n")
}

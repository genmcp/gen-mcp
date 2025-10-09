package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	"github.com/genmcp/gen-mcp/pkg/invocation"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/cli"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&mcpFilePath, "file", "f", "mcpfile.yaml", "mcp file to read from")
	runCmd.Flags().StringVarP(&serverConfigPath, "server-config", "s", "", "optional server config file (mcpserver.yaml)")
	runCmd.Flags().BoolVarP(&detach, "detach", "d", false, "whether to detach when running")
}

var mcpFilePath string
var serverConfigPath string
var detach bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a MCP server",
	Run:   executeRunCmd,
}

func executeRunCmd(cobraCmd *cobra.Command, args []string) {
	mcpFilePath, err := filepath.Abs(mcpFilePath)
	if err != nil {
		fmt.Printf("failed to resolve mcp file path: %s\n", err.Error())
		return
	}

	if _, err := os.Stat(mcpFilePath); err != nil {
		fmt.Printf("no file found at mcp file path\n")
		return
	}

	// Resolve server config path if provided
	if serverConfigPath != "" {
		serverConfigPath, err = filepath.Abs(serverConfigPath)
		if err != nil {
			fmt.Printf("failed to resolve server config path: %s\n", err.Error())
			return
		}

		if _, err := os.Stat(serverConfigPath); err != nil {
			fmt.Printf("no file found at server config path\n")
			return
		}
	}

	mcpFile, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		fmt.Printf("invalid mcp file: %s\n", err)
		return
	}

	err = mcpFile.Validate(invocation.InvocationValidator)
	if err != nil {
		fmt.Printf("invalid mcp file: %s\n", err)
		return
	}

	// Determine which runtime to check for stdio
	var transportProtocol string
	if serverConfigPath != "" {
		serverConfig, err := mcpfile.ParseMCPServerConfig(serverConfigPath)
		if err != nil {
			fmt.Printf("invalid server config: %s\n", err)
			return
		}
		if serverConfig.Runtime != nil {
			transportProtocol = serverConfig.Runtime.TransportProtocol
		}
	} else if mcpFile.Runtime != nil {
		transportProtocol = mcpFile.Runtime.TransportProtocol
	}

	if transportProtocol == mcpfile.TransportProtocolStdio && detach {
		// TODO: re-enable this logging when we figure out logging w. stdio
		// fmt.Printf("cannot detach when running stdio transport\n")
		detach = false
	}

	if !detach {
		// Run servers directly in the current process
		err := mcpserver.RunServersWithConfig(context.Background(), mcpFilePath, serverConfigPath)
		if err != nil {
			fmt.Printf("genmcp-server failed with %s\n", err.Error())
		}
		return
	}

	// Detached mode: spawn the same command without --detach flag
	cmdArgs := []string{"run", "-f", mcpFilePath}
	if serverConfigPath != "" {
		cmdArgs = append(cmdArgs, "-s", serverConfigPath)
	}
	cmd := exec.Command(os.Args[0], cmdArgs...)
	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start genmcp-server: %s\n", err.Error())
		return
	}

	// Save PID for stop command
	processManager := utils.GetProcessManager()
	err = processManager.SaveProcessId(mcpFilePath, cmd.Process.Pid)
	if err != nil {
		fmt.Printf("failed to save pid for genmcp server, to stop the server you will need to manually kill pid %d: %s\n", cmd.Process.Pid, err.Error())
	}

	fmt.Printf("successfully started gen-mcp server...\n")
}

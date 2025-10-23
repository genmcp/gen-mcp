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
	runCmd.Flags().StringVarP(&mcpFilePath, "file", "f", "mcpfile.yaml", "mcp file to read from (legacy format) or tool definitions file (new format)")
	runCmd.Flags().StringVarP(&serverConfigPath, "server-config", "s", "mcpserver.yaml", "server configuration file (new format only)")
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

	serverConfigPath, err = filepath.Abs(serverConfigPath)
	if err != nil {
		fmt.Printf("failed to resolve server config path: %s\n", err.Error())
		return
	}

	// Try to determine which format we're using
	var mcpServer *mcpfile.MCPServer
	
	// Check if server config file exists (new format)
	if _, err := os.Stat(serverConfigPath); err == nil {
		// New format: separate files
		fmt.Printf("Using new format with separate server config and tool definitions\n")
		
		serverConfig, err := mcpfile.ParseMCPServerConfig(serverConfigPath)
		if err != nil {
			fmt.Printf("invalid server config file: %s\n", err)
			return
		}

		// Check if tool definitions file exists
		var toolDefs *mcpfile.MCPToolDefinitions
		if _, err := os.Stat(mcpFilePath); err == nil {
			toolDefs, err = mcpfile.ParseMCPToolDefinitions(mcpFilePath)
			if err != nil {
				fmt.Printf("invalid tool definitions file: %s\n", err)
				return
			}
		} else {
			// No tool definitions file, create empty one
			toolDefs = &mcpfile.MCPToolDefinitions{
				Kind:        mcpfile.KindMCPToolDefinitions,
				FileVersion: mcpfile.MCPFileVersion,
			}
		}

		mcpServer = mcpfile.CombineConfigs(serverConfig, toolDefs)
	} else {
		// Legacy format: single file
		fmt.Printf("Using legacy format with single mcpfile.yaml\n")
		
		if _, err := os.Stat(mcpFilePath); err != nil {
			fmt.Printf("no file found at mcp file path\n")
			return
		}

		mcpFile, err := mcpfile.ParseMCPFile(mcpFilePath)
		if err != nil {
			fmt.Printf("invalid mcp file: %s\n", err)
			return
		}

		mcpServer = &mcpFile.MCPServer
	}

	err = mcpServer.Validate(invocation.InvocationValidator)
	if err != nil {
		fmt.Printf("invalid mcp configuration: %s\n", err)
		return
	}

	if mcpServer.Runtime.TransportProtocol == mcpfile.TransportProtocolStdio && detach {
		// TODO: re-enable this logging when we figure out logging w. stdio
		// fmt.Printf("cannot detach when running stdio transport\n")
		detach = false
	}

	if !detach {
		// Run servers directly in the current process
		// For new format, we need to write a temporary combined file for RunServers
		var configPath string
		if _, err := os.Stat(serverConfigPath); err == nil {
			// New format: create temporary combined config
			tmpFile, err := os.CreateTemp("", "mcpfile-*.yaml")
			if err != nil {
				fmt.Printf("failed to create temporary config file: %s\n", err.Error())
				return
			}
			defer os.Remove(tmpFile.Name())
			configPath = tmpFile.Name()

			// Convert MCPServer back to MCPFile for RunServers
			mcpFile := &mcpfile.MCPFile{
				FileVersion: mcpfile.MCPFileVersion,
				MCPServer:   *mcpServer,
			}

			data, err := mcpfile.SerializeMCPFile(mcpFile)
			if err != nil {
				fmt.Printf("failed to serialize config: %s\n", err.Error())
				return
			}

			if _, err := tmpFile.Write(data); err != nil {
				fmt.Printf("failed to write temporary config: %s\n", err.Error())
				return
			}
			tmpFile.Close()
		} else {
			// Legacy format: use original file
			configPath = mcpFilePath
		}

		err := mcpserver.RunServers(context.Background(), configPath)
		if err != nil {
			fmt.Printf("genmcp-server failed with %s\n", err.Error())
		}
		return
	}

	// Detached mode: spawn the same command without --detach flag
	cmdArgs := []string{"run", "-f", mcpFilePath}
	if _, err := os.Stat(serverConfigPath); err == nil {
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

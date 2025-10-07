package cli

import (
	"fmt"
	"os"

	"github.com/genmcp/gen-mcp/pkg/cli/utils"
	cliconverter "github.com/genmcp/gen-mcp/pkg/converter/cli"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(convertCliCmd)
	convertCliCmd.Flags().StringVarP(&mcpOutputPath, "out", "o", "mcpfile.yaml", "the path to write the mcp file to")
}

var mcpOutputPath string

var convertCliCmd = &cobra.Command{
	Use:   "convert-cli <command1> [command2] [command3] ...",
	Short: "Convert one or more CLI commands to a MCPFile",
	Args:  cobra.MinimumNArgs(1),
	Run:   executeConvertCliCmd,
}

func executeConvertCliCmd(cobraCmd *cobra.Command, args []string) {
	commandItems := []cliconverter.CommandItem{}

	for _, cliCommand := range args {
		_, err := cliconverter.ExtractCLICommandInfo(cliCommand, &commandItems)
		if err != nil {
			fmt.Printf("encountered errors while extracting cli command info for '%s': %s\n", cliCommand, err.Error())
			return
		}
	}

	mcpFile, err := cliconverter.ConvertCommandsToMCPFile(&commandItems)
	if err != nil {
		fmt.Printf("encountered errors while converting commands to mcp file: %s\n", err.Error())
		return
	}

	mcpFileBytes, err := yaml.Marshal(mcpFile)
	if err != nil {
		fmt.Printf("could not marshal mcp file: %s\n", err.Error())
		return
	}

	mcpFileBytes = utils.AppendSchemaHeader(mcpFileBytes)

	fmt.Printf("%s", string(mcpFileBytes))

	err = os.WriteFile(mcpOutputPath, mcpFileBytes, 0644)
	if err != nil {
		fmt.Printf("could not write mcpfile to file at path %s: %s", mcpOutputPath, err.Error())
		return
	}
}

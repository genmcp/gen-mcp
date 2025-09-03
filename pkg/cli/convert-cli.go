package cli

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/cli_converter"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(convertCliCmd)
	convertCliCmd.Flags().StringVarP(&mcpOutputPath, "out", "o", "mcpfile.yaml", "the path to write the mcp file to")
}

var mcpOutputPath string

var convertCliCmd = &cobra.Command{
	Use:   "convert-cli",
	Short: "Convert CLI command to a MCPFile",
	Args:  cobra.ExactArgs(1),
	Run:   executeConvertCliCmd,
}

func executeConvertCliCmd(cobraCmd *cobra.Command, args []string) {
	cliCommand := args[0]

	mcpFile, err := cli_converter.ConvertCliCommandToMcpFile(cliCommand)
	if err != nil {
		fmt.Printf("encountered errors while converting cli command to mcp file: %s\n", err.Error())
	}

	mcpFileBytes, err := yaml.Marshal(mcpFile)
	if err != nil {
		fmt.Printf("could not marshal mcp file: %s\n", err.Error())
		return
	}
	fmt.Printf("%s", string(mcpFileBytes))

	// err = os.WriteFile(outputPath, mcpFileBytes, 0644)
	// if err != nil {
	// 	fmt.Printf("could not write mcpfile to file at path %s: %s", outputPath, err.Error())
	// }
}

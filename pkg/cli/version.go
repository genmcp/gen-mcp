package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print GenMCP's version",
	Run:   executeVersionCmd,
}

func executeVersionCmd(cobraCmd *cobra.Command, args []string) {
	err := cobra.NoArgs(cobraCmd, args)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	fmt.Printf("genmcp version %s\n", cliVersion)
}

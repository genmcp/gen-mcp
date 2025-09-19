package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cliVersion string

var rootCmd = &cobra.Command{
	Use:   "genmcp",
	Short: "genmcp manages gen-mcp servers, and their configuration",
}

func Execute(version string) {
	if version == "" {
		cliVersion = "development"
	} else {
		cliVersion = version
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

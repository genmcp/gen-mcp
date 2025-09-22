package cli

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var cliVersion string

var rootCmd = &cobra.Command{
	Use:   "genmcp",
	Short: "genmcp manages gen-mcp servers, and their configuration",
}

func Execute(version string) {
	if version == "" {
		cliVersion = getDevVersion().String()
	} else {
		cliVersion = version
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type devVersion struct {
	commit               string
	hasUncommitedChanges bool
}

func (dv devVersion) String() string {
	if dv.hasUncommitedChanges {
		return fmt.Sprintf("development@%s+uncommitedChanges", dv.commit)
	}
	return fmt.Sprintf("development@%s", dv.commit)
}

func getDevVersion() devVersion {
	dv := devVersion{}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) >= 7 {
                   dv.commit = setting.Value[:7]
               } else {
                  dv.commit = setting.Value
               }
			case "vcs.modified":
				dv.hasUncommitedChanges = setting.Value == "true"
			}
		}
	}

	return dv
}

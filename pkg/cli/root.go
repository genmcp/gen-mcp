package cli

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var cliVersion string
var debugMode bool

var rootCmd = &cobra.Command{
	Use:   "genmcp",
	Short: "genmcp manages gen-mcp servers, and their configuration",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogging()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "v", false, "enable debug/verbose logging")
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

func setupLogging() {
	var level slog.Level
	if debugMode {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if debugMode {
		// Use text handler for more readable debug output
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		// Use JSON handler for production
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	if debugMode {
		slog.Debug("Debug mode enabled")
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

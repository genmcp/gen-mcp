package cli

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

// CommandRunner is a function type for running commands
type CommandRunner func(string) (string, error)

// DefaultCommandRunner is the default implementation that actually executes commands
var DefaultCommandRunner CommandRunner = runCommandImpl

// RunCommand is the public interface that uses the current CommandRunner
var RunCommand = DefaultCommandRunner

// runCommandImpl is the actual implementation that executes commands
func runCommandImpl(cmdStr string) (string, error) {

	if cmdStr == "" {
		return "", nil
	}

	// Parse the command string using shell-style parsing
	args, err := shlex.Split(cmdStr)
	if err != nil {
		return "", err
	}

	if len(args) == 0 {
		return "", nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return strings.TrimSpace(out.String()), err
}

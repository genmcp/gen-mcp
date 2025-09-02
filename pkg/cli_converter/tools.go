package cli_converter

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

// RunCommand takes a command line string, executes it locally, and returns its stdout output as a string.
// Uses shell-style parsing to handle quoted arguments properly.
// If the command fails, it returns an error.
func RunCommand(cmdStr string) (string, error) {
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

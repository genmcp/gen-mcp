package cli

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Skip("Skipping test: .env file not found")
	}
}

// mockMyFileToolHelp returns the mock help text for myfiletool command
func mockMyFileToolHelp() string {
	return `myfiletool - Advanced file management and processing tool

USAGE:
    myfiletool [OPTIONS] <SOURCE> <DESTINATION>

DESCRIPTION:
    A powerful file management utility that can copy, move, sync, and process files
    with advanced filtering and transformation capabilities. Supports batch operations,
    pattern matching, and various output formats.

ARGUMENTS:
    <SOURCE>        Source file or directory path (required)
    <DESTINATION>   Destination file or directory path (required)

OPTIONS:
    -v, --verbose           Enable verbose output with detailed progress information
    -f, --force             Force operation, overwrite existing files without prompting
    -r, --recursive         Process directories recursively
    -o, --output <FORMAT>   Specify output format (json, xml, csv, plain) [default: plain]
    -p, --pattern <REGEX>   Filter files using regular expression pattern
    -s, --size <SIZE>       Filter files by size (e.g., +100M, -1G, 500K)
    -t, --type <TYPE>       File type filter (file, dir, link, all) [default: all]
    -m, --mode <MODE>       Operation mode (copy, move, sync, analyze) [default: copy]
    -c, --config <FILE>     Use custom configuration file
    -q, --quiet             Suppress all output except errors
    -h, --help              Show this help message
    -V, --version           Show version information

EXAMPLES:
    myfiletool /home/user/docs /backup/docs
        Copy all files from docs to backup directory
    
    myfiletool --verbose --force --recursive /src /dst
        Recursively copy with verbose output and force overwrite
    
    myfiletool --pattern "\.txt$" --output json /data /processed
        Copy only .txt files and output results in JSON format
    
    myfiletool --mode analyze --size +10M /large-files /dev/null
        Analyze files larger than 10MB without copying

For more information, visit: https://github.com/example/myfiletool`
}

// setupMockRunCommand sets up a mock RunCommand function for testing
func setupMockRunCommand(t *testing.T) func() {
	t.Helper()

	// Store the original RunCommand function
	originalRunCommand := RunCommand

	// Set up the mock
	RunCommand = func(cmdStr string) (string, error) {
		if cmdStr == "myfiletool --help" {
			return mockMyFileToolHelp(), nil
		}
		// For other commands, return an error or call original
		return originalRunCommand(cmdStr)
	}

	// Return a cleanup function
	return func() {
		RunCommand = originalRunCommand
	}
}

func TestOpenAIConnection(t *testing.T) {
	setupTestEnv(t)

	content, err := RunInference(
		"You are a helpful assistant, who gives direct answers.",
		"what is 1 + 2 ?",
	)
	assert.NoError(t, err, "OpenAI inference should not fail")
	assert.NotNil(t, content)
	assert.Contains(t, content, "3")
}

func TestRunCommand(t *testing.T) {
	// Test basic command
	content, err := RunCommand("echo Hello World")
	assert.NoError(t, err, "RunCommand should not fail")
	assert.NotNil(t, content)
	assert.Equal(t, "Hello World", content)
}

func TestRunCommandWithQuotes(t *testing.T) {
	// Test command with quoted arguments (shlex handles this properly)
	content, err := RunCommand(`echo "Hello World with spaces"`)
	assert.NoError(t, err, "RunCommand should not fail with quoted args")
	assert.NotNil(t, content)
	assert.Equal(t, "Hello World with spaces", content)
}

func TestRunCommandEmpty(t *testing.T) {
	// Test empty command
	content, err := RunCommand("")
	assert.NoError(t, err, "Empty command should not fail")
	assert.Equal(t, "", content)
}

func TestDetectSubCommand(t *testing.T) {
	setupTestEnv(t)

	_, err := DetectSubCommand("")
	assert.Error(t, err, "Empty command should fail")

	is_subcommand, err := DetectSubCommand("ls")
	assert.NoError(t, err, "DetectSubCommand should not fail")
	assert.False(t, is_subcommand, "ls does not have subcommands")

	is_subcommand, err = DetectSubCommand("git")
	assert.NoError(t, err, "DetectSubCommand should not fail")
	assert.True(t, is_subcommand, "git should have subcommands")
}

func TestExtractSubCommands(t *testing.T) {
	setupTestEnv(t)

	_, err := ExtractSubCommands("")
	assert.Error(t, err, "Empty command should fail")

	subcommands, err := ExtractSubCommands("git")
	assert.NoError(t, err, "ExtractSubCommands should not fail")
	assert.NotNil(t, subcommands)
	assert.Greater(t, len(subcommands), 0)
	assert.Contains(t, subcommands, "add")
	assert.Contains(t, subcommands, "commit")
	assert.Contains(t, subcommands, "push")
	assert.Contains(t, subcommands, "pull")
	assert.Contains(t, subcommands, "status")
}

func TestExtractCommand(t *testing.T) {
	setupTestEnv(t)

	_, err := ExtractCommand("")
	assert.Error(t, err, "command is empty")

	// Set up mock for our custom command
	cleanup := setupMockRunCommand(t)
	defer cleanup()

	commandItem, err := ExtractCommand("myfiletool")
	assert.NoError(t, err, "ExtractCommand should not fail for valid command")
	assert.NotNil(t, commandItem)
	assert.Equal(t, "myfiletool", commandItem.Command)
	assert.NotEmpty(t, commandItem.Data.Description, "Command should have a description")

	// Verify arguments are parsed
	assert.Greater(t, len(commandItem.Data.Arguments), 0, "Command should have arguments")

	// Verify options/flags are parsed
	assert.Greater(t, len(commandItem.Data.Options), 0, "Command should have options")

	// Test that we found the expected arguments
	foundSourceArg := false
	foundDestArg := false
	for _, arg := range commandItem.Data.Arguments {
		if arg.Name == "source" || arg.Name == "SOURCE" {
			foundSourceArg = true
		}
		if arg.Name == "destination" || arg.Name == "DESTINATION" {
			foundDestArg = true
		}
	}
	assert.True(t, foundSourceArg, "Should find source argument")
	assert.True(t, foundDestArg, "Should find destination argument")
}

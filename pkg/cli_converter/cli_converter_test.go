package cli_converter

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

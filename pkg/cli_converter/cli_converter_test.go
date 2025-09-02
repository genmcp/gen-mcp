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

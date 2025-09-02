package cli_converter

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func NewOpenAIClient() openai.Client {
	key := os.Getenv("MODEL_KEY")
	base_url := os.Getenv("MODEL_BASE_URL")
	fmt.Println("base_url:", base_url)

	return openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(base_url),
	)
}

func RunInference(
	system_prompt string,
	user_prompt string,
) (string, error) {
	client := NewOpenAIClient()
	model := os.Getenv("MODEL_NAME")
	fmt.Println("model:", model)

	ctx := context.Background()

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system_prompt),
			openai.UserMessage(user_prompt),
		},
		Model: model,
	}

	chatCompletion, err := client.Chat.Completions.New(ctx, params)

	if err != nil {
		panic(err.Error())
	}

	content := chatCompletion.Choices[0].Message.Content
	return content, nil
}

func DetectSubCommand(cliCommand string) (bool, error) {
	// TODO: Implement subcommand detection logic
	return false, nil
}

package cli_converter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
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
	//Subcommand detection logic
	user_prompt, err := RunCommand(cliCommand + " --help")
	if err != nil {
		panic(err.Error())
	}

	client := NewOpenAIClient()
	ctx := context.Background()

	schema_param := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   "is_sub_command",
		Schema: IsSubCommandResponseSchema,
		Strict: openai.Bool(true),
	}

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(IsSubCommandPrompt),
			openai.UserMessage(user_prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schema_param},
		},
		Model:       os.Getenv("MODEL_NAME"),
		Temperature: param.Opt[float64]{Value: 0.0},
		TopP:        param.Opt[float64]{Value: 1.0},
		MaxTokens:   param.Opt[int64]{Value: 4096},
	}

	paramsJSON, err := params.MarshalJSON()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("params:", string(paramsJSON))

	chat, err := client.Chat.Completions.New(ctx, params)

	if err != nil {
		panic(err.Error())
	}

	var is_sub_command IsSubCommand
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &is_sub_command)
	if err != nil {
		panic(err.Error())
	}

	return is_sub_command.Exists, nil
}

package cli_converter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
)

func NewOpenAIClient() openai.Client {
	key := os.Getenv("MODEL_KEY")
	base_url := os.Getenv("MODEL_BASE_URL")
	// fmt.Println("base_url:", base_url)

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
	// fmt.Println("model:", model)

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
	cliCommand = strings.TrimSpace(cliCommand)
	if len(cliCommand) == 0 {
		return false, errors.New("command is empty")
	}

	//Subcommand detection logic
	user_prompt, err := RunCommand(cliCommand + " --help")
	if err != nil {
		panic(err.Error())
	}

	client := NewOpenAIClient()
	ctx := context.Background()

	schema_param := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   "sub_command",
		Schema: IsSubCommandResponseSchema,
		Strict: openai.Bool(true),
	}

	user_message := "### Command:\n" + cliCommand + "\n\n### Man Page:\n" + user_prompt

	// fmt.Println("User Message:", user_message)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(IsSubCommandPrompt),
			openai.UserMessage(user_message),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schema_param},
		},
		Model:       os.Getenv("MODEL_NAME"),
		Temperature: param.Opt[float64]{Value: 0.0},
		TopP:        param.Opt[float64]{Value: 1.0},
		MaxTokens:   param.Opt[int64]{Value: 4096},
	}

	// paramsJSON, err := params.MarshalJSON()
	// if err != nil {
	// 	panic(err.Error())
	// }
	//fmt.Println("Params JSON:", string(paramsJSON))

	chat, err := client.Chat.Completions.New(ctx, params)
	// fmt.Println("LLM Response:", chat.Choices[0].Message.Content)

	if err != nil {
		panic(err.Error())
	}

	var is_sub_command IsSubCommand
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &is_sub_command)
	if err != nil {
		panic(err.Error())
	}

	return is_sub_command.Bool_Value, nil
}

func ExtractSubCommands(cliCommand string) ([]string, error) {
	cliCommand = strings.TrimSpace(cliCommand)
	if len(cliCommand) == 0 {
		return []string{}, errors.New("command is empty")
	}

	//Subcommand detection logic
	user_prompt, err := RunCommand(cliCommand + " --help")
	if err != nil {
		panic(err.Error())
	}

	client := NewOpenAIClient()
	ctx := context.Background()

	schema_param := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   "sub_commands",
		Schema: SubCommandsResponseSchema,
		Strict: openai.Bool(true),
	}

	user_message := "### Query:\n" + user_prompt

	// fmt.Println("User Message:", user_message)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(ExtractSubCommandsPrompt),
			openai.UserMessage(user_message),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schema_param},
		},
		Model:       os.Getenv("MODEL_NAME"),
		Temperature: param.Opt[float64]{Value: 0.0},
		TopP:        param.Opt[float64]{Value: 1.0},
		MaxTokens:   param.Opt[int64]{Value: 4096},
	}

	chat, err := client.Chat.Completions.New(ctx, params)
	// fmt.Println("LLM Response:", chat.Choices[0].Message.Content)

	if err != nil {
		panic(err.Error())
	}

	var subCommands SubCommands
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &subCommands)
	if err != nil {
		panic(err.Error())
	}

	return subCommands.Commands, nil
}

func ExtractCommand(cliCommand string) (CommandItem, error) {
	cliCommand = strings.TrimSpace(cliCommand)
	if len(cliCommand) == 0 {
		return CommandItem{}, errors.New("command is empty")
	}

	user_prompt, err := RunCommand(cliCommand + " --help")
	if err != nil {
		panic(err.Error())
	}

	// remove shorthand flags
	re := regexp.MustCompile(`-\w\b`)
	user_prompt = re.ReplaceAllString(user_prompt, "")

	client := NewOpenAIClient()
	ctx := context.Background()

	schema_param := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   "extract_command",
		Schema: CommandResponseSchema,
		Strict: openai.Bool(true),
	}

	user_message := "### Command:\n" + cliCommand + "\n\n### Man Page:\n" + user_prompt

	// fmt.Println("User Message:", user_message)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(ExtractCommandPrompt),
			openai.UserMessage(user_message),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schema_param},
		},
		Model:       os.Getenv("MODEL_NAME"),
		Temperature: param.Opt[float64]{Value: 0.0},
		TopP:        param.Opt[float64]{Value: 1.0},
		MaxTokens:   param.Opt[int64]{Value: 4096},
	}

	chat, err := client.Chat.Completions.New(ctx, params)
	// fmt.Println("LLM Response:", chat.Choices[0].Message.Content)

	if err != nil {
		panic(err.Error())
	}

	var command Command
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &command)
	if err != nil {
		panic(err.Error())
	}

	postProcessOptions(&command.Options)

	return CommandItem{
		Command: cliCommand,
		Data:    command,
	}, nil
}

func postProcessOptions(options *[]Option) {
	for i, option := range *options {
		option.Flag = strings.TrimSpace(option.Flag)
		re := regexp.MustCompile(`--[a-zA-Z0-9\-]+`)
		matches := re.FindAllString(option.Flag, -1)
		if len(matches) > 0 {
			// No, this will not update the original objects in the main function because 'option' is a copy of each element in the slice.
			// To update the original slice elements, you need to use the index to access and modify the elements directly:
			(*options)[i].Flag = matches[0]
		}
	}
}

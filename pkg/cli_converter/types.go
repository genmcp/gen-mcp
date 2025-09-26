package cli_converter

import (
	"github.com/invopop/jsonschema"
)

func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

type IsSubCommand struct {
	Bool_Value bool `json:"bool_value"`
}

type SubCommands struct {
	Commands []string `json:"commands"`
}

type Argument struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"`
}

type Option struct {
	Flag        string `json:"flag"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type Command struct {
	Description string     `json:"description"`
	Arguments   []Argument `json:"arguments"`
	Options     []Option   `json:"options"`
}

type CommandItem struct {
	Command string  `json:"command"`
	Data    Command `json:"data"`
}

var IsSubCommandResponseSchema = GenerateSchema[IsSubCommand]()
var SubCommandsResponseSchema = GenerateSchema[SubCommands]()
var CommandResponseSchema = GenerateSchema[Command]()

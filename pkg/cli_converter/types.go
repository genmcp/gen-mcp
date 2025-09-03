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

var IsSubCommandResponseSchema = GenerateSchema[IsSubCommand]()

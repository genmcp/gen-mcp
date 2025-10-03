package utils

import (
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/google/jsonschema-go/jsonschema"
)

func FormatStringForParam(paramName string, schema *jsonschema.Schema) (string, error) {
	schema, err := lookupParam(paramName, schema)
	if err != nil {
		return "", err
	}

	switch schema.Type {
	case invocation.JsonSchemaTypeArray, invocation.JsonSchemaTypeNull, invocation.JsonSchemaTypeObject:
		return "%v", nil
	case invocation.JsonSchemaTypeBoolean:
		return "%b", nil
	case invocation.JsonSchemaTypeInteger:
		return "%d", nil
	case invocation.JsonSchemaTypeNumber:
		return "%f", nil
	case invocation.JsonSchemaTypeString:
		return "%s", nil
	default:
		return "", fmt.Errorf("unknown json schema for type: '%s'", schema.Type)
	}
}

func lookupParam(paramName string, schema *jsonschema.Schema) (*jsonschema.Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("input schema is nil")
	}
	path := strings.Split(paramName, ".")
	currentSchema := schema
	var ok bool

	for _, p := range path {
		currentSchema, ok = currentSchema.Properties[p]
		if !ok {
			return nil, fmt.Errorf("path parameter %s has no corresponding property in the input schema", paramName)
		}
	}

	return currentSchema, nil
}

package invocation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	JsonSchemaTypeObject  = "object"
	JsonSchemaTypeNumber  = "number"
	JsonSchemaTypeInteger = "integer"
	JsonSchemaTypeString  = "string"
	JsonSchemaTypeArray   = "array"
	JsonSchemaTypeBoolean = "boolean"
	JsonSchemaTypeNull    = "null"
)

// Builder is an interface used to build any objects needed to invoke a tool as
// the request JSON is parsed. This avoids extra allocations/passes through the
// parsed map[string]any, which possibly contains nested maps
type Builder interface {
	// SetField provides the builder with the parsed value, as well as the json dot separated path where the value comes from
	SetField(path string, value any)

	// GetResult returns the final builder resulting object
	GetResult() (any, error)
}

type DynamicJson struct {
	Builders []Builder
}

func (dj *DynamicJson) ParseJson(data []byte, schema *jsonschema.Schema) (map[string]any, error) {
	if schema.Type != JsonSchemaTypeObject {
		return nil, fmt.Errorf("error parsing json from schema: top level schema type must be object")
	}

	return dj.parseObject(data, schema, "")
}

func (dj *DynamicJson) parseObject(data []byte, currentSchema *jsonschema.Schema, currentPath string) (map[string]any, error) {
	var rawFieldMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawFieldMap); err != nil {
		return nil, fmt.Errorf("invalid json object format: %w", err)
	}

	resultMap := make(map[string]any, len(rawFieldMap))

	var parseErr error
	for fieldName, rawMessage := range rawFieldMap {
		fieldSchema, isDefined := currentSchema.Properties[fieldName]

		newPath := fieldName
		if currentPath != "" {
			newPath = strings.Join([]string{currentPath, newPath}, ".")
		}

		if isDefined {

			parsedValue, err := dj.parseValue(rawMessage, fieldSchema, newPath)
			if err != nil {
				parseErr = errors.Join(parseErr, err)
				continue
			}
			resultMap[fieldName] = parsedValue
		} else {
			if currentSchema.AdditionalProperties == nil {
				parseErr = errors.Join(parseErr, fmt.Errorf("extraneous field found in json at path %s: %s", currentPath, fieldName))
				continue
			}

			var genericValue any
			if err := json.Unmarshal(rawMessage, &genericValue); err != nil {
				parseErr = errors.Join(parseErr, fmt.Errorf("error parsing additional property %s: %w", fieldName, err))
				continue
			}

			resultMap[fieldName] = genericValue
			for _, builder := range dj.Builders {
				builder.SetField(newPath, genericValue)
			}
		}
	}

	for _, fieldName := range currentSchema.Required {
		if _, ok := resultMap[fieldName]; !ok {
			parseErr = errors.Join(parseErr, fmt.Errorf("missing required field: %s", fieldName))
		}
	}

	return resultMap, parseErr
}

func (dj *DynamicJson) parseArray(rawArray []byte, schema *jsonschema.Schema, currentPath string) ([]any, error) {
	if schema.Items == nil {
		var result []any
		if err := json.Unmarshal(rawArray, &result); err != nil {
			return nil, fmt.Errorf("expected a json array but got %s: %w", string(rawArray), err)
		}
		return result, nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(rawArray, &rawItems); err != nil {
		return nil, fmt.Errorf("expected a json array but got %s: %w", string(rawArray), err)
	}

	result := make([]any, 0, len(rawItems))
	var itemErr error
	for i, rawItem := range rawItems {
		itemPath := fmt.Sprintf("%s[%d]", currentPath, i)
		parsedItem, err := dj.parseValue(rawItem, schema.Items, itemPath)
		if err != nil {
			itemErr = errors.Join(itemErr, err)
		}
		result = append(result, parsedItem)
	}

	return result, itemErr
}

func (dj *DynamicJson) parseValue(rawMessage json.RawMessage, schema *jsonschema.Schema, currentPath string) (any, error) {
	var result any
	isLeafNode := false

	switch schema.Type {
	case JsonSchemaTypeObject:
		obj, err := dj.parseObject(rawMessage, schema, currentPath)
		if err != nil {
			return nil, err
		}
		result = obj
	case JsonSchemaTypeArray:
		arr, err := dj.parseArray(rawMessage, schema, currentPath)
		if err != nil {
			return nil, err
		}
		result = arr
	case JsonSchemaTypeString:
		var s string
		if err := json.Unmarshal(rawMessage, &s); err != nil {
			return nil, fmt.Errorf("expected a string, but got %s: %w", string(rawMessage), err)
		}
		result = s
		isLeafNode = true
	case JsonSchemaTypeInteger:
		var i int
		if err := json.Unmarshal(rawMessage, &i); err != nil {
			return nil, fmt.Errorf("expected an integer, but got %s: %w", string(rawMessage), err)
		}
		result = i
		isLeafNode = true
	case JsonSchemaTypeNumber:
		var f float64
		if err := json.Unmarshal(rawMessage, &f); err != nil {
			return nil, fmt.Errorf("expected a number, but got %s: %w", string(rawMessage), err)
		}
		result = f
		isLeafNode = true
	case JsonSchemaTypeBoolean:
		var b bool
		if err := json.Unmarshal(rawMessage, &b); err != nil {
			return nil, fmt.Errorf("expected a boolean, but got %s: %w", string(rawMessage), err)
		}
		result = b
		isLeafNode = true
	default:
		return nil, fmt.Errorf("unsupported schema type: %s", schema.Type)
	}

	if isLeafNode {
		for _, builder := range dj.Builders {
			builder.SetField(currentPath, result)
		}
	}

	return result, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/genmcp/gen-mcp/pkg/invocation/cli"
	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	googlejsonschema "github.com/google/jsonschema-go/jsonschema"
)

// schemaType holds the type information and its corresponding Go comment location.
type schemaType struct {
	Type interface{}
	Base string
	Path string
}

// fixRequiredFields post-processes the schema to fix required fields based on struct tags.
// invopop/jsonschema doesn't understand google/jsonschema-go's "required"/"optional" tags,
// so we need to read them ourselves and fix the generated schema.
func fixRequiredFields(schema *jsonschema.Schema, types []schemaType) {
	if schema.Definitions == nil {
		return
	}

	// For each type we reflected, examine its struct tags and fix the required fields
	for _, item := range types {
		t := reflect.TypeOf(item.Type)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		// Process this type and any nested types
		fixRequiredFieldsForType(schema, t)
	}
}

func fixRequiredFieldsForType(schema *jsonschema.Schema, t reflect.Type) {
	if t.Kind() != reflect.Struct {
		return
	}

	typeName := t.Name()
	def, exists := schema.Definitions[typeName]
	if !exists {
		return
	}

	// Read the actual struct tags to determine what should be required
	requiredFields := []string{}
	explicitlyOptional := make(map[string]bool)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Get the JSON field name, handling inline embeds
		jsonName := jsonTag
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			parts := strings.Split(jsonTag, ",")
			jsonName = parts[0]
			// Check if this is an inline embed
			for _, part := range parts[1:] {
				if part == "inline" {
					jsonName = "" // Mark as inline
					break
				}
			}
		}

		jsonschemaTag := field.Tag.Get("jsonschema")

		// Handle inline embeds - merge their required fields
		if jsonName == "" && strings.Contains(jsonTag, "inline") {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct && fieldType != t {
				fixRequiredFieldsForType(schema, fieldType)

				nestedTypeName := fieldType.Name()
				if nestedDef, nestedExists := schema.Definitions[nestedTypeName]; nestedExists {
					requiredFields = append(requiredFields, nestedDef.Required...)
				}
			}
		} else if jsonName != "" {
			if strings.Contains(jsonschemaTag, "required") {
				requiredFields = append(requiredFields, jsonName)
			} else if strings.Contains(jsonschemaTag, "optional") {
				explicitlyOptional[jsonName] = true
			}

			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct && fieldType != t {
				fixRequiredFieldsForType(schema, fieldType)
			}
		}
	}

	// Merge with existing required fields, preserving those not explicitly optional
	finalRequired := make(map[string]bool)

	// Add existing required fields unless they're explicitly optional
	for _, existing := range def.Required {
		if !explicitlyOptional[existing] {
			finalRequired[existing] = true
		}
	}

	// Add new required fields from struct tags
	for _, newRequired := range requiredFields {
		finalRequired[newRequired] = true
	}

	// Convert back to slice and update
	def.Required = make([]string, 0, len(finalRequired))
	for fieldName := range finalRequired {
		def.Required = append(def.Required, fieldName)
	}
	// Sort required fields to ensure deterministic output
	sort.Strings(def.Required)
}

// createJSONSchemaMetaSchema returns a JSON Schema that describes JSON Schema itself.
// This provides better IDE autocomplete and validation for inputSchema/outputSchema fields.
func createJSONSchemaMetaSchema() *jsonschema.Schema {
	// Create a simple schema that describes common JSON Schema properties
	// without deep recursion to avoid validation issues
	schema := &jsonschema.Schema{
		Type:                 "object",
		Description:          "A JSON Schema object defining the structure and validation rules.",
		AdditionalProperties: jsonschema.TrueSchema, // Allow any JSON Schema properties
	}

	schema.Properties = jsonschema.NewProperties()

	schema.Properties.Set("type", &jsonschema.Schema{
		Description: "The type of the value. Can be a single type or an array of types.",
		OneOf: []*jsonschema.Schema{
			{Type: "string", Enum: []any{"string", "number", "integer", "boolean", "object", "array", "null"}},
			{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
		},
	})

	schema.Properties.Set("properties", &jsonschema.Schema{
		Type:                 "object",
		Description:          "An object where each key is a property name and each value is a schema.",
		AdditionalProperties: jsonschema.TrueSchema,
	})

	schema.Properties.Set("required", &jsonschema.Schema{
		Type:        "array",
		Description: "An array of required property names.",
		Items:       &jsonschema.Schema{Type: "string"},
	})

	schema.Properties.Set("additionalProperties", &jsonschema.Schema{
		Description: "Whether additional properties are allowed (boolean) or a schema they must match (object).",
	})

	schema.Properties.Set("items", &jsonschema.Schema{
		Description: "Schema for array items. Can be a single schema or an array of schemas for tuple validation.",
	})

	schema.Properties.Set("description", &jsonschema.Schema{
		Type:        "string",
		Description: "A description of the schema's purpose.",
	})

	schema.Properties.Set("title", &jsonschema.Schema{
		Type:        "string",
		Description: "A short title for the schema.",
	})

	schema.Properties.Set("default", &jsonschema.Schema{
		Description: "The default value for this schema.",
	})

	// Enum
	schema.Properties.Set("enum", &jsonschema.Schema{
		Type:        "array",
		Description: "An array of allowed values.",
	})

	// Const
	schema.Properties.Set("const", &jsonschema.Schema{
		Description: "A constant value that must match exactly.",
	})

	schema.Properties.Set("$ref", &jsonschema.Schema{
		Type:        "string",
		Description: "A reference to another schema definition (e.g., '#/$defs/MyType').",
	})

	schema.Properties.Set("$defs", &jsonschema.Schema{
		Type:                 "object",
		Description:          "A container for reusable schema definitions.",
		AdditionalProperties: jsonschema.TrueSchema,
	})

	schema.Properties.Set("definitions", &jsonschema.Schema{
		Type:                 "object",
		Description:          "Legacy container for reusable schema definitions. Use $defs instead.",
		AdditionalProperties: jsonschema.TrueSchema,
	})

	schema.Properties.Set("allOf", &jsonschema.Schema{
		Type:        "array",
		Description: "Must match all of the schemas in the array.",
	})

	schema.Properties.Set("anyOf", &jsonschema.Schema{
		Type:        "array",
		Description: "Must match at least one of the schemas in the array.",
	})

	schema.Properties.Set("oneOf", &jsonschema.Schema{
		Type:        "array",
		Description: "Must match exactly one of the schemas in the array.",
	})

	schema.Properties.Set("not", &jsonschema.Schema{
		Description: "Must not match this schema.",
	})

	schema.Properties.Set("minimum", &jsonschema.Schema{
		Type:        "number",
		Description: "Minimum value for numbers.",
	})

	schema.Properties.Set("maximum", &jsonschema.Schema{
		Type:        "number",
		Description: "Maximum value for numbers.",
	})

	schema.Properties.Set("exclusiveMinimum", &jsonschema.Schema{
		Type:        "number",
		Description: "Exclusive minimum value for numbers.",
	})

	schema.Properties.Set("exclusiveMaximum", &jsonschema.Schema{
		Type:        "number",
		Description: "Exclusive maximum value for numbers.",
	})

	schema.Properties.Set("multipleOf", &jsonschema.Schema{
		Type:        "number",
		Description: "Value must be a multiple of this number.",
	})

	schema.Properties.Set("minLength", &jsonschema.Schema{
		Type:        "integer",
		Description: "Minimum length for strings.",
	})

	schema.Properties.Set("maxLength", &jsonschema.Schema{
		Type:        "integer",
		Description: "Maximum length for strings.",
	})

	schema.Properties.Set("pattern", &jsonschema.Schema{
		Type:        "string",
		Description: "Regular expression pattern that strings must match.",
	})

	schema.Properties.Set("format", &jsonschema.Schema{
		Type:        "string",
		Description: "Format hint for strings (e.g., 'email', 'uri', 'date-time', 'uuid').",
	})

	schema.Properties.Set("minItems", &jsonschema.Schema{
		Type:        "integer",
		Description: "Minimum number of items in arrays.",
	})

	schema.Properties.Set("maxItems", &jsonschema.Schema{
		Type:        "integer",
		Description: "Maximum number of items in arrays.",
	})

	schema.Properties.Set("uniqueItems", &jsonschema.Schema{
		Type:        "boolean",
		Description: "Whether array items must be unique.",
	})

	schema.Properties.Set("minProperties", &jsonschema.Schema{
		Type:        "integer",
		Description: "Minimum number of properties in objects.",
	})

	schema.Properties.Set("maxProperties", &jsonschema.Schema{
		Type:        "integer",
		Description: "Maximum number of properties in objects.",
	})

	schema.Properties.Set("patternProperties", &jsonschema.Schema{
		Type:                 "object",
		Description:          "Schemas for properties matching regex patterns.",
		AdditionalProperties: jsonschema.TrueSchema,
	})

	return schema
}

func main() {
	// Use a slice to guarantee the processing order.
	// mcpfile.MCPFile will be processed first.
	types := []schemaType{
		{
			Type: &mcpfile.MCPFile{},
			Base: "github.com/genmcp/gen-mcp/pkg/mcpfile",
			Path: "../../pkg/mcpfile",
		},
		{
			Type: &http.HttpInvocationConfig{},
			Base: "github.com/genmcp/gen-mcp/pkg/invocation",
			Path: "../../pkg/invocation",
		},
		{
			Type: &cli.CliInvocationConfig{},
			Base: "github.com/genmcp/gen-mcp/pkg/invocation",
			Path: "../../pkg/invocation",
		},
		{
			Type: &extends.ExtendsConfig{},
			Base: "github.com/genmcp/gen-mcp/pkg/invocation",
			Path: "../../pkg/invocation",
		},
	}

	var schema *jsonschema.Schema

	for _, item := range types {
		reflector := new(jsonschema.Reflector)

		// Don't automatically require all properties - we'll use struct tags to determine this
		reflector.RequiredFromJSONSchemaTags = true

		// WORKAROUND: Handle google/jsonschema-go Schema type
		// invopop/jsonschema can't properly reflect google's Schema because it uses
		// json:"-" tags on the Type field. Instead, we return a detailed meta-schema
		// that describes JSON Schema itself, providing better IDE autocomplete.
		reflector.Mapper = func(t reflect.Type) *jsonschema.Schema {
			if t == reflect.TypeOf(&googlejsonschema.Schema{}) || t == reflect.TypeOf(googlejsonschema.Schema{}) {
				return createJSONSchemaMetaSchema()
			}
			return nil
		}

		if err := reflector.AddGoComments(item.Base, item.Path); err != nil {
			log.Fatalf("Failed to add Go comments: %v", err)
		}
		currentSchema := reflector.Reflect(item.Type)

		if schema == nil {
			schema = currentSchema
		} else {
			for k, v := range currentSchema.Definitions {
				// Avoid overwriting existing definitions
				if _, exists := schema.Definitions[k]; exists {
					fmt.Printf("Skipping existing definition: %s\n", k)
					continue
				}
				fmt.Printf("Adding definition: %s\n", k)
				schema.Definitions[k] = v
			}
		}
	}

	// Fix required fields by reading the actual struct tags from our Go types
	fixRequiredFields(schema, types)

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal schema: %v", err)
	}

	// Build paths
	specsDir := filepath.Join("..", "..", "specs")
	versionedFile := filepath.Join(specsDir, fmt.Sprintf("mcpfile-schema-%s.json", mcpfile.MCPFileVersion))
	latestFile := filepath.Join(specsDir, "mcpfile-schema.json")

	// Write versioned schema
	if err := os.WriteFile(versionedFile, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write versioned schema: %v", err)
	}

	// Write latest schema (same content)
	if err := os.WriteFile(latestFile, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write latest schema: %v", err)
	}
}

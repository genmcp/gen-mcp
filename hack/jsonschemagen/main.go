package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/genmcp/gen-mcp/pkg/invocation/cli"
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
			Type: &http.HttpInvocationData{},
			Base: "github.com/genmcp/gen-mcp/pkg/invocation",
			Path: "../../pkg/invocation",
		},
		{
			Type: &cli.CliInvocationData{},
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
		// json:"-" tags on the Type field. Instead, we return a simple object schema
		// that allows any properties (which is what we want for inputSchema/outputSchema).
		reflector.Mapper = func(t reflect.Type) *jsonschema.Schema {
			if t == reflect.TypeOf(&googlejsonschema.Schema{}) || t == reflect.TypeOf(googlejsonschema.Schema{}) {
				return &jsonschema.Schema{
					Type: "object",
					// By not setting AdditionalProperties, we allow any properties
					// This makes inputSchema/outputSchema accept any valid JSON Schema
				}
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

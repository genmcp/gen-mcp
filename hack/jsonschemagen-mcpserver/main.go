package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

// schemaType holds the type information and its corresponding Go comment location.
type schemaType struct {
	Type interface{}
	Base string
	Path string
}

func main() {
	// Generate schema for MCPServerConfig
	types := []schemaType{
		{
			Type: &mcpfile.MCPServerConfig{},
			Base: "github.com/genmcp/gen-mcp/pkg/mcpfile",
			Path: "../../pkg/mcpfile",
		},
	}

	var schema *jsonschema.Schema

	for _, item := range types {
		reflector := new(jsonschema.Reflector)
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

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal schema: %v", err)
	}

	// Build paths
	specsDir := filepath.Join("..", "..", "specs")
	versionedFile := filepath.Join(specsDir, fmt.Sprintf("mcpserver-schema-%s.json", mcpfile.MCPFileVersion))
	latestFile := filepath.Join(specsDir, "mcpserver-schema.json")

	// Write versioned schema
	if err := os.WriteFile(versionedFile, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write versioned schema: %v", err)
	}

	// Write latest schema (same content)
	if err := os.WriteFile(latestFile, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write latest schema: %v", err)
	}

	fmt.Printf("Generated mcpserver schema version %s\n", mcpfile.MCPFileVersion)
}

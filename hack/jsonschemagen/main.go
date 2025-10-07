package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func main() {
	reflector := new(jsonschema.Reflector)
	if err := reflector.AddGoComments("github.com/genmcp/gen-mcp/pkg/mcpfile", "./../../pkg/mcpfile"); err != nil {
		log.Fatalf("Failed to add Go comments: %v", err)
	}

	schema := reflector.Reflect(&mcpfile.MCPFile{})

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal schema: %v", err)
	}

	// write the schema to a file
	outPath := filepath.Join("..", "..", "mcpfile-schema.json")
	if err := os.WriteFile(outPath, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write schema to file: %v", err)
	}
}

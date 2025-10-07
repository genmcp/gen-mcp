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

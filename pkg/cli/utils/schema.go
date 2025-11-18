package utils

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/config"
)

// AppendToolDefinitionsSchemaHeader appends the schema header for tool definitions files
func AppendToolDefinitionsSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpfile-schema-%s.json\n\n",
		config.SchemaVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

// AppendServerConfigSchemaHeader appends the schema header for server config files
func AppendServerConfigSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpserver-schema-%s.json\n\n",
		config.SchemaVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

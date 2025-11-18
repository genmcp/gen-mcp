package utils

import (
	"fmt"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
)

// AppendToolDefinitionsSchemaHeader appends the schema header for tool definitions files
func AppendToolDefinitionsSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/tool-definitions-schema-%s.json\n\n",
		definitions.SchemaVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

// AppendServerConfigSchemaHeader appends the schema header for server config files
func AppendServerConfigSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/server-config-schema-%s.json\n\n",
		definitions.SchemaVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

// AppendSchemaHeader is deprecated. Use AppendToolDefinitionsSchemaHeader or AppendServerConfigSchemaHeader instead.
// Kept for backward compatibility.
func AppendSchemaHeader(bytes []byte) []byte {
	return AppendToolDefinitionsSchemaHeader(bytes)
}

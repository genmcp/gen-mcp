package utils

import (
	"fmt"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
)

func AppendSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpfile-schema-%s.json\n\n",
		definitions.SchemaVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

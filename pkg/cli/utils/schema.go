package utils

import (
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func AppendSchemaHeader(bytes []byte) []byte {
	schemaHeader := fmt.Sprintf(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/genmcp/gen-mcp/refs/heads/main/specs/mcpfile-schema-%s.json\n\n",
		mcpfile.MCPFileVersion,
	)

	return append([]byte(schemaHeader), bytes...)
}

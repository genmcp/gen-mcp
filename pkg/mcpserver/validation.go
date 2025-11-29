package mcpserver

import (
	"errors"
	"fmt"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
)

// Validate validates the MCPServer configuration
func (s *MCPServer) Validate(invocationValidator definitions.InvocationValidator) error {
	var err error = nil

	if toolDefsErr := s.MCPToolDefinitions.Validate(invocationValidator); toolDefsErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server tool definitions: %w", toolDefsErr))
	}

	if serverConfigErr := s.MCPServerConfig.Validate(); serverConfigErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server config: %w", serverConfigErr))
	}

	return err
}

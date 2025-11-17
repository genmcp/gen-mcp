package mcpserver

import (
	"errors"
	"fmt"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// InvocationValidator validates invocation primitives
type InvocationValidator func(primitive invocation.Primitive) error

// Validate validates the MCPServer configuration
func (s *MCPServer) Validate(invocationValidator InvocationValidator) error {
	var err error = nil

	// Convert to the validator types expected by the subpackages
	definitionsValidator := definitions.InvocationValidator(invocationValidator)
	serverconfigValidator := serverconfig.InvocationValidator(invocationValidator)

	if toolDefsErr := s.MCPToolDefinitions.Validate(definitionsValidator); toolDefsErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server tool definitions: %w", toolDefsErr))
	}

	if serverConfigErr := s.MCPServerConfig.Validate(serverconfigValidator); serverConfigErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server config: %w", serverConfigErr))
	}

	return err
}

package mcpfile

import (
	"errors"
	"fmt"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// TODO: duplicate type
type InvocationValidator func(primitive invocation.Primitive) error

func (s *MCPServer) Validate(invocationValidator InvocationValidator) error {
	var err error = nil

	// TODO: conversion to duplicate type
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

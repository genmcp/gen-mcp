package server

import (
	"errors"
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

// TODO: remove type
type InvocationValidator func(primitive invocation.Primitive) error

// TODO: remove param
func (m *MCPServerConfigFile) Validate() error {
	var err error = nil
	if m.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid mcpfile: name is required"))
	}

	if m.Version == "" {
		err = errors.Join(err, fmt.Errorf("invalid mcpfile: version is required"))
	}

	if runtimeErr := m.Runtime.Validate(); runtimeErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid mcpfile, runtime is invalid: %w", runtimeErr))
	}

	return err
}

func (s *MCPServerConfig) Validate() error {
	var err error = nil
	if s.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid server: name is required"))
	}

	if s.Version == "" {
		err = errors.Join(err, fmt.Errorf("invalid server: version is required"))
	}

	if runtimeErr := s.Runtime.Validate(); runtimeErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server, runtime is invalid: %w", err))
	}

	return err
}

func (r *ServerRuntime) Validate() error {
	var err error = nil
	if r.TransportProtocol != TransportProtocolStdio && r.TransportProtocol != TransportProtocolStreamableHttp {
		err = errors.Join(
			err,
			fmt.Errorf(
				"invalid runtime: transport protocol must be one of (%s, %s), received %s",
				TransportProtocolStdio,
				TransportProtocolStreamableHttp,
				r.TransportProtocol,
			),
		)
	}

	if r.TransportProtocol == TransportProtocolStreamableHttp {
		if r.StreamableHTTPConfig == nil {
			err = errors.Join(
				err,
				fmt.Errorf(
					"transportProtocol is %s, but streamableHttpConfig is not set",
					TransportProtocolStreamableHttp,
				),
			)
		}

		if r.StreamableHTTPConfig.Port <= 0 {
			err = errors.Join(err, fmt.Errorf("streamableHttpConfig.port must be greater than 0"))
		}

		if r.StreamableHTTPConfig.BasePath == "" {
			r.StreamableHTTPConfig.BasePath = DefaultBasePath
		}
	}

	return err
}

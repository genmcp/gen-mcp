package server

import (
	"errors"
	"fmt"
)

func (m *MCPServerConfigFile) Validate() error {
	var err error = nil

	if runtimeErr := m.Runtime.Validate(); runtimeErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server config file, runtime is invalid: %w", runtimeErr))
	}

	return err
}

func (s *MCPServerConfig) Validate() error {
	var err error = nil

	if runtimeErr := s.Runtime.Validate(); runtimeErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid server, runtime is invalid: %w", runtimeErr))
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
		} else {
			// Only validate fields if StreamableHTTPConfig is set
			if r.StreamableHTTPConfig.Port <= 0 {
				err = errors.Join(err, fmt.Errorf("streamableHttpConfig.port must be greater than 0"))
			}
		}
	}

	return err
}

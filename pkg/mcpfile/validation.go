package mcpfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type InvocationValidator func(invocationType string, data json.RawMessage, tool *Tool) error

func (t *Tool) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if t.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool: name is required"))
	}

	if t.Description == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool: description is required"))
	}

	if t.InputSchema == nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputSchema is required"))
	} else {
		resolved, schemaErr := t.InputSchema.Resolve(nil)
		if schemaErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid tool: inputSchema is not valid: %w", schemaErr))
		} else {
			t.ResolvedInputSchema = resolved
		}
	}

	if t.InputSchema != nil && strings.ToLower(t.InputSchema.Type) != "object" {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputScheme must be type object at the root"))
	}

	if t.InvocationData == nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not set for the tool"))
	} else if invocationErr := invocationValidator(t.InvocationType, t.InvocationData, t); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not valid: %w", invocationErr))
	}

	return err
}

func (s *MCPServer) Validate(invocationValidator InvocationValidator) error {
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

	for i, t := range s.Tools {
		if toolErr := t.Validate(invocationValidator); toolErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid server: tools[%d] is invalid: %w", i, toolErr))
		}
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

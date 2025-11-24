package mcpfile

import (
	"errors"
	"fmt"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

type InvocationValidator func(primitive invocation.Primitive) error

func (m *MCPToolDefinitionsFile) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if m.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool definitions file: name is required"))
	}

	if m.Version == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool definitions file: version is required"))
	}

	for i, t := range m.Tools {
		if toolErr := t.Validate(invocationValidator); toolErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid tool definitions file: tools[%d] is invalid: %w", i, toolErr))
		}
	}

	return err
}

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

	if t.InvocationConfigWrapper == nil || t.InvocationConfigWrapper.Config == nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not set for the tool"))
	} else if invocationErr := invocationValidator(t); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not valid: %w", invocationErr))
	}

	return err
}

func (p *Prompt) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if p.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid prompt: name is required"))
	}
	if p.Description == "" {
		err = errors.Join(err, fmt.Errorf("invalid prompt: description is required"))
	}

	if p.InputSchema == nil {
		err = errors.Join(err, fmt.Errorf("invalid prompt: inputSchema is required"))
	} else {
		resolved, schemaErr := p.InputSchema.Resolve(nil)
		if schemaErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid prompt: inputSchema is not valid: %w", schemaErr))
		} else {
			p.ResolvedInputSchema = resolved
		}
	}

	if p.InputSchema != nil && strings.ToLower(p.InputSchema.Type) != "object" {
		err = errors.Join(err, fmt.Errorf("invalid prompt: inputScheme must be type object at the root"))
	}
	if p.InvocationConfigWrapper == nil || p.InvocationConfigWrapper.Config == nil {
		err = errors.Join(err, fmt.Errorf("invalid prompt: invocation is not set for the prompt"))
	} else if invocationErr := invocationValidator(p); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid prompt: invocation is not valid: %w", invocationErr))
	}
	return err
}

func (r *Resource) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if r.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource: name is required"))
	}
	if r.Description == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource: description is required"))
	}
	if r.URI == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource: uri is required"))
	}
	if r.InputSchema != nil && strings.ToLower(r.InputSchema.Type) != "object" {
		err = errors.Join(err, fmt.Errorf("invalid resource: inputScheme must be type object at the root"))
	}
	if r.InvocationConfigWrapper == nil || r.InvocationConfigWrapper.Config == nil {
		err = errors.Join(err, fmt.Errorf("invalid resource: invocation is not set for the resource"))
	} else if invocationErr := invocationValidator(r); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid resource: invocation is not valid: %w", invocationErr))
	}
	return err
}

func (rt *ResourceTemplate) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if rt.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource template: name is required"))
	}
	if rt.Description == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource template: description is required"))
	}
	if rt.URITemplate == "" {
		err = errors.Join(err, fmt.Errorf("invalid resource template: uriTemplate is required"))
	}
	if rt.InputSchema == nil {
		err = errors.Join(err, fmt.Errorf("invalid resource template: inputSchema is required"))
	} else {
		resolved, schemaErr := rt.InputSchema.Resolve(nil)
		if schemaErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid resource template: inputSchema is not valid: %w", schemaErr))
		} else {
			rt.ResolvedInputSchema = resolved
		}
	}
	if rt.InputSchema != nil && strings.ToLower(rt.InputSchema.Type) != "object" {
		err = errors.Join(err, fmt.Errorf("invalid resource template: inputScheme must be type object at the root"))
	}
	if rt.InvocationConfigWrapper == nil || rt.InvocationConfigWrapper.Config == nil {
		err = errors.Join(err, fmt.Errorf("invalid resource template: invocation is not set for the resource template"))
	} else if invocationErr := invocationValidator(rt); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid resource template: invocation is not valid: %w", invocationErr))
	}
	return err
}

func (s *MCPToolDefinitions) Validate(invocationValidator InvocationValidator) error {
	var err error = nil
	if s.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid server: name is required"))
	}

	if s.Version == "" {
		err = errors.Join(err, fmt.Errorf("invalid server: version is required"))
	}

	for i, t := range s.Tools {
		if toolErr := t.Validate(invocationValidator); toolErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid server: tools[%d] is invalid: %w", i, toolErr))
		}
	}

	for i, p := range s.Prompts {
		if promptErr := p.Validate(invocationValidator); promptErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid server: prompts[%d] is invalid: %w", i, promptErr))
		}
	}

	return err
}

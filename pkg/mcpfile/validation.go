package mcpfile

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var validHttpMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodHead:   {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

func (js *JsonSchema) Validate() error {
	if js == nil {
		return nil
	}

	switch js.Type {
	case JsonSchemaTypeArray:
		if js.Properties != nil {
			return fmt.Errorf("cannot set properties on a array")
		}

		if js.AdditionalProperties != nil {
			return fmt.Errorf("cannot set additionalProperties on a array")
		}

		if len(js.Required) > 0 {
			return fmt.Errorf("cannot set required on a array")
		}

		if js.Items != nil {
			err := js.Items.Validate()
			if err != nil {
				return fmt.Errorf("invalid array items definition: %v", err)
			}
		}
	case JsonSchemaTypeObject:
		if js.Items != nil {
			return fmt.Errorf("cannot set items on an object")
		}

		if js.Properties != nil {
			var err error = nil
			for k, v := range js.Properties {
				propertyError := v.Validate()
				if propertyError != nil {
					err = errors.Join(err, fmt.Errorf("error with property %s: %v", k, propertyError))
				}
			}
			if err != nil {
				return fmt.Errorf("object has invalid properties: %v", err)
			}
		}

		if len(js.Required) > 0 {
			missingProperties := []string{}
			for _, req := range js.Required {
				_, ok := js.Properties[req]
				if !ok {
					missingProperties = append(missingProperties, req)
				}
			}

			if len(missingProperties) > 0 {
				return fmt.Errorf("object has no definition for the following required properties: %s", strings.Join(missingProperties, ", "))
			}
		}
	case JsonSchemaTypeBoolean, JsonSchemaTypeInteger, JsonSchemaTypeNumber, JsonSchemaTypeNull, JsonSchemaTypeString:
		if js.Items != nil {
			return fmt.Errorf("cannot set items on a %s", js.Type)
		}

		if js.Properties != nil {
			return fmt.Errorf("cannot set properties on a %s", js.Type)
		}

		if js.AdditionalProperties != nil {
			return fmt.Errorf("cannot set additionalProperties on a %s", js.Type)
		}

		if len(js.Required) > 0 {
			return fmt.Errorf("cannot set required on a %s", js.Type)
		}
	}

	return nil
}

func (t *Tool) Validate() error {
	var err error = nil
	if t.Name == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool: name is required"))
	}

	if t.Description == "" {
		err = errors.Join(err, fmt.Errorf("invalid tool: description is required"))
	}

	if t.InputSchema == nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputSchema is required"))
	} else if schemaErr := t.InputSchema.Validate(); schemaErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputSchema is not valid: %w", schemaErr))
	}

	if t.InputSchema != nil && t.InputSchema.Type != JsonSchemaTypeObject {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputScheme must be type object at the root"))
	}

	if schemaErr := t.OutputSchema.Validate(); schemaErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: outputSchema is not valid: %w", schemaErr))
	}

	if t.Invocation == nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not set for the tool"))
	} else if invocationErr := t.Invocation.Validate(t); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not valid: %w", invocationErr))
	}

	return err
}

func (s *MCPServer) Validate() error {
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
		if toolErr := t.Validate(); toolErr != nil {
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
			r.StreamableHTTPConfig.BasePath = "/mcp"
		}
	}

	return err
}

func (h *HttpInvocation) Validate(t *Tool) error {
	var err error = nil
	ok := IsValidHttpMethod(h.Method)
	if !ok {
		err = fmt.Errorf("invalid http request method for http invocation")
	}

	urlParts := strings.Split(h.URL, "{}")
	formattedUrlParts := []string{}
	// this isn't strictly validation, but it allows us to create typed format string
	for _, paramName := range h.pathParameters {
		if t.InputSchema.Properties == nil {
			err = errors.Join(err, fmt.Errorf("http invocation has %s path parameter, but there are no properties defined on the input schema", paramName))
			continue
		}

		param, ok := t.InputSchema.Properties[paramName]
		if !ok {
			err = errors.Join(err, fmt.Errorf("http invocation has %s path parameter, but there is no corresponding property defined on the input schema", paramName))
			continue
		}

		switch param.Type {
		case JsonSchemaTypeArray, JsonSchemaTypeNull, JsonSchemaTypeObject:
			err = errors.Join(err, fmt.Errorf(
				"http invocation path parameter %s has type %s in the input schema, which is not one of (string, number, integer, boolean)",
				paramName,
				param.Type,
			))
		case JsonSchemaTypeBoolean:
			formattedUrlParts = append(formattedUrlParts, urlParts[0], "%t")
			urlParts = urlParts[1:]
		case JsonSchemaTypeInteger:
			formattedUrlParts = append(formattedUrlParts, urlParts[0], "%d")
			urlParts = urlParts[1:]
		case JsonSchemaTypeNumber:
			formattedUrlParts = append(formattedUrlParts, urlParts[0], "%f")
			urlParts = urlParts[1:]
		case JsonSchemaTypeString:
			formattedUrlParts = append(formattedUrlParts, urlParts[0], "%s")
			urlParts = urlParts[1:]
		}
	}

	if err == nil {
		formattedUrlParts = append(formattedUrlParts, urlParts...)
		h.URL = strings.Join(formattedUrlParts, "")
	}

	return err
}

func (c *CliInvocation) Validate(t *Tool) error {
	var err error = nil

	for k, v := range c.TemplateVariables {
		if templateVariableErr := v.Validate(t); templateVariableErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid cli invocation: templateVariables.%s is invalid: %v", k, templateVariableErr))
		}
	}
	for _, tv := range c.TemplateVariables {
		err = errors.Join(err, tv.Validate(t))
	}

	commandParts := strings.Split(c.Command, "{}")
	formattedCommandParts := []string{}
	for _, paramName := range c.commandParameters {
		if t.InputSchema.Properties == nil {
			err = errors.Join(err, fmt.Errorf("cli invocation has %s command parameter, but there are no properties defined on the input schema", paramName))
			continue
		}

		tv, ok := c.TemplateVariables[paramName]
		if ok && tv.Property != "" {
			paramName = tv.Property
		}

		param, paramOk := t.InputSchema.Properties[paramName]
		if !paramOk {
			err = errors.Join(err, fmt.Errorf("cli invocation has %s command parameter, but there is no corresponding property defined on the input schema", paramName))
		}

		switch param.Type {
		case JsonSchemaTypeArray, JsonSchemaTypeNull, JsonSchemaTypeObject:
			err = errors.Join(err, fmt.Errorf(
				"cli invocation command parameter %s has type %s in the input schema, which is not one of (string, number, integer, boolean)",
				paramName,
				param.Type,
			))
		case JsonSchemaTypeBoolean:
			if ok {
				tv.setFormatVariable("%t")
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%s")
				commandParts = commandParts[1:]
			} else {
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%t")
				commandParts = commandParts[1:]
			}
		case JsonSchemaTypeInteger:
			if ok {
				tv.setFormatVariable("%d")
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%s")
				commandParts = commandParts[1:]
			} else {
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%d")
				commandParts = commandParts[1:]
			}
		case JsonSchemaTypeNumber:
			if ok {
				tv.setFormatVariable("%f")
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%s")
				commandParts = commandParts[1:]
			} else {
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%f")
				commandParts = commandParts[1:]
			}
		case JsonSchemaTypeString:
			if ok {
				tv.setFormatVariable("%s")
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%s")
				commandParts = commandParts[1:]
			} else {
				formattedCommandParts = append(formattedCommandParts, commandParts[0], "%s")
				commandParts = commandParts[1:]
			}
		}
	}

	if err == nil {
		formattedCommandParts = append(formattedCommandParts, commandParts...)
		c.Command = strings.Join(formattedCommandParts, "")
	}

	return err
}

func (tv *TemplateVariable) Validate(t *Tool) error {
	var err error = nil

	if len(tv.formatParameters) > 1 {
		err = errors.Join(err, fmt.Errorf("template variable has more than one parameters defined on it, when it should only have one"))
	}

	if len(tv.formatParameters) == 0 {
		return err
	}

	if _, ok := t.InputSchema.Properties[tv.formatParameters[0]]; !ok && tv.formatParameters[0] != tv.Property {
		err = errors.Join(err, fmt.Errorf("template format has a variable name that does not match the property name and is not set on the input schema"))
	}

	return err
}

func (tc *TemplateVariable) setFormatVariable(formatVariable string) {
	offset := strings.Index(tc.Format, "{}")
	if offset == -1 {
		return
	}

	tc.Format = tc.Format[:offset] + formatVariable + tc.Format[offset+2:]
}

func IsValidHttpMethod(method string) bool {
	_, ok := validHttpMethods[strings.ToUpper(method)]
	return ok
}

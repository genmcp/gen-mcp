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
		err = errors.Join(err, fmt.Errorf("invalid tool: inputSchema is not valid: %v", schemaErr))
	}

	if t.InputSchema.Type != JsonSchemaTypeObject {
		err = errors.Join(err, fmt.Errorf("invalid tool: inputScheme must be type object at the root"))
	}

	if schemaErr := t.OutputSchema.Validate(); schemaErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: outputSchema is not valid: %v", schemaErr))
	}

	if invocationErr := t.Invocation.Validate(); invocationErr != nil {
		err = errors.Join(err, fmt.Errorf("invalid tool: invocation is not valid: %v", invocationErr))
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

	for i, t := range s.Tools {
		if toolErr := t.Validate(); toolErr != nil {
			err = errors.Join(err, fmt.Errorf("invalid server: tools[%d] is invalid: %v", i, toolErr))
		}
	}

	return err
}

func (h *HttpInvocation) Validate() error {
	_, ok := validHttpMethods[h.Method]
	if !ok {
		return fmt.Errorf("invalid http request method for http invocation")
	}

	return nil
}

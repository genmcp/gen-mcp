package http

import (
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

var validHttpMethods = map[string]struct{}{
	nethttp.MethodGet:    {},
	nethttp.MethodHead:   {},
	nethttp.MethodPost:   {},
	nethttp.MethodPut:    {},
	nethttp.MethodPatch:  {},
	nethttp.MethodDelete: {},
}

// The configuration for making an HTTP request.
// This is a pure data structure with no parsing logic - all struct tags only.
type HttpInvocationConfig struct {
	// The URL for the HTTP request.
	//
	// It can contain placeholders in the form of {paramName} which correspond to parameters from the input schema.
	// It can contain placeholders in the form of {headers.paramName} which correspond to headers from the incoming
	// http request (won't work in stdio).
	// It can contain placeholders in the form of ${ENV_VAR_NAME} or {env.ENV_VAR_NAME} which correspond to env vars
	URL string `json:"url,omitempty" jsonschema:"required"` // even though this is required for the type, we don't require it on every nested extends instance of this struct, so we have omitempty

	// The headers for the HTTP request.
	//
	// Values can contain placeholders in the form of {paramName} which correspond to parameters from the input schema.
	// Values can contain placeholders in the form of {headers.paramName} which correspond to headers from the incoming
	// http request (won't work in stdio).
	// Values can contain placeholders in the form of ${ENV_VAR_NAME} or {env.ENV_VAR_NAME} which correspond to env vars
	Headers map[string]string `json:"headers,omitempty" jsonschema:"optional"`

	// The HTTP method to be used for the request (e.g., "GET", "POST").
	Method string `json:"method,omitempty" jsonschema:"required,enum=GET,enum=POST,enum=PUT,enum=PATCH,enum=DELETE,enum=HEAD"`
}

var _ invocation.InvocationConfig = &HttpInvocationConfig{}

func (hic *HttpInvocationConfig) Validate() error {
	if hic.URL == "" {
		return fmt.Errorf("URL is required")
	}

	// URL template validation is handled during template parsing
	if !IsValidHttpMethod(hic.Method) {
		return fmt.Errorf("invalid http request method: '%s'", hic.Method)
	}

	return nil
}

func (hic *HttpInvocationConfig) DeepCopy() invocation.InvocationConfig {
	headers := make(map[string]string, len(hic.Headers))
	for k, v := range hic.Headers {
		headers[k] = v
	}

	return &HttpInvocationConfig{
		URL:     hic.URL,
		Headers: headers,
		Method:  hic.Method,
	}
}

func IsValidHttpMethod(method string) bool {
	_, ok := validHttpMethods[strings.ToUpper(method)]
	return ok
}

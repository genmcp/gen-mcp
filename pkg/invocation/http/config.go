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
	// The URL for the HTTP request. It can contain placeholders in the form of {paramName} which correspond to parameters from the input schema.
	URL string `json:"url" jsonschema:"required"`

	// The HTTP method to be used for the request (e.g., "GET", "POST").
	Method string `json:"method" jsonschema:"required,enum=GET,enum=POST,enum=PUT,enum=PATCH,enum=DELETE,enum=HEAD"`
}

var _ invocation.InvocationConfig = &HttpInvocationConfig{}

func (hic *HttpInvocationConfig) Validate() error {
	// URL template validation is handled during template parsing
	if !IsValidHttpMethod(hic.Method) {
		return fmt.Errorf("invalid http request method: '%s'", hic.Method)
	}

	return nil
}

func IsValidHttpMethod(method string) bool {
	_, ok := validHttpMethods[strings.ToUpper(method)]
	return ok
}

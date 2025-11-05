package http

import (
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/template"
)

var validHttpMethods = map[string]struct{}{
	nethttp.MethodGet:    {},
	nethttp.MethodHead:   {},
	nethttp.MethodPost:   {},
	nethttp.MethodPut:    {},
	nethttp.MethodPatch:  {},
	nethttp.MethodDelete: {},
}

// The structure for HTTP invocation configuration.
// It is used to parse the raw JSON data that specifies how to make an HTTP request.
type HttpInvocationData struct {
	// Detailed HTTP invocation configuration.
	Http HttpInvocationConfig `json:"http" jsonschema:"required"`
}

// The configuration for making an HTTP request.
type HttpInvocationConfig struct {
	// The URL for the HTTP request. It can contain placeholders in the form of {paramName} which correspond to parameters from the input schema.
	URL string `json:"url" jsonschema:"required"`

	// The HTTP method to be used for the request (e.g., "GET", "POST").
	Method string `json:"method" jsonschema:"required,enum=GET,enum=POST,enum=PUT,enum=PATCH,enum=DELETE,enum=HEAD"`

	// ParsedTemplate is the parsed template for the URL path.
	// This field is for internal use and is not part of the JSON schema.
	ParsedTemplate *template.ParsedTemplate `json:"-"`

	// MCP URI template (for resource templates only).
	URITemplate string `json:"-"`
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

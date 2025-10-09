package http

import (
	"errors"
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

// The structure for HTTP invocation configuration.
// It is used to parse the raw JSON data that specifies how to make an HTTP request.
type HttpInvocationData struct {
	// Detailed HTTP invocation configuration.
	Http HttpInvocationConfig `json:"http" jsonschema:"required"`
}

// The configuration for making an HTTP request.
type HttpInvocationConfig struct {
	// The URL template for the HTTP request. It can contain placeholders in the form of '%' which correspond to parameters from the input schema.
	PathTemplate string `json:"url" jsonschema:"required"`

	// PathIndices maps parameter names to their positional index in the PathTemplate.
	// This field is for internal use and is not part of the JSON schema.
	PathIndices map[string]int `json:"-"`

	// The HTTP method to be used for the request (e.g., "GET", "POST").
	Method string `json:"method" jsonschema:"required,enum=GET,enum=POST,enum=PUT,enum=PATCH,enum=DELETE,enum=HEAD"`

	// MCP URI template (for resource templates only).
	URITemplate string `json:"-"`
}

var _ invocation.InvocationConfig = &HttpInvocationConfig{}

func (hic *HttpInvocationConfig) Validate() error {
	var err error = nil

	validPathIndicesCount := strings.Count(hic.PathTemplate, "%") == len(hic.PathIndices)
	if !validPathIndicesCount {
		err = fmt.Errorf("path indices do not match the number of template variables in the path template. expected %d, received %d", len(hic.PathIndices), strings.Count(hic.PathTemplate, "%"))
	}

	if !IsValidHttpMethod(hic.Method) {
		err = errors.Join(err, fmt.Errorf("invalid http request method: '%s'", hic.Method))
	}

	return err
}

func IsValidHttpMethod(method string) bool {
	_, ok := validHttpMethods[strings.ToUpper(method)]
	return ok
}

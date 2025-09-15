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

type HttpInvocationConfig struct {
	PathTemplate string         `json:"url"`
	PathIndices  map[string]int `json:"-"`
	Method       string         `json:"method"`
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

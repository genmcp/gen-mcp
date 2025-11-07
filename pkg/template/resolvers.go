package template

import (
	"fmt"
	nethttp "net/http"
)

// MapResolver resolves field values from a string map.
type MapResolver struct {
	data map[string]string
}

// NewMapResolver creates a resolver that looks up values in the provided map.
func NewMapResolver(data map[string]string) *MapResolver {
	return &MapResolver{data: data}
}

func (m *MapResolver) Resolve(fieldName string) (string, error) {
	val, ok := m.data[fieldName]
	if !ok {
		return "", fmt.Errorf("field '%s' not found", fieldName)
	}
	return val, nil
}

// HttpHeaderResolver resolves field values from HTTP headers.
type HttpHeaderResolver struct {
	headers nethttp.Header
}

// NewHttpHeaderResolver creates a resolver that looks up values in HTTP headers.
func NewHttpHeaderResolver(headers nethttp.Header) *HttpHeaderResolver {
	return &HttpHeaderResolver{headers: headers}
}

func (h *HttpHeaderResolver) Resolve(fieldName string) (string, error) {
	val := h.headers.Get(fieldName)
	if val == "" {
		return "", fmt.Errorf("header '%s' not found", fieldName)
	}
	return val, nil
}

package mcpfile

import (
	"encoding/json"
	"fmt"
)

func (t *Tool) MarshalJSON() ([]byte, error) {
	type Doppleganger Tool

	type DummyInvocation struct {
		Http *HttpInvocation `json:"http,omitempty"`
		Cli  *CliInvocation  `json:"cli,omitempty"`
	}
	tmp := &struct {
		Invocation DummyInvocation `json:"invocation"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(t),
	}

	if http, ok := t.Invocation.(*HttpInvocation); ok {
		tmp.Invocation.Http = http
	} else if cli, ok := t.Invocation.(*CliInvocation); ok {
		tmp.Invocation.Cli = cli
	}

	return json.Marshal(tmp)
}

func (h *HttpInvocation) MarshalJSON() ([]byte, error) {
	type Doppleganger HttpInvocation

	tmp := &struct {
		URL string `json:"url"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(h),
	}

	formattedParams := []any{}
	for _, p := range h.pathParameters {
		formattedParams = append(formattedParams, fmt.Sprintf("{%s}", p))
	}

	urlBytes := []byte(h.URL)
	for i := range len(urlBytes) {
		if urlBytes[i] == '%' && i+1 < len(urlBytes) {
			urlBytes[i+1] = 's'
		}
	}

	tmp.URL = fmt.Sprintf(string(urlBytes), formattedParams...)

	return json.Marshal(tmp)
}

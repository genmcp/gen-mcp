package mcpfile

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
)

func ParseMCPFile(path string) (*MCPFile, error) {
	mcpFile := &MCPFile{}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to mcpfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcpfile: %v", err)
	}

	err = yaml.Unmarshal(data, mcpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpfile: %v", err)
	}

	return mcpFile, nil
}

func (t *Tool) UnmarshalJSON(data []byte) error {
	type Doppleganger Tool

	tmp := struct {
		Invocation map[string]any `json:"invocation"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(t),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if len(tmp.Invocation) != 1 {
		return fmt.Errorf("only one invocation handler should be defined per tool")
	}

	for k, v := range tmp.Invocation {
		d, err := json.Marshal(v)
		if err != nil {
			// this should never happen
			return err
		}

		switch k {
		case InvocationTypeHttp:
			httpInvocation := &HttpInvocation{}
			err = json.Unmarshal(d, httpInvocation)
			if err != nil {
				return err
			}

			t.Invocation = httpInvocation
		default:
			return fmt.Errorf("unrecognized invocation format")
			
		}

	}

	return nil
}

func (h *HttpInvocation) UnmarshalJSON(data []byte) error {
	type Doppleganger HttpInvocation

	tmp := struct {
		URL string `json:"url"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(h),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	u, err := url.Parse(tmp.URL)
	if err != nil {
		return err
	}

	h.URL = *u
	h.Method = strings.ToUpper(h.Method)

	return nil
}

package mcpfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

	for s := range slices.Values(mcpFile.Servers) {
		err = errors.Join(err, s.Validate())
	}
	if err != nil {
		return nil, err
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

		switch strings.ToLower(k) {
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

	// for now this is not required, but including this so that new tags are picked up on
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	h.Method = strings.ToUpper(h.Method)

	// iterate over the (possibly) templated URL string and:
	// 1. collect any path parameters - in order
	// 2. replace each path parameter with {}, to be replaced later

	chunks := []string{}
	paramNames := []string{}
	var chunk strings.Builder
	for i := 0; i < len(tmp.URL); {
		if tmp.URL[i] == '{' {
			chunks = append(chunks, chunk.String(), "{}")
			chunk.Reset()

			offset := strings.Index(tmp.URL[i:], "}") + i
			if offset == -1 {
				return fmt.Errorf("unterminated path parameter found in URL")
			}

			paramName := tmp.URL[i+1 : offset]

			paramNames = append(paramNames, paramName)

			i = offset + 1
			continue
		} else if tmp.URL[i] == '}' {
			return fmt.Errorf("no opening bracket for a closing bracket in URL")
		}

		chunk.WriteByte(tmp.URL[i])
		i++
	}

	chunks = append(chunks, chunk.String())
	h.URL = strings.Join(chunks, "")

	if len(paramNames) > 0 {
		h.pathParameters = paramNames
	}

	return nil
}

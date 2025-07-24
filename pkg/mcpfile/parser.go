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

func (s *MCPServer) UnmarshalJSON(data []byte) error {
	type Doppleganger MCPServer

	tmp := struct {
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(s),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if s.Runtime == nil {
		s.Runtime = &ServerRuntime{
			TransportProtocol: TransportProtocolStreamableHttp,
			StreamableHTTPConfig: &StreamableHTTPConfig{
				Port: 3000,
			},
		}
	}

	if s.Runtime.TransportProtocol == TransportProtocolStreamableHttp && s.Runtime.StreamableHTTPConfig == nil {
		s.Runtime.StreamableHTTPConfig = &StreamableHTTPConfig{
			Port: 3000,
		}
	}

	return nil

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
		case InvocationTypeCli:
			cliInvocation := &CliInvocation{}
			err = json.Unmarshal(d, cliInvocation)
			if err != nil {
				return err
			}

			t.Invocation = cliInvocation
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

func (c *CliInvocation) UnmarshalJSON(data []byte) error {
	type Doppleganger CliInvocation

	tmp := struct {
		Command string `json:"command"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(c),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	// iterate over the (possibly) templated command string and:
	// 1. collect any parameters - in order
	// 2. replace each path parameter with %s

	chunks := []string{}
	paramNames := []string{}
	var chunk strings.Builder
	for i := 0; i < len(tmp.Command); {
		if tmp.Command[i] == '{' {
			chunks = append(chunks, chunk.String(), "{}")
			chunk.Reset()

			offset := strings.Index(tmp.Command[i:], "}") + i
			if offset-i == -1 {
				return fmt.Errorf("unterminated path parameter found in URL")
			}

			paramName := tmp.Command[i+1 : offset]

			paramNames = append(paramNames, paramName)

			i = offset + 1
			continue
		} else if tmp.Command[i] == '}' {
			return fmt.Errorf("no opening bracket for a closing bracket in URL")
		}

		chunk.WriteByte(tmp.Command[i])
		i++
	}

	chunks = append(chunks, chunk.String())
	c.Command = strings.Join(chunks, "")

	if len(paramNames) > 0 {
		c.commandParameters = paramNames
	}

	return nil
}

func (tv *TemplateVariable) UnmarshalJSON(data []byte) error {
	type Doppleganger TemplateVariable

	tmp := struct {
		Format string `json:"format"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(tv),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	chunks := []string{}
	paramNames := []string{}
	var chunk strings.Builder
	for i := 0; i < len(tmp.Format); {
		if tmp.Format[i] == '{' {
			chunks = append(chunks, chunk.String(), "{}")
			chunk.Reset()

			offset := strings.Index(tmp.Format[i:], "}") + i
			if offset-i == -1 {
				return fmt.Errorf("unterminated parameter found in template variable format")
			}

			paramName := tmp.Format[i+1 : offset]

			paramNames = append(paramNames, paramName)

			i = offset + 1
			continue
		} else if tmp.Format[i] == '}' {
			return fmt.Errorf("no opening bracket for a closing bracket in template variable format")
		}

		chunk.WriteByte(tmp.Format[i])
		i++
	}

	chunks = append(chunks, chunk.String())
	tv.Format = strings.Join(chunks, "")

	if len(paramNames) > 0 {
		tv.formatParameters = paramNames
	}

	return nil
}

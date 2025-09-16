package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	DefaultBasePath = "/mcp"
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
				Port:     3000,
				BasePath: DefaultBasePath,
			},
		}
	}

	if s.Runtime.TransportProtocol == TransportProtocolStreamableHttp && s.Runtime.StreamableHTTPConfig == nil {
		s.Runtime.StreamableHTTPConfig = &StreamableHTTPConfig{
			Port:     3000,
			BasePath: DefaultBasePath,
		}
	}

	return nil

}

func (t *Tool) UnmarshalJSON(data []byte) error {
	type Doppleganger Tool

	tmp := struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
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
		t.InvocationType = strings.ToLower(k)
		t.InvocationData = v
	}

	return nil
}

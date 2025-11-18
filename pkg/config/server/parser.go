package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/config"
	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"sigs.k8s.io/yaml"
)

const (
	DefaultBasePath = "/mcp"
)

func ParseMCPFile(path string) (*MCPServerConfigFile, error) {
	mcpFile := &MCPServerConfigFile{}

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

func (m *MCPServerConfigFile) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct to get all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal Kind separately
	if k, ok := raw["kind"]; ok {
		if err := json.Unmarshal(k, &m.Kind); err != nil {
			return err
		}
	}

	// Validate kind
	if m.Kind == "" {
		return fmt.Errorf("kind field is required, expected %s", KindMCPServerConfig)
	}
	if m.Kind != KindMCPServerConfig {
		return fmt.Errorf("invalid kind %s, expected %s", m.Kind, KindMCPServerConfig)
	}

	// Unmarshal SchemaVersion separately
	if fv, ok := raw["schemaVersion"]; ok {
		if err := json.Unmarshal(fv, &m.SchemaVersion); err != nil {
			return err
		}
	}

	if m.SchemaVersion != config.SchemaVersion {
		return fmt.Errorf("invalid schema version %s, expected %s - please migrate your file and handle any breaking changes", m.SchemaVersion, config.SchemaVersion)
	}

	// Unmarshal the rest into MCPServerConfig
	if err := json.Unmarshal(data, &m.MCPServerConfig); err != nil {
		return err
	}

	return nil
}

func (s *MCPServerConfig) UnmarshalJSON(data []byte) error {
	type Doppleganger MCPServerConfig

	tmp := struct {
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(s),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if len(s.InvocationBases) > 0 {
		extends.SetBases(s.InvocationBases)
	}

	// Only set defaults if we have a server defined (name or version present)
	if s.Name != "" || s.Version != "" {
		if s.Runtime == nil {
			s.Runtime = &ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &StreamableHTTPConfig{
					Port:      3000,
					BasePath:  DefaultBasePath,
					Stateless: true,
				},
			}
		}

		if s.Runtime.TransportProtocol == TransportProtocolStreamableHttp && s.Runtime.StreamableHTTPConfig == nil {
			s.Runtime.StreamableHTTPConfig = &StreamableHTTPConfig{
				Port:      3000,
				BasePath:  DefaultBasePath,
				Stateless: true,
			}
		}
	}

	return nil

}

func (s *StreamableHTTPConfig) UnmarshalJSON(data []byte) error {
	type Doppleganger StreamableHTTPConfig

	tmp := struct {
		Stateless *bool `json:"stateless,omitempty"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(s),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if tmp.Stateless != nil {
		s.Stateless = *tmp.Stateless
	} else {
		s.Stateless = true
	}

	return nil
}

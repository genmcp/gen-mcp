package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"github.com/google/jsonschema-go/jsonschema"
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

func (m *MCPFile) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct to get all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal FileVersion separately
	if fv, ok := raw["mcpFileVersion"]; ok {
		if err := json.Unmarshal(fv, &m.FileVersion); err != nil {
			return err
		}
	}

	if m.FileVersion != MCPFileVersion {
		return fmt.Errorf("invalid mcp file version %s, expected %s - please migrate your file and handle any breaking changes", m.FileVersion, MCPFileVersion)
	}

	// Unmarshal the rest into MCPServerConfig
	if err := json.Unmarshal(data, &m.MCPServer); err != nil {
		return err
	}

	return nil
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

func (t *Tool) UnmarshalJSON(data []byte) error {
	type Doppleganger Tool

	tmp := (*Doppleganger)(t)

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	if t.InputSchema == nil {
		t.InputSchema = &jsonschema.Schema{
			Properties: make(map[string]*jsonschema.Schema),
		}
	}

	if t.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		t.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	if t.InputSchema.Type == "" {
		t.InputSchema.Type = "object"
	}

	return nil
}

func (p *Prompt) UnmarshalJSON(data []byte) error {
	type Doppleganger Prompt

	tmp := (*Doppleganger)(p)

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	return nil
}

func (p *Resource) UnmarshalJSON(data []byte) error {
	type Doppleganger Resource

	tmp := (*Doppleganger)(p)

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	return nil
}

func (p *ResourceTemplate) UnmarshalJSON(data []byte) error {
	type Doppleganger ResourceTemplate

	tmp := (*Doppleganger)(p)

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
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

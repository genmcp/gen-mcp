package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/config"
	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"github.com/google/jsonschema-go/jsonschema"
	"sigs.k8s.io/yaml"
)

// ParseMCPFile parses a Tool Definitions File (mcpfile.yaml)
func ParseMCPFile(path string) (*MCPToolDefinitionsFile, error) {
	mcpFile := &MCPToolDefinitionsFile{}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to tool definitions file: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool definitions file: %v", err)
	}

	err = yaml.Unmarshal(data, mcpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool definitions file: %v", err)
	}

	return mcpFile, nil
}

func (m *MCPToolDefinitionsFile) UnmarshalJSON(data []byte) error {
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
		return fmt.Errorf("kind field is required, expected %s", KindMCPToolDefinitions)
	}
	if m.Kind != KindMCPToolDefinitions {
		return fmt.Errorf("invalid kind %s, expected %s", m.Kind, KindMCPToolDefinitions)
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

	// Unmarshal the rest into MCPToolDefinitions
	if err := json.Unmarshal(data, &m.MCPToolDefinitions); err != nil {
		return err
	}

	return nil
}

func (s *MCPToolDefinitions) UnmarshalJSON(data []byte) error {
	type Doppleganger MCPToolDefinitions

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

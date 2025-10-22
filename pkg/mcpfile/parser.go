package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"sigs.k8s.io/yaml"
)

const (
	DefaultBasePath = "/mcp"
)

// ParseMCPServerConfig parses a server configuration file (mcpserver.yaml).
func ParseMCPServerConfig(path string) (*MCPServerConfig, error) {
	config := &MCPServerConfig{}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to mcpserver config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcpserver config: %v", err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpserver config: %v", err)
	}

	return config, nil
}

// ParseMCPToolDefinitions parses a tool definitions file (mcpfile.yaml).
func ParseMCPToolDefinitions(path string) (*MCPToolDefinitions, error) {
	defs := &MCPToolDefinitions{}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to mcpfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcpfile: %v", err)
	}

	err = yaml.Unmarshal(data, defs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpfile: %v", err)
	}

	return defs, nil
}

// ParseMCPFile parses an MCP file and returns the combined server configuration.
// It supports both the legacy format (single file) and the new format (separate files).
// For the new format, pass the path to the mcpserver.yaml file.
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

// CombineConfigs combines a server config and tool definitions into a unified MCPServer.
func CombineConfigs(serverConfig *MCPServerConfig, toolDefs *MCPToolDefinitions) *MCPServer {
	return &MCPServer{
		Name:              serverConfig.Name,
		Version:           serverConfig.Version,
		Runtime:           serverConfig.Runtime,
		Instructions:      toolDefs.Instructions,
		Tools:             toolDefs.Tools,
		Prompts:           toolDefs.Prompts,
		Resources:         toolDefs.Resources,
		ResourceTemplates: toolDefs.ResourceTemplates,
	}
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

	// Unmarshal the rest into MCPServer
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
		// ensure that this is object
		t.InputSchema.Type = "object"
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

func (p *Prompt) UnmarshalJSON(data []byte) error {
	type Doppleganger Prompt

	tmp := struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(p),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	if len(tmp.Invocation) != 1 {
		return fmt.Errorf("only one invocation handler should be defined per prompt")
	}

	for k, v := range tmp.Invocation {
		p.InvocationType = strings.ToLower(k)
		p.InvocationData = v
	}

	return nil
}

func (p *Resource) UnmarshalJSON(data []byte) error {
	type Doppleganger Resource

	tmp := struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(p),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	if len(tmp.Invocation) != 1 {
		return fmt.Errorf("only one invocation handler should be defined per resource")
	}

	for k, v := range tmp.Invocation {
		p.InvocationType = strings.ToLower(k)
		p.InvocationData = v
	}

	return nil
}

func (p *ResourceTemplate) UnmarshalJSON(data []byte) error {
	type Doppleganger ResourceTemplate

	tmp := struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(p),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	if p.InputSchema != nil && p.InputSchema.Properties == nil {
		// set the properties to be not nil so that it serializes as {} (required for some clients to properly parse the tool)
		p.InputSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	if len(tmp.Invocation) != 1 {
		return fmt.Errorf("only one invocation handler should be defined per resource template")
	}

	for k, v := range tmp.Invocation {
		p.InvocationType = strings.ToLower(k)
		p.InvocationData = v
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

func (c *MCPServerConfig) UnmarshalJSON(data []byte) error {
type Doppleganger MCPServerConfig

tmp := struct {
*Doppleganger
}{
Doppleganger: (*Doppleganger)(c),
}

err := json.Unmarshal(data, &tmp)
if err != nil {
return err
}

if c.FileVersion != MCPFileVersion {
return fmt.Errorf("invalid mcp file version %s, expected %s - please migrate your file and handle any breaking changes", c.FileVersion, MCPFileVersion)
}

if c.Kind != KindMCPServerConfig {
return fmt.Errorf("invalid kind %s, expected %s", c.Kind, KindMCPServerConfig)
}

// Set defaults for runtime if needed
if c.Runtime != nil {
if c.Runtime.TransportProtocol == TransportProtocolStreamableHttp && c.Runtime.StreamableHTTPConfig == nil {
c.Runtime.StreamableHTTPConfig = &StreamableHTTPConfig{
Port:      3000,
BasePath:  DefaultBasePath,
Stateless: true,
}
}
}

return nil
}

func (t *MCPToolDefinitions) UnmarshalJSON(data []byte) error {
type Doppleganger MCPToolDefinitions

tmp := struct {
*Doppleganger
}{
Doppleganger: (*Doppleganger)(t),
}

err := json.Unmarshal(data, &tmp)
if err != nil {
return err
}

if t.FileVersion != MCPFileVersion {
return fmt.Errorf("invalid mcp file version %s, expected %s - please migrate your file and handle any breaking changes", t.FileVersion, MCPFileVersion)
}

if t.Kind != KindMCPToolDefinitions {
return fmt.Errorf("invalid kind %s, expected %s", t.Kind, KindMCPToolDefinitions)
}

return nil
}

// SerializeMCPFile serializes an MCPFile to YAML format.
func SerializeMCPFile(mcpFile *MCPFile) ([]byte, error) {
return yaml.Marshal(mcpFile)
}

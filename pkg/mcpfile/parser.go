package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"sigs.k8s.io/yaml"
)

const (
	DefaultBasePath = "/mcp"
)

// TODO: remove
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

// TODO: remove
func (m *MCPFile) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct to get all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal SchemaVersion separately
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
	// Unmarshal into both embedded structs
	err := json.Unmarshal(data, &s.MCPToolDefinitions)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &s.MCPServerConfig)
	if err != nil {
		return err
	}

	// Merge invocation bases and set them for extends
	mergedBases := s.InvocationBases()
	if len(mergedBases) > 0 {
		extends.SetBases(mergedBases)
	}

	// Only set defaults if we have a server defined (name or version present)
	name := s.Name()
	version := s.Version()
	if name != "" || version != "" {
		if s.MCPServerConfig.Runtime == nil {
			s.MCPServerConfig.Runtime = &ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &StreamableHTTPConfig{
					Port:      3000,
					BasePath:  DefaultBasePath,
					Stateless: true,
				},
			}
		}

		if s.MCPServerConfig.Runtime.TransportProtocol == TransportProtocolStreamableHttp && s.MCPServerConfig.Runtime.StreamableHTTPConfig == nil {
			s.MCPServerConfig.Runtime.StreamableHTTPConfig = &StreamableHTTPConfig{
				Port:      3000,
				BasePath:  DefaultBasePath,
				Stateless: true,
			}
		}
	}

	return nil
}

// Note: UnmarshalJSON methods for Tool, Prompt, Resource, ResourceTemplate are defined in pkg/config/definitions
// and are available through type aliases. UnmarshalJSON for StreamableHTTPConfig is defined in pkg/config/server.

// Note: UnmarshalJSON method for StreamableHTTPConfig is defined in pkg/config/server and is available through type alias.

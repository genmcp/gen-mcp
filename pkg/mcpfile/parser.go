package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	"sigs.k8s.io/yaml"
)

const (
	DefaultBasePath = "/mcp"
)

// ParseMCPFile parses an MCP file and returns an MCPServer.
// It validates the mcpFileVersion field if present.
func ParseMCPFile(path string) (*MCPServer, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to mcpfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcpfile: %v", err)
	}

	// Check version before unmarshaling
	var raw map[string]json.RawMessage
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpfile: %v", err)
	}

	if fv, ok := raw["mcpFileVersion"]; ok {
		var fileVersion string
		if err := json.Unmarshal(fv, &fileVersion); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mcpFileVersion: %v", err)
		}
		if fileVersion != MCPFileVersion {
			return nil, fmt.Errorf("invalid mcp file version %s, expected %s - please migrate your file and handle any breaking changes", fileVersion, MCPFileVersion)
		}
	}

	mcpServer := &MCPServer{}
	err = yaml.Unmarshal(data, mcpServer)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpfile: %v", err)
	}

	return mcpServer, nil
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
			s.MCPServerConfig.Runtime = &serverconfig.ServerRuntime{
				TransportProtocol: TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
					Port:      3000,
					BasePath:  DefaultBasePath,
					Stateless: true,
				},
			}
		}

		if s.MCPServerConfig.Runtime.TransportProtocol == TransportProtocolStreamableHttp && s.MCPServerConfig.Runtime.StreamableHTTPConfig == nil {
			s.MCPServerConfig.Runtime.StreamableHTTPConfig = &serverconfig.StreamableHTTPConfig{
				Port:      3000,
				BasePath:  DefaultBasePath,
				Stateless: true,
			}
		}
	}

	return nil
}

// Note: UnmarshalJSON methods for Tool, Prompt, Resource, ResourceTemplate are defined in pkg/config/definitions.
// UnmarshalJSON for StreamableHTTPConfig is defined in pkg/config/server.

package mcpfile

import (
	"fmt"
	"os"
	"sync"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"go.uber.org/zap"
)

// TODO: not used?
const (
	SchemaVersion                   = "0.2.0"
	TransportProtocolStreamableHttp = "streamablehttp"
	TransportProtocolStdio          = "stdio"
	KindMCPServerConfig             = "MCPServerConfig"
)

// StreamableHTTPConfig defines configuration for the HTTP-based runtime.
type StreamableHTTPConfig struct {
	// Port number to listen on.
	Port int `json:"port" jsonschema:"required"`

	// Base path for the MCP server (default: /mcp).
	BasePath string `json:"basePath,omitempty" jsonschema:"optional"`

	// Indicates whether the server is stateless (default: true when unset).
	Stateless bool `json:"stateless,omitempty" jsonschema:"optional"`

	// OAuth 2.0 configuration for protected resources.
	Auth *AuthConfig `json:"auth,omitempty" jsonschema:"optional"`

	// TLS configuration for HTTPS.
	TLS *TLSConfig `json:"tls,omitempty" jsonschema:"optional"`
}

// TLSConfig defines paths to TLS certificate and private key files.
type TLSConfig struct {
	// Absolute path to the server's public certificate.
	CertFile string `json:"certFile,omitempty" jsonschema:"optional"`

	// Absolute path to the server's private key.
	KeyFile string `json:"keyFile,omitempty" jsonschema:"optional"`
}

// AuthConfig defines OAuth 2.0 authorization settings.
type AuthConfig struct {
	// List of authorization server URLs for token validation.
	AuthorizationServers []string `json:"authorizationServers,omitempty" jsonschema:"optional"`

	// URI for the JSON Web Key Set (JWKS) used for token verification.
	JWKSURI string `json:"jwksUri,omitempty" jsonschema:"optional"`
}

// StdioConfig defines configuration for stdio transport protocol.
type StdioConfig struct{}

// ServerRuntime defines transport protocol and associated configuration.
type ServerRuntime struct {
	// Transport protocol to use (streamablehttp or stdio).
	TransportProtocol string `json:"transportProtocol" jsonschema:"required"`

	// Configuration for streamable HTTP transport protocol.
	StreamableHTTPConfig *StreamableHTTPConfig `json:"streamableHttpConfig,omitempty" jsonschema:"optional"`

	// Configuration for stdio transport protocol.
	StdioConfig *StdioConfig `json:"stdioConfig,omitempty" jsonschema:"optional"`

	// Configuration for the server logging
	LoggingConfig *logging.LoggingConfig `json:"loggingConfig" jsonschema:"optional"`

	baseLogger     *zap.Logger
	initLoggerOnce sync.Once
}

// GetBaseLogger returns the base logger for the server, defaulting to noop if either the runtime or
// any of the logging config is nil or has errors
func (sr *ServerRuntime) GetBaseLogger() *zap.Logger {
	if sr == nil {
		return zap.NewNop()
	}

	sr.initLoggerOnce.Do(func() {
		if sr.LoggingConfig != nil {
			logger, err := sr.LoggingConfig.BuildBase()
			if err != nil || logger == nil {
				// Surface the error to stderr before falling back
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: Failed to build base logger, using no-op logger: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "ERROR: BuildBase returned nil logger, using no-op logger\n")
				}
				logger = zap.NewNop()
			}
			sr.baseLogger = logger
		} else {
			sr.baseLogger = zap.NewNop()
		}
	})

	return sr.baseLogger
}

// MCPServerConfig defines the metadata and capabilities of an MCP server.
type MCPServerConfig struct {
	// Name of the MCP server.
	Name string `json:"name" jsonschema:"required"`

	// Semantic version of the MCP server.
	Version string `json:"version" jsonschema:"required"`

	// Runtime configuration for the MCP server.
	Runtime *ServerRuntime `json:"runtime,omitempty" jsonschema:"optional"`

	// A set of instructions provided by the server to the client about how to use the server
	Instructions string `json:"instructions,omitempty" jsonschema:"optional"`

	// InvocationBases contains base configs for invocations
	InvocationBases map[string]*invocation.InvocationConfigWrapper `json:"invocationBases,omitempty" jsonschema:"optional"`
}

// MCPServerConfigFile is the root structure of an MCP configuration file.
type MCPServerConfigFile struct {
	// Kind identifies the type of MCP file.
	Kind string `json:"kind" jsonschema:"required"`

	// Version of the MCP file format.
	SchemaVersion string `json:"schemaVersion" jsonschema:"required"`

	// MCP server definition.
	MCPServerConfig `json:",inline"`
}

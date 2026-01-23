package server

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"go.uber.org/zap"
)

const (
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

	// Health check configuration for k8s probes.
	Health *HealthConfig `json:"health,omitempty" jsonschema:"optional"`
}

// TLSConfig defines paths to TLS certificate and private key files.
type TLSConfig struct {
	// Absolute path to the server's public certificate.
	CertFile string `json:"certFile,omitempty" jsonschema:"optional"`

	// Absolute path to the server's private key.
	KeyFile string `json:"keyFile,omitempty" jsonschema:"optional"`
}

type HealthConfig struct {
	// Enable health endpoints (default: true when running HTTP)
	Enabled bool `json:"enabled,omitempty" jsonschema:"optional"`

	// Path for liveness probe (default: /healthz)
	LivenessPath string `json:"livenessPath,omitempty" jsonschema:"optional"`

	// Path for readiness probe (default: /readyz)
	ReadinessPath string `json:"readinessPath,omitempty" jsonschema:"optional"`
}

// ClientTLSConfig defines TLS settings for outbound HTTP requests.
// Use this to configure custom CA certificates for connecting to internal services
// that use certificates signed by a corporate or private CA.
type ClientTLSConfig struct {
	// Paths to CA certificate files (PEM format) to trust for outbound HTTPS requests.
	// These are added to the system's default certificate pool.
	CACertFiles []string `json:"caCertFiles,omitempty" jsonschema:"optional"`

	// Path to a directory containing CA certificate files (PEM format).
	// All .pem and .crt files in this directory will be loaded.
	CACertDir string `json:"caCertDir,omitempty" jsonschema:"optional"`

	// If true, skip TLS certificate verification for outbound requests.
	// WARNING: This is insecure and should only be used for testing.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" jsonschema:"optional"`
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

	// TLS configuration for outbound HTTP requests (e.g., custom CA certificates).
	// Use this when connecting to internal services that use certificates signed by a corporate CA.
	ClientTLSConfig *ClientTLSConfig `json:"clientTlsConfig,omitempty" jsonschema:"optional"`

	baseLogger     *zap.Logger
	initLoggerOnce sync.Once

	httpClient     *http.Client
	httpClientErr  error
	httpClientOnce sync.Once
}

// GetBaseLogger returns the base logger for the server.
// If LoggingConfig is nil, it defaults to a console logger with info level to ensure
// startup messages are visible as documented in tutorials.
// If LoggingConfig is provided but fails to build, it falls back to a console logger.
// If the runtime is nil, it returns a no-op logger.
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
					fmt.Fprintf(os.Stderr, "ERROR: Failed to build base logger, using default console logger: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "ERROR: BuildBase returned nil logger, using default console logger\n")
				}
				// Fall back to default console logger
				config := zap.NewDevelopmentConfig()
				config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
				config.Encoding = "console"
				logger, _ = config.Build()
				if logger == nil {
					logger = zap.NewNop()
				}
			}
			sr.baseLogger = logger
		} else {
			// Default to console logger with info level when no logging config is provided
			// This ensures users can see startup messages as documented in tutorials
			config := zap.NewDevelopmentConfig()
			config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
			config.Encoding = "console"
			logger, err := config.Build()
			if err != nil || logger == nil {
				// Last resort: use no-op logger
				sr.baseLogger = zap.NewNop()
			} else {
				sr.baseLogger = logger
			}
		}
	})

	return sr.baseLogger
}

// MCPServerConfig defines the runtime configuration of an MCP server.
type MCPServerConfig struct {
	// Runtime configuration for the MCP server.
	Runtime *ServerRuntime `json:"runtime,omitempty" jsonschema:"optional"`
}

// MCPServerConfigFile is the root structure of a Server Config File (mcpserver.yaml).
type MCPServerConfigFile struct {
	// Kind identifies the type of GenMCP config file.
	Kind string `json:"kind" jsonschema:"required"`

	// Version of the GenMCP config file format.
	SchemaVersion string `json:"schemaVersion" jsonschema:"required"`

	// MCP server definition.
	MCPServerConfig `json:",inline"`
}

package server

// Default values for server configuration.
const (
	// DefaultBasePath is the default base path for the MCP server.
	DefaultBasePath = "/mcp"

	// DefaultPort is the default port for the streamable HTTP server.
	DefaultPort = 8080

	// DefaultLivenessPath is the default path for the liveness probe endpoint.
	DefaultLivenessPath = "/healthz"

	// DefaultReadinessPath is the default path for the readiness probe endpoint.
	DefaultReadinessPath = "/readyz"
)

// ApplyDefaults applies default values to the MCPServerConfig after parsing.
func (s *MCPServerConfig) ApplyDefaults() {
	if s.Runtime == nil {
		s.Runtime = &ServerRuntime{}
	}
	s.Runtime.ApplyDefaults()
}

// ApplyDefaults applies default values to the MCPServerConfigFile after parsing.
func (m *MCPServerConfigFile) ApplyDefaults() {
	m.MCPServerConfig.ApplyDefaults()
}

// ApplyDefaults applies default values to ServerRuntime.
func (r *ServerRuntime) ApplyDefaults() {
	if r.TransportProtocol == "" {
		r.TransportProtocol = TransportProtocolStreamableHttp
	}

	if r.TransportProtocol == TransportProtocolStreamableHttp {
		if r.StreamableHTTPConfig == nil {
			r.StreamableHTTPConfig = &StreamableHTTPConfig{}
		}
		r.StreamableHTTPConfig.ApplyDefaults()
	}
}

// ApplyDefaults applies default values to StreamableHTTPConfig.
func (s *StreamableHTTPConfig) ApplyDefaults() {
	if s.Port <= 0 {
		s.Port = DefaultPort
	}
	if s.BasePath == "" {
		s.BasePath = DefaultBasePath
	}
	// Stateless defaults to true when nil
	if s.Stateless == nil {
		stateless := true
		s.Stateless = &stateless
	}

	if s.Health == nil {
		s.Health = &HealthConfig{}
	}
	s.Health.ApplyDefaults()
}

// ApplyDefaults applies default values to HealthConfig.
func (h *HealthConfig) ApplyDefaults() {
	// Enabled defaults to true when nil
	if h.Enabled == nil {
		enabled := true
		h.Enabled = &enabled
	}
	if h.LivenessPath == "" {
		h.LivenessPath = DefaultLivenessPath
	}
	if h.ReadinessPath == "" {
		h.ReadinessPath = DefaultReadinessPath
	}
}

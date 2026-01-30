package server

import (
	"fmt"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName  string
		expected      *MCPServerConfigFile
		wantErr       bool
		errorContains string
	}{
		"default": {
			testFileName: "server-default.yaml",
			expected: &MCPServerConfigFile{
				Kind:          KindMCPServerConfig,
				SchemaVersion: config.SchemaVersion,
				MCPServerConfig: MCPServerConfig{
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      DefaultPort,
							BasePath:  DefaultBasePath,
							Stateless: ptr.To(true),
							Health: &HealthConfig{
								Enabled:       ptr.To(true),
								ReadinessPath: "/readyz",
								LivenessPath:  "/healthz",
							},
						},
					},
				},
			},
		},
		"stateful": {
			testFileName: "server-stateful.yaml",
			expected: &MCPServerConfigFile{
				Kind:          KindMCPServerConfig,
				SchemaVersion: config.SchemaVersion,
				MCPServerConfig: MCPServerConfig{
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000, // explicitly set in YAML
							Stateless: ptr.To(false),
							Health: &HealthConfig{
								Enabled:       ptr.To(true),
								ReadinessPath: "/readyz",
								LivenessPath:  "/healthz",
							},
						},
					},
				},
			},
		},
		"server runtime stdio": {
			testFileName: "server-runtime-stdio.yaml",
			expected: &MCPServerConfigFile{
				Kind:          KindMCPServerConfig,
				SchemaVersion: config.SchemaVersion,
				MCPServerConfig: MCPServerConfig{
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStdio,
					},
				},
			},
		},
		"full demo": {
			testFileName: "full-demo.yaml",
			expected: &MCPServerConfigFile{
				Kind:          KindMCPServerConfig,
				SchemaVersion: config.SchemaVersion,
				MCPServerConfig: MCPServerConfig{
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      8008,
							BasePath:  DefaultBasePath,
							Stateless: ptr.To(true),
							Health: &HealthConfig{
								Enabled:       ptr.To(true),
								ReadinessPath: "/readyz",
								LivenessPath:  "/healthz",
							},
						},
					},
				},
			},
		},
		"with tls": {
			testFileName: "server-tls.yaml",
			expected: &MCPServerConfigFile{
				Kind:          KindMCPServerConfig,
				SchemaVersion: config.SchemaVersion,
				MCPServerConfig: MCPServerConfig{
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      7007,
							BasePath:  DefaultBasePath,
							Stateless: ptr.To(true),
							TLS: &TLSConfig{
								CertFile: "/path/to/server.crt",
								KeyFile:  "/path/to/server.key",
							},
							Health: &HealthConfig{
								Enabled:       ptr.To(true),
								ReadinessPath: "/readyz",
								LivenessPath:  "/healthz",
							},
						},
					},
				},
			},
		},
		"invalid version 0.0.1": {
			testFileName:  "invalid-schema-version.yaml",
			wantErr:       true,
			errorContains: "invalid schema version",
		},
	}

	for testName, testCase := range tt {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			mcpFile, err := ParseMCPFile(fmt.Sprintf("./testdata/%s", testCase.testFileName))
			if testCase.wantErr {
				assert.Error(t, err, "parsing mcp file should cause an error")
				assert.ErrorContains(t, err, testCase.errorContains, "the error should contain the right message")
			} else {
				assert.NoError(t, err, "parsing mcp file should succeed")
				// Apply defaults after parsing (this is the new pattern: parse -> apply defaults -> validate)
				mcpFile.ApplyDefaults()
			}

			assert.Equal(t, testCase.expected, mcpFile)
		})

	}
}

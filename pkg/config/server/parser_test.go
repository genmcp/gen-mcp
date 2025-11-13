package mcpfile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName  string
		expected      *MCPServerFile
		wantErr       bool
		errorContains string
	}{
		"no servers": {
			testFileName: "no-servers.yaml",
			expected: &MCPServerFile{
				FileVersion: MCPFileVersion,
			},
		},
		"stateful": {
			testFileName: "one-server-stateful.yaml",
			expected: &MCPServerFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: false,
						},
					},
				},
			},
		},
		"server runtime stdio": {
			testFileName: "server-runtime-stdio.yaml",
			expected: &MCPServerFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStdio,
					},
				},
			},
		},
		"full demo": {
			testFileName: "full-demo.yaml",
			expected: &MCPServerFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "git-github-example",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      8008,
							Stateless: true,
						},
					},
				},
			},
		},
		"with tls": {
			testFileName: "one-server-tls.yaml",
			expected: &MCPServerFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      7007,
							Stateless: true,
							TLS: &TLSConfig{
								CertFile: "/path/to/server.crt",
								KeyFile:  "/path/to/server.key",
							},
						},
					},
				},
			},
		},
		"invalid version 0.0.1": {
			testFileName:  "invalid-file-version.yaml",
			wantErr:       true,
			errorContains: "invalid mcp file version",
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
			}

			assert.Equal(t, testCase.expected, mcpFile)
		})

	}
}

package mcpfile

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName string
		expected     *MCPFile
		wantErr      bool
	}{
		"no servers": {
			testFileName: "no-servers.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
			},
		},
		"one server, no tools": {
			testFileName: "one-server-no-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port:     3000,
								BasePath: DefaultBasePath,
							},
						},
					},
				},
			},
		},
		"one server, with tools": {
			testFileName: "one-server-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port:     3000,
								BasePath: DefaultBasePath,
							},
						},
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"companyName": {
											Type:        "string",
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								InvocationData: json.RawMessage(`{"method":"POST","url":"http://localhost:5000"}`),
								InvocationType: "http",
							},
						},
					},
				},
			},
		},
		"one server, with tools and http params": {
			testFileName: "one-server-tools-http-params.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port:     3000,
								BasePath: DefaultBasePath,
							},
						},
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"companyName": {
											Type:        "string",
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								InvocationData: json.RawMessage(`{"method":"GET","url":"http://localhost:5000/{companyName}/users"}`),
								InvocationType: "http",
							},
						},
					},
				},
			},
		},
		"one server, cli invocation": {
			testFileName: "one-server-cli-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								BasePath: DefaultBasePath,
								Port:     3000,
							},
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"repoUrl": {
											Type:        "string",
											Description: "The git url of the repo to clone",
										},
										"depth": {
											Type:        "integer",
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        "boolean",
											Description: "Whether to return verbose logs",
										},
									},
									Required: []string{"repoUrl"},
								},
								InvocationData: json.RawMessage(`{"command":"git clone {repoUrl} {depth} {verbose}","templateVariables":{"depth":{"format":"--depth {depth}"},"verbose":{"format":"--verbose","omitIfFalse":true}}}`),
								InvocationType: "cli",
							},
						},
					},
				},
			},
		},
		"server runtime stdio": {
			testFileName: "server-runtime-stdio.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStdio,
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"repoUrl": {
											Type:        "string",
											Description: "The git url of the repo to clone",
										},
										"depth": {
											Type:        "integer",
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        "boolean",
											Description: "Whether to return verbose logs",
										},
									},
									Required: []string{"repoUrl"},
								},
								InvocationData: json.RawMessage(`{"command":"git clone {repoUrl} {depth} {verbose}","templateVariables":{"depth":{"format":"--depth {depth}"},"verbose":{"format":"--verbose","omitIfFalse":true}}}`),
								InvocationType: "cli",
							},
						},
					},
				},
			},
		},
		"full demo": {
			testFileName: "full-demo.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "git-github-example",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port: 8008,
							},
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"repoUrl": {
											Type:        "string",
											Description: "The git url of the repo to clone. If cloning with ssh, this should be the ssh url, if cloning with https this should be the https url.",
										},
										"depth": {
											Type:        "integer",
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        "boolean",
											Description: "Whether to return verbose logs",
										},
										"path": {
											Type:        "string",
											Description: "The relative or absolute path to clone the repo to, if not cloning to {current directory}/{repo name}",
										},
									},
									Required: []string{"repoUrl"},
								},
								InvocationData: json.RawMessage(`{"command":"git clone {depth} {verbose} {repoUrl} {path}","templateVariables":{"depth":{"format":"--depth {depth}"},"verbose":{"format":"--verbose","omitIfFalse":true}}}`),
								InvocationType: "cli",
							},
							{
								Name:        "ensure_dir_exists",
								Title:       "Ensure directory exists",
								Description: "Ensure that a given directory exists on the machine",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"path": {
											Type:        "string",
											Description: "The path to the directory",
										},
									},
									Required: []string{"path"},
								},
								InvocationData: json.RawMessage(`{"command":"mkdir -p {path}"}`),
								InvocationType: "cli",
							},
							{
								Name:        "get_repo_url",
								Title:       "Get repository url",
								Description: "Get the https or ssh url for a git repository given the organization name and repo name",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"org": {
											Type:        "string",
											Description: "The name of the github organization",
										},
										"repoName": {
											Type:        "string",
											Description: "The name of the github repository",
										},
										"scheme": {
											Type:        "string",
											Description: "The scheme of the returned url. Must be one of https or ssh",
										},
									},
									Required: []string{"org", "repoName"},
								},
								InvocationData: json.RawMessage(`{"method":"GET","url":"http://localhost:9090/repos/{org}/{repoName}"}`),
								InvocationType: "http",
							},
						},
					},
				},
			},
		},
		"one server, with tls": {
			testFileName: "one-server-tls.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port: 7007,
								TLS: &TLSConfig{
									CertFile: "/path/to/server.crt",
									KeyFile:  "/path/to/server.key",
								},
							},
						},
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"companyName": {
											Type:        "string",
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								InvocationData: json.RawMessage(`{"method":"POST","url":"http://localhost:5000"}`),
								InvocationType: "http",
							},
						},
					},
				},
			},
		},
	}

	for testName, testCase := range tt {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			mcpFile, err := ParseMCPFile(fmt.Sprintf("./testdata/%s", testCase.testFileName))
			if testCase.wantErr {
				assert.Error(t, err, "parsing mcp file should cause an error")
			} else {
				assert.NoError(t, err, "parsing mcp file should succeed")
			}

			assert.Equal(t, testCase.expected, mcpFile)
		})

	}
}

package mcpfile

import (
	"fmt"
	"net/http"
	"testing"

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
								Port: 3000,
								BasePath: "/mcp",
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
								Port: 3000,
								BasePath: "/mcp",
							},
						},
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"companyName": {
											Type:        JsonSchemaTypeString,
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								Invocation: &HttpInvocation{
									URL:    "http://localhost:5000",
									Method: http.MethodPost,
								},
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
								Port: 3000,
								BasePath: "/mcp",
							},
						},
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"companyName": {
											Type:        JsonSchemaTypeString,
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								Invocation: &HttpInvocation{
									URL:            "http://localhost:5000/%s/users",
									Method:         http.MethodGet,
									pathParameters: []string{"companyName"},
								},
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
								Port: 3000,
								BasePath: "/mcp",
							},
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"repoUrl": {
											Type:        JsonSchemaTypeString,
											Description: "The git url of the repo to clone",
										},
										"depth": {
											Type:        JsonSchemaTypeInteger,
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        JsonSchemaTypeBoolean,
											Description: "Whether to return verbose logs",
										},
									},
									Required: []string{"repoUrl"},
								},
								Invocation: &CliInvocation{
									Command: "git clone %s %s %s",
									TemplateVariables: map[string]*TemplateVariable{
										"depth": {
											Format:           "--depth %d",
											formatParameters: []string{"depth"},
										},
										"verbose": {
											Format:      "--verbose",
											OmitIfFalse: true,
										},
									},
									commandParameters: []string{"repoUrl", "depth", "verbose"},
								},
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
							TransportProtocol: "stdio",
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"repoUrl": {
											Type:        JsonSchemaTypeString,
											Description: "The git url of the repo to clone",
										},
										"depth": {
											Type:        JsonSchemaTypeInteger,
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        JsonSchemaTypeBoolean,
											Description: "Whether to return verbose logs",
										},
									},
									Required: []string{"repoUrl"},
								},
								Invocation: &CliInvocation{
									Command: "git clone %s %s %s",
									TemplateVariables: map[string]*TemplateVariable{
										"depth": {
											Format:           "--depth %d",
											formatParameters: []string{"depth"},
										},
										"verbose": {
											Format:      "--verbose",
											OmitIfFalse: true,
										},
									},
									commandParameters: []string{"repoUrl", "depth", "verbose"},
								},
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
							TransportProtocol: "streamablehttp",
							StreamableHTTPConfig: &StreamableHTTPConfig{
								Port: 8008,
								BasePath: "/mcp",
							},
						},
						Tools: []*Tool{
							{
								Name:        "clone_repo",
								Title:       "Clone git repository",
								Description: "Clone a git repository from a url to the local machine",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"repoUrl": {
											Type:        JsonSchemaTypeString,
											Description: "The git url of the repo to clone. If cloning with ssh, this should be the ssh url, if cloning with https this should be the https url.",
										},
										"depth": {
											Type:        JsonSchemaTypeInteger,
											Description: "The number of commits to clone",
										},
										"verbose": {
											Type:        JsonSchemaTypeBoolean,
											Description: "Whether to return verbose logs",
										},
										"path": {
											Type:        JsonSchemaTypeString,
											Description: "The relative or absolute path to clone the repo to, if not cloning to {current directory}/{repo name}",
										},
									},
									Required: []string{"repoUrl"},
								},
								Invocation: &CliInvocation{
									Command: "git clone %s %s %s %s",
									TemplateVariables: map[string]*TemplateVariable{
										"depth": {
											Format:           "--depth %d",
											formatParameters: []string{"depth"},
										},
										"verbose": {
											Format:      "--verbose",
											OmitIfFalse: true,
										},
									},
									commandParameters: []string{"depth", "verbose", "repoUrl", "path"},
								},
							},
							{
								Name:        "ensure_dir_exists",
								Title:       "Ensure directory exists",
								Description: "Ensure that a given directory exists on the machine",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"path": {
											Type:        JsonSchemaTypeString,
											Description: "The path to the directory",
										},
									},
									Required: []string{"path"},
								},
								Invocation: &CliInvocation{
									Command:           "mkdir -p %s",
									commandParameters: []string{"path"},
								},
							},
							{
								Name:        "get_repo_url",
								Title:       "Get repository url",
								Description: "Get the https or ssh url for a git repository given the organization name and repo name",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"org": {
											Type:        JsonSchemaTypeString,
											Description: "The name of the github organization",
										},
										"repoName": {
											Type:        JsonSchemaTypeString,
											Description: "The name of the github repository",
										},
										"scheme": {
											Type:        JsonSchemaTypeString,
											Description: "The scheme of the returned url. Must be one of https or ssh",
										},
									},
									Required: []string{"org", "repoName"},
								},
								Invocation: &HttpInvocation{
									URL:            "http://localhost:9090/repos/%s/%s",
									Method:         http.MethodGet,
									pathParameters: []string{"org", "repoName"},
								},
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

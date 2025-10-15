package mcpfile

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName  string
		expected      *MCPFile
		wantErr       bool
		errorContains string
	}{
		"no servers": {
			testFileName: "no-servers.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
			},
		},
		"no tools": {
			testFileName: "one-server-no-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
						},
					},
				},
			},
		},
		"with tools": {
			testFileName: "one-server-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
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
							Annotations: &ToolAnnotations{
								IdempotentHint:  ptr.To(false),
								ReadOnlyHint:    ptr.To(true),
								OpenWorldHint:   ptr.To(false),
								DestructiveHint: ptr.To(false),
							},
						},
					},
				},
			},
		},
		"with tools and http params": {
			testFileName: "one-server-tools-http-params.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
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
		"cli invocation": {
			testFileName: "one-server-cli-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
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
		"stateful": {
			testFileName: "one-server-stateful.yaml",
			expected: &MCPFile{
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
		"server runtime stdio": {
			testFileName: "server-runtime-stdio.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
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
		"one server, prompts": {
			testFileName: "one-server-prompts.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
						},
					},
					Prompts: []*Prompt{
						{
							Name:        "code_review",
							Title:       "Request Code Review",
							Description: "Asks the LLM to analyze code quality and suggest improvements",
							Arguments: []*PromptArgument{
								{
									Name:        "code",
									Title:       "Code",
									Description: "The code to review",
									Required:    true,
								},
							},
							InputSchema: &jsonschema.Schema{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"code": {
										Type:        "string",
										Description: "The code to review",
									},
								},
								Required: []string{"code"},
							},
							InvocationData: json.RawMessage(`{"method":"POST","url":"http://localhost:5000"}`),
							InvocationType: "http",
						},
					},
				},
			},
		}, "one server, resources": {
			testFileName: "one-server-resources.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
						},
					},
					Resources: []*Resource{
						{
							Name:           "web_server_access_log",
							Title:          "Web Server Access Log",
							Description:    "Contains a record of all requests made to the web server",
							MIMEType:       "text/plain",
							Size:           1024,
							URI:            "http://localhost:5000/access.log",
							InvocationData: json.RawMessage(`{"method":"GET","url":"http://localhost:5000"}`),
							InvocationType: "http",
						},
					},
				},
			},
		},
		"one server, resource templates": {
			testFileName: "one-server-resource-templates.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				MCPServer: MCPServer{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
						},
					},
					ResourceTemplates: []*ResourceTemplate{
						{
							Name:        "weather-forecast",
							Title:       "Weather Forecast",
							Description: "Get weather forecast for any city and date",
							MIMEType:    "application/json",
							URITemplate: "weather://forecast/{city}/{date}",
							InputSchema: &jsonschema.Schema{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"city": {
										Type:        "string",
										Description: "The city to get weather for",
									},
									"date": {
										Type:        "string",
										Description: "The date to get weather for",
									},
								},
								Required: []string{"city", "date"},
							},
							InvocationData: json.RawMessage(`{"method":"GET","url":"http://localhost:5000/forecast"}`),
							InvocationType: "http",
						},
					},
				},
			},
		},
		"full demo": {
			testFileName: "full-demo.yaml",
			expected: &MCPFile{
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
		"with tls": {
			testFileName: "one-server-tls.yaml",
			expected: &MCPFile{
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

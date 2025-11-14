package mcpfile

import (
	"fmt"
	"testing"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	cliInv "github.com/genmcp/gen-mcp/pkg/invocation/cli"
	httpInv "github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName  string
		expected      *MCPServer
		wantErr       bool
		errorContains string
	}{
		"no servers": {
			testFileName: "no-servers.yaml",
			expected:     &MCPServer{},
		},
		"no tools": {
			testFileName: "one-server-no-tools.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
						},
					},
				},
			},
		},
		"with instructions": {
			testFileName: "one-server-instructions.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Instructions: "These are the server instructions.\nIt can be a multi line string\n",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
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
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "http",
								Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:5000",
									Method: "POST",
								},
							},
							Annotations: &definitions.ToolAnnotations{
								IdempotentHint:  ptr.To(false),
								ReadOnlyHint:    ptr.To(true),
								OpenWorldHint:   ptr.To(false),
								DestructiveHint: ptr.To(false),
							},
						},
					},
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
						},
					},
				},
			},
		},
		"with tools and http params": {
			testFileName: "one-server-tools-http-params.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "http",
								Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:5000/{companyName}/users",
									Method: "GET",
								},
							},
						},
					},
				},
			},
		},
		"cli invocation": {
			testFileName: "one-server-cli-tools.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "cli",
								Config: &cliInv.CliInvocationConfig{
									Command: "git clone {repoUrl} {depth} {verbose}",
									TemplateVariables: map[string]*cliInv.TemplateVariable{
										"depth": {
											Template:    "--depth {depth}",
											OmitIfFalse: false,
										},
										"verbose": {
											Template:    "--verbose",
											OmitIfFalse: true,
										},
									},
								},
							},
						},
					},
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:      3000,
							BasePath:  DefaultBasePath,
							Stateless: true,
						},
					},
				},
			},
		},
		"stateful": {
			testFileName: "one-server-stateful.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "cli",
								Config: &cliInv.CliInvocationConfig{
									Command: "git clone {repoUrl} {depth} {verbose}",
									TemplateVariables: map[string]*cliInv.TemplateVariable{
										"depth": {
											Template:    "--depth {depth}",
											OmitIfFalse: false,
										},
										"verbose": {
											Template:    "--verbose",
											OmitIfFalse: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"server runtime stdio": {
			testFileName: "server-runtime-stdio.yaml",
			expected: &MCPServer{
					MCPToolDefinitions: definitions.MCPToolDefinitions{
						Name:    "test-server",
						Version: "1.0.0",
						Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "cli",
								Config: &cliInv.CliInvocationConfig{
									Command: "git clone {repoUrl} {depth} {verbose}",
									TemplateVariables: map[string]*cliInv.TemplateVariable{
										"depth": {
											Template:    "--depth {depth}",
											OmitIfFalse: false,
										},
										"verbose": {
											Template:    "--verbose",
											OmitIfFalse: true,
										},
									},
								},
							},
						},
					},
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStdio,
					},
				},
			},
		},
		"one server, prompts": {
			testFileName: "one-server-prompts.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Prompts: []*definitions.Prompt{
							{
								Name:        "code_review",
								Title:       "Request Code Review",
								Description: "Asks the LLM to analyze code quality and suggest improvements",
							Arguments: []*definitions.PromptArgument{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "http",
								Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:5000",
									Method: "POST",
								},
							},
						},
					},
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
						},
					},
				},
			},
		}, "one server, resources": {
			testFileName: "one-server-resources.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					Resources: []*definitions.Resource{
							{
								Name:           "web_server_access_log",
								Title:          "Web Server Access Log",
								Description:    "Contains a record of all requests made to the web server",
								MIMEType:       "text/plain",
								Size:           1024,
								URI:            "http://localhost:5000/access.log",
								InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
									Type: "http",
									Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:5000",
									Method: "GET",
								},
							},
						},
					},
				},
			},
		},
		"one server, resource templates": {
			testFileName: "one-server-resource-templates.yaml",
			expected: &MCPServer{
					MCPToolDefinitions: definitions.MCPToolDefinitions{
						Name:    "test-server",
						Version: "1.0.0",
					},
					MCPServerConfig: serverconfig.MCPServerConfig{
						Name:    "test-server",
						Version: "1.0.0",
						Runtime: &serverconfig.ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							BasePath:  DefaultBasePath,
							Port:      3000,
							Stateless: true,
						},
					},
				},
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "test-server",
					Version: "1.0.0",
					ResourceTemplates: []*definitions.ResourceTemplate{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "http",
								Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:5000/forecast",
									Method: "GET",
								},
							},
						},
					},
				},
			},
		},
		"full demo": {
			testFileName: "full-demo.yaml",
			expected: &MCPServer{
				MCPToolDefinitions: definitions.MCPToolDefinitions{
					Name:    "git-github-example",
					Version: "1.0.0",
					Tools: []*definitions.Tool{
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "cli",
								Config: &cliInv.CliInvocationConfig{
									Command: "git clone {depth} {verbose} {repoUrl} {path}",
									TemplateVariables: map[string]*cliInv.TemplateVariable{
										"depth": {
											Template:    "--depth {depth}",
											OmitIfFalse: false,
										},
										"verbose": {
											Template:    "--verbose",
											OmitIfFalse: true,
										},
									},
								},
							},
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "cli",
								Config: &cliInv.CliInvocationConfig{
									Command: "mkdir -p {path}",
								},
							},
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
							InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
								Type: "http",
								Config: &httpInv.HttpInvocationConfig{
									URL:    "http://localhost:9090/repos/{org}/{repoName}",
									Method: "GET",
								},
							},
						},
					},
					MCPServerConfig: serverconfig.MCPServerConfig{
						Name:    "git-github-example",
						Version: "1.0.0",
						Runtime: &serverconfig.ServerRuntime{
							TransportProtocol: TransportProtocolStreamableHttp,
							StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
								Port:      8008,
								Stateless: true,
							},
						},
					},
				},
			},
		},
		"with tls": {
			testFileName: "one-server-tls.yaml",
			expected: &MCPServer{
					MCPToolDefinitions: definitions.MCPToolDefinitions{
						Name:    "test-server",
						Version: "1.0.0",
						Tools: []*definitions.Tool{
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
								InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
									Type: "http",
									Config: &httpInv.HttpInvocationConfig{
										URL:    "http://localhost:5000",
										Method: "POST",
									},
								},
							},
						},
					},
				},
				MCPServerConfig: serverconfig.MCPServerConfig{
					Name:    "test-server",
					Version: "1.0.0",
					Runtime: &serverconfig.ServerRuntime{
						TransportProtocol: TransportProtocolStreamableHttp,
						StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
							Port:      7007,
							Stateless: true,
							TLS: &serverconfig.TLSConfig{
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
			mcpServer, err := ParseMCPFile(fmt.Sprintf("./testdata/%s", testCase.testFileName))
			if testCase.wantErr {
				assert.Error(t, err, "parsing mcp file should cause an error")
				assert.ErrorContains(t, err, testCase.errorContains, "the error should contain the right message")
			} else {
				assert.NoError(t, err, "parsing mcp file should succeed")
			}

			assert.Equal(t, testCase.expected, mcpServer)
		})

	}
}




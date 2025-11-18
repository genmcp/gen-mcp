package test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/genmcp/gen-mcp/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var _ = Describe("Basic Integration", Ordered, func() {
	Describe("MCP Server Basic HTTP Functionality", Ordered, func() {
		const (
			mcpServerPort = 8020
			mcpServerURL  = "http://localhost:8020/mcp"
		)

		var (
			backendServer       *httptest.Server
			mcpConfig           *mcpserver.MCPServer
			mcpServerCancelFunc context.CancelFunc
		)

		BeforeEach(func() {
			backendServer = createMockBackendServerIntegration()
			mcpConfig = createBasicTestMCPConfig(backendServer.URL, mcpServerPort)

			By("starting MCP server")
			ctx := context.Background()
			ctx, mcpServerCancelFunc = context.WithCancel(ctx)

			go func() {
				defer GinkgoRecover()
				err := runtime.DoRunServer(ctx, mcpConfig)
				if err != nil && !strings.Contains(err.Error(), "Server closed") {
					Fail(fmt.Sprintf("Failed to start MCP server: %v", err))
				}
			}()

			// Give server time to start
			time.Sleep(500 * time.Millisecond)
		})

		AfterEach(func() {
			By("cleaning up test servers")
			if backendServer != nil {
				backendServer.Close()
			}
			if mcpServerCancelFunc != nil {
				mcpServerCancelFunc()
			}
		})

		Describe("HTTP Server Tests", func() {
			It("should start and respond to HTTP requests", func() {
				By("making HTTP request to MCP server")
				resp, err := http.Get(mcpServerURL)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = resp.Body.Close()
				}()

				By("receiving response from server")
				// The server should respond (not reject the connection)
				// It may return 400/405 for invalid MCP requests, but should be reachable
				Expect(resp.StatusCode).To(SatisfyAny(
					Equal(http.StatusOK),
					Equal(http.StatusBadRequest),
					Equal(http.StatusMethodNotAllowed),
				))
			})
		})

		Describe("MCP Protocol Tests", func() {
			var (
				client  *mcp.Client
				session *mcp.ClientSession
			)

			BeforeEach(func() {
				By("creating MCP client")
				client = mcp.NewClient(&mcp.Implementation{
					Name:    "test client",
					Version: "0.0.1",
				}, nil)

				By("connecting to MCP server")
				transport := &mcp.StreamableClientTransport{
					Endpoint: mcpServerURL,
				}
				Eventually(func() error {
					var err error
					session, err = client.Connect(context.Background(), transport, nil)
					return err
				}, 2*time.Second, 100*time.Millisecond).Should(Succeed())
			})

			AfterEach(func() {
				if session != nil {
					_ = session.Close()
				}
			})

			It("should successfully complete MCP handshake", func() {
				By("verifying successful initialization")
				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())
				Expect(initResult.ServerInfo).NotTo(BeNil())
			})

			It("should return server instructions", func() {
				By("verifying instructions are present")
				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())
				Expect(initResult.Instructions).To(Equal("This is a test HTTP server with tools, prompts, and resources for integration testing."))
			})

			It("should list available tools", func() {
				By("listing tools")
				toolsResult, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(toolsResult.Tools).To(HaveLen(1))
				Expect(toolsResult.Tools[0].Name).To(Equal("get_users"))
			})

			It("should successfully call tools", func() {
				By("calling tool")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "get_users",
					Arguments: map[string]any{
						"companyName": "test_company",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Content).To(HaveLen(1))

				textResult, ok := result.Content[0].(*mcp.TextContent)
				Expect(ok).To(BeTrue())
				Expect(textResult.Text).To(MatchJSON(`{"users": [{"name": "John Doe", "email": "john@test_company.com"}]}`))
			})

			It("should list available prompts", func() {
				By("listing prompts")
				promptsResult, err := session.ListPrompts(context.Background(), &mcp.ListPromptsParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(promptsResult.Prompts).To(HaveLen(1))
				Expect(promptsResult.Prompts[0].Name).To(Equal("greeting_prompt"))
			})

			It("should successfully call prompts", func() {
				By("calling prompt")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name: "greeting_prompt",
					Arguments: map[string]string{
						"userName": "Alice",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Messages).To(HaveLen(1))
				Expect(result.Messages[0].Content.(*mcp.TextContent).Text).To(ContainSubstring("Hello, Alice"))
			})

			It("should list available resources", func() {
				By("listing resources")
				resourcesResult, err := session.ListResources(context.Background(), &mcp.ListResourcesParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resourcesResult.Resources).To(HaveLen(1))
				Expect(resourcesResult.Resources[0].Name).To(Equal("server_info"))
			})

			It("should successfully read resources", func() {
				By("reading resource")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
					URI: fmt.Sprintf("%s/server/info", backendServer.URL),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Contents).To(HaveLen(1))
				Expect(result.Contents[0].Text).To(MatchJSON(`{"server":"test-server","version":"1.0","status":"running"}`))
			})
		})
	})

	Describe("CLI Invocation Tests", Ordered, func() {
		const (
			mcpServerPort = 8021
		)

		var (
			mcpConfig           *mcpserver.MCPServer
			mcpServerCancelFunc context.CancelFunc
		)

		BeforeEach(func() {
			mcpConfig = createCLITestMCPConfig(mcpServerPort)

			By("starting MCP server with CLI tools")
			ctx := context.Background()
			ctx, mcpServerCancelFunc = context.WithCancel(ctx)

			go func() {
				defer GinkgoRecover()
				err := runtime.DoRunServer(ctx, mcpConfig)
				if err != nil && !strings.Contains(err.Error(), "Server closed") {
					Fail(fmt.Sprintf("Failed to start MCP server: %v", err))
				}
			}()

			// Give server time to start
			time.Sleep(500 * time.Millisecond)
		})

		AfterEach(func() {
			By("cleaning up MCP server")
			if mcpServerCancelFunc != nil {
				mcpServerCancelFunc()
			}
		})

		Describe("CLI Commands Execution", func() {
			var (
				client  *mcp.Client
				session *mcp.ClientSession
			)

			BeforeEach(func() {
				By("creating MCP client")
				client = mcp.NewClient(&mcp.Implementation{
					Name:    "test cli client",
					Version: "0.0.1",
				}, nil)

				By("connecting to MCP server")
				transport := &mcp.StreamableClientTransport{
					Endpoint: fmt.Sprintf("http://localhost:%d/mcp", mcpServerPort),
				}

				Eventually(func() error {
					var err error
					session, err = client.Connect(context.Background(), transport, nil)
					return err
				}, 2*time.Second, 100*time.Millisecond).Should(Succeed())
			})

			AfterEach(func() {
				if session != nil {
					_ = session.Close()
				}
			})

			It("should execute CLI commands successfully", func() {
				By("calling CLI tool")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "list_files",
					Arguments: map[string]any{
						"path": ".",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Content).To(HaveLen(1))

				textResult, ok := result.Content[0].(*mcp.TextContent)
				Expect(ok).To(BeTrue())
				Expect(textResult.Text).NotTo(BeEmpty())
			})

			It("should list available prompts", func() {
				By("listing prompts")
				promptsResult, err := session.ListPrompts(context.Background(), &mcp.ListPromptsParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(promptsResult.Prompts).To(HaveLen(1))
				Expect(promptsResult.Prompts[0].Name).To(Equal("greeting_prompt"))
			})

			It("should execute CLI prompt invocation successfully", func() {
				By("calling CLI prompt")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name: "greeting_prompt",
					Arguments: map[string]string{
						"userName": "Bob",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Messages).To(HaveLen(1))
				Expect(result.Messages[0].Content.(*mcp.TextContent).Text).To(ContainSubstring("Hello, Bob"))
			})

			It("should list available resources", func() {
				By("listing resources")
				resourcesResult, err := session.ListResources(context.Background(), &mcp.ListResourcesParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resourcesResult.Resources).To(HaveLen(1))
				Expect(resourcesResult.Resources[0].Name).To(Equal("server_info"))
			})

			It("should read CLI resource invocation successfully", func() {
				By("reading CLI resource")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
					URI: "file://tmp/server-info.json",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Contents).To(HaveLen(1))
				Expect(result.Contents[0].URI).To(Equal("file://tmp/server-info.json"))
				Expect(result.Contents[0].Text).To(Equal("test-server-cli-server version 1.0 running\n"))
			})
		})
	})
})

func createMockBackendServerIntegration() *httptest.Server {
	By("creating mock backend server")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/server/info" {
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintln(w, `{"server":"test-server","version":"1.0","status":"running"}`)
			Expect(err).NotTo(HaveOccurred())
			return
		}
		if r.URL.Path == "/prompts/greeting_prompt" && r.Method == "POST" {
			// Read and parse the request body to extract userName
			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err == nil {
				if userName, ok := requestBody["userName"].(string); ok {
					w.Header().Set("Content-Type", "application/json")
					response := fmt.Sprintf(`{"message":"Hello, %s! Welcome to our system."}`, userName)
					_, err := fmt.Fprintln(w, response)
					Expect(err).NotTo(HaveOccurred())
					return
				}
			}
			// Fallback response if userName not found
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintln(w, `{"message":"Hello! Welcome to our system."}`)
			Expect(err).NotTo(HaveOccurred())
			return
		}
		// Default response for user endpoints
		_, err := fmt.Fprintln(w, `{"users": [{"name": "John Doe", "email": "john@test_company.com"}]}`)
		Expect(err).NotTo(HaveOccurred())
	}))
}

func createBasicTestMCPConfig(backendURL string, port int) *mcpserver.MCPServer {
	By("creating basic test MCP configuration")

	toolDefsYAML := fmt.Sprintf(`
kind: MCPToolDefinitions
schemaVersion: 0.2.0
name: test-server
version: "1.0"
instructions: "This is a test HTTP server with tools, prompts, and resources for integration testing."
tools:
  - name: get_users
    title: Users Provider
    description: Get list of users from a given company
    inputSchema:
      type: object
      properties:
        companyName:
          type: string
          description: Name of the company
      required:
        - companyName
    invocation:
      http:
        url: "%s/{companyName}/users"
        method: GET
prompts:
  - name: greeting_prompt
    description: Generate a greeting message for a user
    inputSchema:
      type: object
      properties:
        userName:
          type: string
          description: Name of the user to greet
      required:
        - userName
    invocation:
      http:
        method: POST
        url: "%s/prompts/greeting_prompt"
resources:
  - name: server_info
    description: Information about the server
    uri: "%s/server/info"
    mimeType: "application/json"
    invocation:
      http:
        url: "%s/server/info"
        method: GET
`, backendURL, backendURL, backendURL, backendURL)

	serverConfigYAML := fmt.Sprintf(`
kind: MCPServerConfig
schemaVersion: 0.2.0
name: test-server
version: "1.0"
instructions: "This is a test HTTP server with tools, prompts, and resources for integration testing."
runtime:
  streamableHttpConfig:
    port: %d
    basePath: "/mcp"
  transportProtocol: streamablehttp
`, port)

	toolDefsFile, err := os.CreateTemp("", "mcp-tooldefs-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(toolDefsFile.Name())

	_, err = toolDefsFile.WriteString(toolDefsYAML)
	Expect(err).NotTo(HaveOccurred())
	toolDefsFile.Close()

	serverConfigFile, err := os.CreateTemp("", "mcp-serverconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(serverConfigFile.Name())

	_, err = serverConfigFile.WriteString(serverConfigYAML)
	Expect(err).NotTo(HaveOccurred())
	serverConfigFile.Close()

	toolDefs, err := definitions.ParseMCPFile(toolDefsFile.Name())
	Expect(err).NotTo(HaveOccurred())

	serverConfig, err := serverconfig.ParseMCPFile(serverConfigFile.Name())
	Expect(err).NotTo(HaveOccurred())

	return &mcpserver.MCPServer{
		MCPToolDefinitions: toolDefs.MCPToolDefinitions,
		MCPServerConfig:    serverConfig.MCPServerConfig,
	}
}

func createCLITestMCPConfig(port int) *mcpserver.MCPServer {
	By("creating CLI test MCP configuration")

	toolDefsYAML := `
kind: MCPToolDefinitions
schemaVersion: 0.2.0
name: test-cli-server
version: "1.0"
tools:
  - name: list_files
    title: List Files
    description: List files in a directory
    inputSchema:
      type: object
      properties:
        path:
          type: string
          description: Path to list
      required:
        - path
    invocation:
      cli:
        command: "ls -la {path}"
prompts:
  - name: greeting_prompt
    description: Generate a greeting message for a user
    inputSchema:
      type: object
      properties:
        userName:
          type: string
          description: Name of the user to greet
      required:
        - userName
    invocation:
      cli:
        command: "echo 'Hello, {userName}! Welcome to our system.'"
resources:
  - name: server_info
    description: Information about the server
    uri: "file://tmp/server-info.json"
    mimeType: "text/plain"
    invocation:
      cli:
        command: "echo 'test-server-cli-server version 1.0 running'"
`

	serverConfigYAML := fmt.Sprintf(`
kind: MCPServerConfig
schemaVersion: 0.2.0
name: test-cli-server
version: "1.0"
runtime:
  streamableHttpConfig:
    port: %d
    basePath: "/mcp"
  transportProtocol: streamablehttp
`, port)

	toolDefsFile, err := os.CreateTemp("", "mcp-cli-tooldefs-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(toolDefsFile.Name())

	_, err = toolDefsFile.WriteString(toolDefsYAML)
	Expect(err).NotTo(HaveOccurred())
	toolDefsFile.Close()

	serverConfigFile, err := os.CreateTemp("", "mcp-cli-serverconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(serverConfigFile.Name())

	_, err = serverConfigFile.WriteString(serverConfigYAML)
	Expect(err).NotTo(HaveOccurred())
	serverConfigFile.Close()

	toolDefs, err := definitions.ParseMCPFile(toolDefsFile.Name())
	Expect(err).NotTo(HaveOccurred())

	serverConfig, err := serverconfig.ParseMCPFile(serverConfigFile.Name())
	Expect(err).NotTo(HaveOccurred())

	return &mcpserver.MCPServer{
		MCPToolDefinitions: toolDefs.MCPToolDefinitions,
		MCPServerConfig:    serverConfig.MCPServerConfig,
	}
}

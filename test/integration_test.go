package test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
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
			mcpConfig           *mcpfile.MCPFile
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
				err := mcpserver.RunServer(ctx, mcpConfig.Server)
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
				var err error
				session, err = client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
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
		})
	})

	Describe("CLI Invocation Tests", Ordered, func() {
		const (
			mcpServerPort = 8021
		)

		var (
			mcpConfig           *mcpfile.MCPFile
			mcpServerCancelFunc context.CancelFunc
		)

		BeforeEach(func() {
			mcpConfig = createCLITestMCPConfig(mcpServerPort)

			By("starting MCP server with CLI tools")
			ctx := context.Background()
			ctx, mcpServerCancelFunc = context.WithCancel(ctx)

			go func() {
				defer GinkgoRecover()
				err := mcpserver.RunServer(ctx, mcpConfig.Server)
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

		Describe("CLI Tool Execution", func() {
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
				var err error
				session, err = client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
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
		})
	})
})

func createMockBackendServerIntegration() *httptest.Server {
	By("creating mock backend server")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, `{"users": [{"name": "John Doe", "email": "john@test_company.com"}]}`)
		Expect(err).NotTo(HaveOccurred())
	}))
}

func createBasicTestMCPConfig(backendURL string, port int) *mcpfile.MCPFile {
	By("creating basic test MCP configuration")

	mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
name: test-server
version: "1.0"
runtime:
  streamableHttpConfig:
    port: %d
    basePath: "/mcp"
  transportProtocol: streamablehttp
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
`, port, backendURL)

	tmpfile, err := os.CreateTemp("", "mcp-basic-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := tmpfile.Close()
		if err != nil {
			fmt.Printf("closing temp mcp file failed, may cause issues with test: %s\n", err.Error())
		}
	}()

	_, err = tmpfile.WriteString(mcpYAML)
	Expect(err).NotTo(HaveOccurred())

	config, err := mcpfile.ParseMCPFile(tmpfile.Name())
	Expect(err).NotTo(HaveOccurred())

	// Clean up the temporary config file immediately since ParseMCPFile has read it
	err = os.Remove(tmpfile.Name())
	Expect(err).NotTo(HaveOccurred())

	return config
}

func createCLITestMCPConfig(port int) *mcpfile.MCPFile {
	By("creating CLI test MCP configuration")

	mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
servers:
  - name: test-cli-server
    version: "1.0"
    runtime:
      streamableHttpConfig:
        port: %d
        basePath: "/mcp"
      transportProtocol: streamablehttp
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
`, port)

	tmpfile, err := os.CreateTemp("", "mcp-cli-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := tmpfile.Close()
		if err != nil {
			fmt.Printf("closing temp mcp file failed, may cause issues with test: %s\n", err.Error())
		}
	}()

	_, err = tmpfile.WriteString(mcpYAML)
	Expect(err).NotTo(HaveOccurred())

	config, err := mcpfile.ParseMCPFile(tmpfile.Name())
	Expect(err).NotTo(HaveOccurred())

	// Clean up the temporary config file immediately since ParseMCPFile has read it
	err = os.Remove(tmpfile.Name())
	Expect(err).NotTo(HaveOccurred())

	return config
}
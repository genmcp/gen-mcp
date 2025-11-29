package test

import (
	"github.com/genmcp/gen-mcp/pkg/runtime"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	keycloakBaseURL = "http://localhost:8081"
	masterRealm     = "master"
	clientID        = "genmcp-client"
	testUsername    = "admin"
	testPassword    = "admin"
)

var _ = Describe("OAuth Integration", Ordered, func() {

	BeforeAll(func() {
		stopKeycloak()
		startKeycloakWithInit()
		createTestClient()
	})

	AfterAll(func() {
		stopKeycloak()
	})

	Describe("MCP Server with OAuth Protection", Ordered, func() {
		const (
			mcpServerURL  = "http://localhost:8018/mcp"
			mcpServerPort = 8018
		)

		var (
			backendServer       *httptest.Server
			callbackServer      *httptest.Server
			mcpConfig           *mcpserver.MCPServer
			mcpServerCancelFunc context.CancelFunc
		)

		BeforeEach(func() {
			backendServer = createMockBackendServer()
			callbackServer = createOAuthCallbackServer()
			mcpConfig = createTestMCPConfig(backendServer.URL, mcpServerPort)

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

			time.Sleep(500 * time.Millisecond)
		})

		AfterEach(func() {
			By("cleaning up test servers")
			if backendServer != nil {
				backendServer.Close()
			}
			if callbackServer != nil {
				callbackServer.Close()
			}
			if mcpServerCancelFunc != nil {
				mcpServerCancelFunc()
			}
		})

		Describe("Protected Resource Metadata", func() {
			It("should expose metadata endpoint", func() {
				By("making a request to the metadata endpoint")
				resp, err := http.Get("http://localhost:8018/.well-known/oauth-protected-resource")
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := resp.Body.Close()
					Expect(err).NotTo(HaveOccurred())
				}()

				By("returning HTTP 200 OK")
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})

			It("should include scopes from MCP file in scopes_supported", func() {
				By("making a request to the metadata endpoint")
				resp, err := http.Get("http://localhost:8018/.well-known/oauth-protected-resource")
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := resp.Body.Close()
					Expect(err).NotTo(HaveOccurred())
				}()

				By("returning HTTP 200 OK")
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("parsing the metadata response")
				var metadata map[string]any
				err = json.NewDecoder(resp.Body).Decode(&metadata)
				Expect(err).NotTo(HaveOccurred())

				By("verifying scopes_supported field contains required scopes")
				scopesSupported, ok := metadata["scopes_supported"].([]any)
				Expect(ok).To(BeTrue(), "scopes_supported should be an array")

				// Convert to string slice for easier comparison
				var scopeStrings []string
				for _, scope := range scopesSupported {
					if scopeStr, ok := scope.(string); ok {
						scopeStrings = append(scopeStrings, scopeStr)
					}
				}

				By("checking that required scopes from MCP file are present")
				Expect(scopeStrings).To(ContainElement("read"))
				Expect(scopeStrings).To(ContainElement("user:read"))
			})
		})

		Describe("Authentication Flow", func() {
			It("should deny requests without token (with mcp client)", func() {
				By("creating MCP client")
				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test oauth client",
					Version: "0.0.1",
				}, nil)

				transport := &mcp.StreamableClientTransport{
					Endpoint: mcpServerURL,
				}

				By("attempting to connect without token")
				_, err := client.Connect(context.Background(), transport, nil)

				By("returning authorization error")
				Expect(err).To(HaveOccurred())
				// The connection should fail due to missing authorization
			})

			It("should successfully initialize with valid token", func() {
				By("obtaining OAuth token via direct access grant")
				token := performDirectAccessGrant(clientID, testUsername, testPassword)
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("creating OAuth-enabled MCP client")
				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test oauth client",
					Version: "0.0.1",
				}, nil)

				oauthTransport := &OAuthRoundTripper{
					Transport:   http.DefaultTransport,
					AccessToken: token.AccessToken,
				}
				httpClient := &http.Client{Transport: oauthTransport}

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerURL,
					HTTPClient: httpClient,
				}

				By("connecting to MCP server with OAuth token")
				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				By("verifying successful initialization")
				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())
				Expect(initResult.ServerInfo).NotTo(BeNil())
			})

			It("should successfully execute tools", func() {
				By("obtaining OAuth token with required scopes")
				token := performDirectAccessGrantWithScopes(clientID, testUsername, testPassword, "openid profile read user:read")
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("creating OAuth-enabled MCP client")
				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test oauth client",
					Version: "0.0.1",
				}, nil)

				oauthTransport := &OAuthRoundTripper{
					Transport:   http.DefaultTransport,
					AccessToken: token.AccessToken,
				}
				httpClient := &http.Client{Transport: oauthTransport}

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerURL,
					HTTPClient: httpClient,
				}

				By("connecting and initializing MCP session")
				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())

				By("calling get_status tool successfully")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      "get_status",
					Arguments: map[string]any{},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Content).To(HaveLen(1))

				textResult, ok := result.Content[0].(*mcp.TextContent)
				Expect(ok).To(BeTrue())
				Expect(textResult.Text).To(MatchJSON(`{"status": "ok"}`))
			})

			It("should allow tool calls when user has required scopes", func() {
				By("obtaining OAuth token with required scopes")
				token := performDirectAccessGrantWithScopes(clientID, testUsername, testPassword, "openid profile read user:read")
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("creating OAuth-enabled MCP client")
				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test oauth client",
					Version: "0.0.1",
				}, nil)

				oauthTransport := &OAuthRoundTripper{
					Transport:   http.DefaultTransport,
					AccessToken: token.AccessToken,
				}
				httpClient := &http.Client{Transport: oauthTransport}

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerURL,
					HTTPClient: httpClient,
				}

				By("connecting and initializing MCP session")
				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				By("calling get_user tool with required scopes")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "get_user",
					Arguments: map[string]any{
						"userId": "test123",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Content).To(HaveLen(1))

				textResult, ok := result.Content[0].(*mcp.TextContent)
				Expect(ok).To(BeTrue())
				Expect(textResult.Text).To(MatchJSON(`{"status": "ok"}`))
			})

			It("should deny tool calls when user lacks required scopes", func() {
				By("obtaining OAuth token with limited scopes")
				token := performDirectAccessGrantWithScopes(clientID, testUsername, testPassword, "openid profile")
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("creating OAuth-enabled MCP client")
				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test oauth client",
					Version: "0.0.1",
				}, nil)

				oauthTransport := &OAuthRoundTripper{
					Transport:   http.DefaultTransport,
					AccessToken: token.AccessToken,
				}
				httpClient := &http.Client{Transport: oauthTransport}

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerURL,
					HTTPClient: httpClient,
				}

				By("connecting and initializing MCP session")
				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				By("attempting to call get_user tool without required scopes")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				_, err = session.CallTool(ctx, &mcp.CallToolParams{
					Name: "get_user",
					Arguments: map[string]any{
						"userId": "test123",
					},
				})

				By("returning authorization error")
				Expect(err).To(HaveOccurred())
				// Should fail due to missing required scopes
			})

			It("should deny unauthorized HTTP requests (direct HTTP request)", func() {
				By("making request without authorization header")
				resp, err := http.Get(mcpServerURL)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := resp.Body.Close()
					Expect(err).NotTo(HaveOccurred())
				}()

				By("returning HTTP 401 Unauthorized")
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

				By("including WWW-Authenticate header")
				wwwAuth := resp.Header.Get("WWW-Authenticate")
				Expect(wwwAuth).To(ContainSubstring("Bearer resource_metadata="))
				Expect(wwwAuth).To(ContainSubstring("http://localhost:8018/.well-known/oauth-protected-resource"))
			})
		})

		// TODO: Add authenticated client tests once new SDK client API is available
	})
})

// OAuthRoundTripper is a custom HTTP transport that adds OAuth Bearer tokens to requests
type OAuthRoundTripper struct {
	Transport   http.RoundTripper
	AccessToken string
}

func (ort *OAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())
	// Add the Authorization header
	req.Header.Set("Authorization", "Bearer "+ort.AccessToken)
	return ort.Transport.RoundTrip(req)
}

// OAuth token represents an OAuth access token
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// performDirectAccessGrant uses the Resource Owner Password Credentials Grant to get a token directly
func performDirectAccessGrant(clientID, username, password string) *OAuthToken {
	return performDirectAccessGrantWithScopes(clientID, username, password, "openid profile")
}

// performDirectAccessGrantWithScopes uses the Resource Owner Password Credentials Grant with custom scopes
func performDirectAccessGrantWithScopes(clientID, username, password, scopes string) *OAuthToken {
	// Use the direct access grant (password grant) to get tokens
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", clientID)
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("scope", scopes)

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakBaseURL, masterRealm)

	resp, err := http.PostForm(tokenURL, formData)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
	}()

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var token OAuthToken
	err = json.NewDecoder(resp.Body).Decode(&token)
	Expect(err).NotTo(HaveOccurred())

	return &token
}

// Helper functions for better test organization
func stopKeycloak() {
	By("stopping Keycloak if running")
	cmd := createKeycloakCommand("--stop")
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Printf("Warning: Failed to stop Keycloak: %v\n", err)
	}
}

func startKeycloakWithInit() {
	By("starting Keycloak with initialization")
	cmd := createKeycloakCommand("--init --start")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred(), "Failed to start Keycloak")
}

func createTestClient() {
	By("creating test client in Keycloak")
	cmd := createKeycloakCommand("--add-client master genmcp-client")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred(), "Failed to create test client")

	By("adding custom scopes to master realm")
	addCustomScope("read")
	addCustomScope("user:read")

	By("assigning custom scopes to test client")
	assignScopeToClient("read")
	assignScopeToClient("user:read")
}

func addCustomScope(scopeName string) {
	By(fmt.Sprintf("adding custom scope '%s' to master realm", scopeName))
	cmd := createKeycloakCommand(fmt.Sprintf("--add-scope master %s", scopeName))
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to add scope %s", scopeName))
}

func assignScopeToClient(scopeName string) {
	By(fmt.Sprintf("assigning scope '%s' to client '%s'", scopeName, clientID))
	cmd := createKeycloakCommand(fmt.Sprintf("--assign-scope master %s %s", clientID, scopeName))
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to assign scope %s to client %s", scopeName, clientID))
}

func createKeycloakCommand(args string) *exec.Cmd {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("./hack/keycloak.sh %s", args))
	cmd.Dir = "../"
	cmd.Env = os.Environ()
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	return cmd
}

func createMockBackendServer() *httptest.Server {
	By("creating mock backend server")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, `{"status": "ok"}`)
		Expect(err).NotTo(HaveOccurred())
	}))
}

func createOAuthCallbackServer() *httptest.Server {
	By("creating OAuth callback server")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "OAuth callback received")
		Expect(err).NotTo(HaveOccurred())
	}))
}

func createTestMCPConfig(backendURL string, port int) *mcpserver.MCPServer {
	By("creating test MCP configuration")

	toolDefsYAML := fmt.Sprintf(`
kind: MCPToolDefinitions
schemaVersion: 0.2.0
name: test-oauth-server-full-flow
version: "1.0"
tools:
  - name: get_status
    description: "Get server status"
    inputSchema:
      type: object
      properties: {}
    outputSchema:
      type: object
      properties:
        status:
          type: string
    invocation:
      http:
        url: "%s/status"
        method: "GET"
  - name: get_user
    description: "Get user by ID"
    inputSchema:
      type: object
      properties:
        userId:
          type: string
      required:
        - userId
    outputSchema:
      type: object
      properties:
        status:
          type: string
    invocation:
      http:
        url: "%s/users/{userId}"
        method: "GET"
    requiredScopes:
      - "read"
      - "user:read"
`, backendURL, backendURL)

	serverConfigYAML := fmt.Sprintf(`
kind: MCPServerConfig
schemaVersion: 0.2.0
runtime:
  streamableHttpConfig:
    port: %d
    basePath: "/mcp"
    auth:
      authorizationServers:
        - %s/realms/%s
      jwksUri: "%s/realms/%s/protocol/openid-connect/certs"
  transportProtocol: streamablehttp
`, port, keycloakBaseURL, masterRealm, keycloakBaseURL, masterRealm)

	toolDefsFile, err := os.CreateTemp("", "mcp-oauth-tooldefs-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := os.Remove(toolDefsFile.Name())
		Expect(err).NotTo(HaveOccurred())
	}()

	_, err = toolDefsFile.WriteString(toolDefsYAML)
	Expect(err).NotTo(HaveOccurred())
	_ = toolDefsFile.Close()

	serverConfigFile, err := os.CreateTemp("", "mcp-oauth-serverconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := os.Remove(serverConfigFile.Name())
		Expect(err).NotTo(HaveOccurred())
	}()

	_, err = serverConfigFile.WriteString(serverConfigYAML)
	Expect(err).NotTo(HaveOccurred())
	_ = serverConfigFile.Close()

	toolDefs, err := definitions.ParseMCPFile(toolDefsFile.Name())
	Expect(err).NotTo(HaveOccurred())

	serverConfig, err := serverconfig.ParseMCPFile(serverConfigFile.Name())
	Expect(err).NotTo(HaveOccurred())

	return &mcpserver.MCPServer{
		MCPToolDefinitions: toolDefs.MCPToolDefinitions,
		MCPServerConfig:    serverConfig.MCPServerConfig,
	}
}

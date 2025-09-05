package test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	keycloakBaseURL = "http://localhost:8080"
	masterRealm     = "master"
	clientID        = "genmcp-client"
	testUsername    = "admin"
	testPassword    = "admin"
)

func TestOAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OAuth Suite")
}

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
			mcpConfig           *mcpfile.MCPFile
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
				err := mcpserver.RunServer(ctx, mcpConfig.Servers[0])
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
				By("creating OAuth MCP client")
				tokenStore := mcpclient.NewMemoryTokenStore()
				client := createOAuthMCPClientWithTokenStore(mcpServerURL, clientID, tokenStore)
				defer func() {
					err := client.Close()
					Expect(err).NotTo(HaveOccurred())
				}()

				By("attempting to initialize without token")
				initRequest := createInitRequest()
				_, err := client.Initialize(context.Background(), initRequest)

				By("returning OAuth authorization required error")
				Expect(mcpclient.IsOAuthAuthorizationRequiredError(err)).To(BeTrue())
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

		Context("With authenticated client", func() {
			var (
				tokenStore mcpclient.TokenStore
				client     *mcpclient.Client
				token      *mcpclient.Token
			)

			BeforeEach(func() {
				By("creating OAuth MCP client with shared token store")
				tokenStore = mcpclient.NewMemoryTokenStore()
				client = createOAuthMCPClientWithTokenStore(mcpServerURL, clientID, tokenStore)

				By("obtaining OAuth token via direct access grant")
				token = performDirectAccessGrant(clientID, testUsername, testPassword)
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("storing token in shared token store")
				err := tokenStore.SaveToken(token)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				if client != nil {
					err := client.Close()
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should successfully initialize with valid token", func() {
				By("attempting to initialize with valid token")
				initRequest := createInitRequest()
				_, err := client.Initialize(context.Background(), initRequest)

				By("completing initialization successfully")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should successfully execute tools", func() {
				By("initializing the client first")
				initRequest := createInitRequest()
				_, err := client.Initialize(context.Background(), initRequest)
				Expect(err).NotTo(HaveOccurred())

				By("calling a tool successfully")
				result := callGetStatusTool(client)

				By("returning expected tool result")
				validateToolResult(result)
			})

			It("should handle multiple tool calls", func() {
				By("initializing the client first")
				initRequest := createInitRequest()
				_, err := client.Initialize(context.Background(), initRequest)
				Expect(err).NotTo(HaveOccurred())

				By("calling the same tool multiple times")
				for i := 1; i <= 3; i++ {
					result := callGetStatusTool(client)
					validateToolResult(result)
				}
			})
		})

		Context("With authenticated client having sufficient scopes", func() {
			var (
				tokenStore mcpclient.TokenStore
				client     *mcpclient.Client
				token      *mcpclient.Token
			)

			BeforeEach(func() {
				By("creating OAuth MCP client with shared token store")
				tokenStore = mcpclient.NewMemoryTokenStore()
				client = createOAuthMCPClientWithTokenStore(mcpServerURL, clientID, tokenStore)

				By("obtaining OAuth token with required scopes via direct access grant")
				token = performDirectAccessGrantWithScopes(clientID, testUsername, testPassword, "openid profile read user:read")
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("storing token in shared token store")
				err := tokenStore.SaveToken(token)
				Expect(err).NotTo(HaveOccurred())

				By("initializing the client")
				initRequest := createInitRequest()
				_, err = client.Initialize(context.Background(), initRequest)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				if client != nil {
					err := client.Close()
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should allow tool calls when user has required scopes", func() {
				By("calling tool that requires 'read' and 'user:read' scopes")
				result := callGetUserTool(client, "authorized-user")

				By("returning successful tool result")
				validateToolResult(result)
			})
		})

		Context("With authenticated client having insufficient scopes", func() {
			var (
				tokenStore mcpclient.TokenStore
				client     *mcpclient.Client
				token      *mcpclient.Token
			)

			BeforeEach(func() {
				By("creating OAuth MCP client with shared token store")
				tokenStore = mcpclient.NewMemoryTokenStore()
				client = createOAuthMCPClientWithTokenStore(mcpServerURL, clientID, tokenStore)

				By("obtaining OAuth token with limited scopes via direct access grant")
				token = performDirectAccessGrantWithScopes(clientID, testUsername, testPassword, "openid profile")
				Expect(token.AccessToken).NotTo(BeEmpty())

				By("storing token in shared token store")
				err := tokenStore.SaveToken(token)
				Expect(err).NotTo(HaveOccurred())

				By("initializing the client")
				initRequest := createInitRequest()
				_, err = client.Initialize(context.Background(), initRequest)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				if client != nil {
					err := client.Close()
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should deny tool calls when user lacks required scopes", func() {
				By("calling tool that requires 'read' and 'user:read' scopes")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				toolCall := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "get_user",
						Arguments: map[string]any{"userId": "unauthorized-user"},
					},
				}

				By("returning authorization error")
				_, err := client.CallTool(ctx, toolCall)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("forbidden"))
			})
		})
	})
})

// performDirectAccessGrant uses the Resource Owner Password Credentials Grant to get a token directly
func performDirectAccessGrant(clientID, username, password string) *mcpclient.Token {
	// Use the direct access grant (password grant) to get tokens
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", clientID)
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("scope", "openid profile")

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakBaseURL, masterRealm)

	resp, err := http.PostForm(tokenURL, formData)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		GinkgoWriter.Println("Expected 200 from token endpoint but got %d. Response body: %s", resp.StatusCode, string(body))
	}

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// Parse the token response
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(tokenResponse.AccessToken).NotTo(BeEmpty(), "Access token should not be empty")

	// Convert to the MCP client token format
	token := &mcpclient.Token{
		AccessToken:  tokenResponse.AccessToken,
		TokenType:    tokenResponse.TokenType,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresIn:    tokenResponse.ExpiresIn,
		Scope:        tokenResponse.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}

	return token
}

// performDirectAccessGrantWithScopes uses the Resource Owner Password Credentials Grant with custom scopes
func performDirectAccessGrantWithScopes(clientID, username, password, scopes string) *mcpclient.Token {
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		GinkgoWriter.Println("Expected 200 from token endpoint but got %d. Response body: %s", resp.StatusCode, string(body))
	}

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// Parse the token response
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(tokenResponse.AccessToken).NotTo(BeEmpty(), "Access token should not be empty")

	// Convert to the MCP client token format
	token := &mcpclient.Token{
		AccessToken:  tokenResponse.AccessToken,
		TokenType:    tokenResponse.TokenType,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresIn:    tokenResponse.ExpiresIn,
		Scope:        tokenResponse.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}

	return token
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

func createTestMCPConfig(backendURL string, port int) *mcpfile.MCPFile {
	By("creating test MCP configuration")

	mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
servers:
  - name: test-oauth-server-full-flow
    version: "1.0"
    runtime:
      streamableHttpConfig:
        port: %d
        basePath: "/mcp"
        auth:
          authorizationServers:
            - %s/realms/%s
          jwksUri: "%s/realms/%s/protocol/openid-connect/certs"
      transportProtocol: streamablehttp
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
`, port, keycloakBaseURL, masterRealm, keycloakBaseURL, masterRealm, backendURL, backendURL)

	tmpfile, err := os.CreateTemp("", "mcp-oauth-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := os.Remove(tmpfile.Name())
		Expect(err).NotTo(HaveOccurred())
	}()

	_, err = tmpfile.WriteString(mcpYAML)
	Expect(err).NotTo(HaveOccurred())
	err = tmpfile.Close()
	Expect(err).NotTo(HaveOccurred())

	config, err := mcpfile.ParseMCPFile(tmpfile.Name())
	Expect(err).NotTo(HaveOccurred())
	return config
}

func createOAuthMCPClientWithTokenStore(serverURL, clientID string, tokenStore mcpclient.TokenStore) *mcpclient.Client {
	By("creating OAuth MCP client with token store")
	oauthConfig := mcpclient.OAuthConfig{
		ClientID:    clientID,
		RedirectURI: "http://localhost:8080/callback",
		Scopes:      []string{"openid", "profile"},
		TokenStore:  tokenStore,
		PKCEEnabled: true,
	}
	client, err := mcpclient.NewOAuthStreamableHttpClient(serverURL, oauthConfig)
	Expect(err).NotTo(HaveOccurred())
	return client
}

func createInitRequest() mcp.InitializeRequest {
	return mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test oauth client",
				Version: "0.0.1",
			},
		},
	}
}

func callGetStatusTool(client *mcpclient.Client) *mcp.CallToolResult {
	By("calling get_status tool")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolCall := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_status",
			Arguments: map[string]any{},
		},
	}

	result, err := client.CallTool(ctx, toolCall)
	Expect(err).NotTo(HaveOccurred())
	return result
}

func callGetUserTool(client *mcpclient.Client, userID string) *mcp.CallToolResult {
	By(fmt.Sprintf("calling get_user tool with userID: %s", userID))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolCall := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_user",
			Arguments: map[string]any{"userId": userID},
		},
	}

	result, err := client.CallTool(ctx, toolCall)
	Expect(err).NotTo(HaveOccurred())
	return result
}

func validateToolResult(result *mcp.CallToolResult) {
	By("validating tool call result")
	Expect(result).NotTo(BeNil())
	Expect(result.Content).To(HaveLen(1))

	textResult, ok := result.Content[0].(mcp.TextContent)
	Expect(ok).To(BeTrue())
	Expect(textResult.Text).To(MatchJSON(`{"status": "ok"}`))
}

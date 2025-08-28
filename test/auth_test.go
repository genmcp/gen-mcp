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

func TestOAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OAuth Suite")
}

var _ = Describe("OAuth", Ordered, func() {

	BeforeAll(func() {
		By("Restart Keycloak, by first making sure it is not running")
		stopCmd := exec.Command("bash", "-c", "./hack/keycloak.sh --stop")
		stopCmd.Dir = "../" // Run from repo root
		stopCmd.Env = os.Environ()
		stopCmd.Stdout = GinkgoWriter
		stopCmd.Stderr = GinkgoWriter
		err := stopCmd.Run()
		Expect(err).NotTo(HaveOccurred(), "Failed to stop1 Keycloak")

		By("Starting Keycloak with initialization")
		cmd := exec.Command("bash", "-c", "./hack/keycloak.sh --init --start")
		cmd.Dir = "../" // Run from repo root
		cmd.Env = os.Environ()
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		err = cmd.Run()
		Expect(err).NotTo(HaveOccurred(), "Failed to start Keycloak")

		By("Creating the genmcp-client in the master realm for testing")
		clientCmd := exec.Command("bash", "-c", "./hack/keycloak.sh --add-client master genmcp-client")
		clientCmd.Dir = "../"
		clientCmd.Env = os.Environ()
		clientCmd.Stdout = GinkgoWriter
		clientCmd.Stderr = GinkgoWriter
		err = clientCmd.Run()
		Expect(err).NotTo(HaveOccurred(), "Failed to add genmcp-client")
	})

	AfterAll(func() {
		By("Stopping Keycloak")

		cmd := exec.Command("bash", "-c", "./hack/keycloak.sh --stop")
		cmd.Dir = "../"
		cmd.Env = os.Environ()
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter

		err := cmd.Run()
		if err != nil {
			GinkgoWriter.Printf("Warning: Failed to stop Keycloak: %v\n", err)
		}
	})

	Describe("MCP server with OAuth enabled", Ordered, func() {

		var backendServer, callbackServer *httptest.Server
		var mcpConfig *mcpfile.MCPFile
		var mcpServerCancelFunc context.CancelFunc

		BeforeEach(func() {
			// Create a mock HTTP server for the backend API
			// Note: This backend API doesn't need OAuth - the OAuth is for the MCP server itself
			backendServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, `{"status": "ok"}`)
			}))

			// Create a callback server to handle OAuth redirects
			callbackServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "OAuth callback received")
			}))

			// Define the MCP file with OAuth configuration using Keycloak
			mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
servers:
  - name: test-oauth-server-full-flow
    version: "1.0"
    runtime:
      streamableHttpConfig:
        port: 8018
        basePath: "/mcp"
        auth:
          authorizationServers:
            - http://localhost:8080/realms/master
          scopesSupported:
            - "read"
            - "write"
            - "admin"
          bearerMethodsSupported:
            - "header"
            - "body"
          jwksUri: "http://localhost:8080/realms/master/protocol/openid-connect/certs"
      transportProtocol: streamablehttp
    tools:
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
`, backendServer.URL)

			// Write the MCP file to a temporary file
			tmpfile, err := os.CreateTemp("", "mcp-oauth-full-*.yaml")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.WriteString(mcpYAML)
			Expect(err).NotTo(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).NotTo(HaveOccurred())

			// Parse the MCP file
			mcpConfig, err = mcpfile.ParseMCPFile(tmpfile.Name())
			Expect(err).NotTo(HaveOccurred())

			// Create cancelable context for the MCP server
			ctx := context.Background()
			ctx, mcpServerCancelFunc = context.WithCancel(ctx)

			// Start the MCP server using RunServer
			go func() {
				defer GinkgoRecover()

				err := mcpserver.RunServer(ctx, mcpConfig.Servers[0])
				if err != nil && !strings.Contains(err.Error(), "Server closed") {
					Expect(err).NotTo(HaveOccurred(), "Failed to start mcp server")
				}
			}()

			// Give the server time to start
			time.Sleep(500 * time.Millisecond)
		})

		AfterEach(func() {
			backendServer.Close()
			callbackServer.Close()
			// Cancel the context to shut down the MCP server
			if mcpServerCancelFunc != nil {
				mcpServerCancelFunc()
			}
		})

		It("Should have a protected resource metadata endpoint", func() {
			// Verify the protected resource metadata endpoint is working
			resp, err := http.Get("http://localhost:8018/.well-known/oauth-protected-resource")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("Should accept authenticated requests", func() {
			// Create an MCP client with OAuth configuration
			mcpServerURL := "http://localhost:8018/mcp"

			oauthConfig := mcpclient.OAuthConfig{
				ClientID:    "genmcp-client",
				RedirectURI: "http://localhost:8080/callback", // Use a fixed redirect URI that should be configured
				Scopes:      []string{"openid", "profile"},
				TokenStore:  mcpclient.NewMemoryTokenStore(),
				PKCEEnabled: true,
				// Let the client discover auth server automatically
			}

			client, err := mcpclient.NewOAuthStreamableHttpClient(mcpServerURL, oauthConfig)
			Expect(err).NotTo(HaveOccurred())
			defer client.Close()

			// Try to initialize - should trigger OAuth flow
			initRequest := mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "test oauth client full flow",
						Version: "0.0.1",
					},
				},
			}

			ctx := context.Background()

			_, err = client.Initialize(ctx, initRequest)
			Expect(mcpclient.IsOAuthAuthorizationRequiredError(err)).To(BeTrue(), "Expected OAuth authorization required error, got: %v", err)

			// Use direct access grant to get a token from Keycloak
			GinkgoWriter.Println("Using direct access grant to obtain OAuth tokens from Keycloak")

			token := performDirectAccessGrant("genmcp-client", "admin", "admin")
			GinkgoWriter.Println("Successfully obtained access token: %s...", token.AccessToken[:20])

			// 11. Store the token in the OAuth client's token store
			err = oauthConfig.TokenStore.SaveToken(token)
			Expect(err).NotTo(HaveOccurred(), "Failed to save token to token store")

			// 13. Now try to initialize the MCP client again with the obtained token
			_, err = client.Initialize(ctx, initRequest)
			Expect(err).NotTo(HaveOccurred(), "Client should connect successfully with OAuth token")

			GinkgoWriter.Println("MCP client initialized successfully with OAuth token")

			// 14. Call a tool to verify the complete flow works
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			toolCall := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "get_user",
					Arguments: map[string]any{
						"userId": "123",
					},
				},
			}

			res, err := client.CallTool(ctx, toolCall)
			Expect(err).NotTo(HaveOccurred(), "Tool call should succeed with OAuth token")

			// 15. Verify the result
			Expect(res).NotTo(BeNil())
			Expect(res.Content).To(HaveLen(1))

			textResult, ok := res.Content[0].(mcp.TextContent)
			Expect(ok).To(BeTrue())
			Expect(textResult.Text).To(MatchJSON(`{"status": "ok"}`))

			GinkgoWriter.Println("Complete OAuth integration test passed - full flow from authentication to tool call successful")
		})

		It("Should deny unauthorized requests", func() {
			// Test unauthorized request returns 401 with proper WWW-Authenticate header
			resp, err := http.Get("http://localhost:8018/mcp")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			wwwAuth := resp.Header.Get("WWW-Authenticate")
			Expect(wwwAuth).To(ContainSubstring("Bearer resource_metadata="))
			Expect(wwwAuth).To(ContainSubstring("http://localhost:8018/.well-known/oauth-protected-resource"))

			GinkgoWriter.Println("OAuth protection working correctly - unauthorized requests properly rejected")
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

	tokenURL := "http://localhost:8080/realms/master/protocol/openid-connect/token"

	resp, err := http.PostForm(tokenURL, formData)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

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

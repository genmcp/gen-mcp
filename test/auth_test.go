package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// performDirectAccessGrant uses the Resource Owner Password Credentials Grant to get a token directly
func performDirectAccessGrant(t *testing.T, clientID, username, password string) *mcpclient.Token {
	// Use the direct access grant (password grant) to get tokens
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", clientID)
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("scope", "openid profile")

	tokenURL := "http://localhost:8080/realms/master/protocol/openid-connect/token"

	resp, err := http.PostForm(tokenURL, formData)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Expected 200 from token endpoint but got %d. Response body: %s", resp.StatusCode, string(body))
	}
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the token response
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	require.NoError(t, err)

	require.NotEmpty(t, tokenResponse.AccessToken, "Access token should not be empty")

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

func TestOAuthIntegrationFullFlow(t *testing.T) {
	// 1. Create a mock HTTP server for the backend API
	// Note: This backend API doesn't need OAuth - the OAuth is for the MCP server itself
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/123", r.URL.Path)
		fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer httpServer.Close()

	// 2. Create a callback server to handle OAuth redirects
	callbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OAuth callback received")
	}))
	defer callbackServer.Close()

	// 3. Define the MCP file with OAuth configuration using Keycloak
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
`, httpServer.URL)

	// 4. Write the MCP file to a temporary file
	tmpfile, err := os.CreateTemp("", "mcp-oauth-full-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(mcpYAML)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	// 5. Parse the MCP file
	mcpConfig, err := mcpfile.ParseMCPFile(tmpfile.Name())
	require.NoError(t, err)

	// 6. Start the MCP server using RunServer
	go func() {
		err := mcpserver.RunServer(mcpConfig.Servers[0])
		if err != nil && !strings.Contains(err.Error(), "Server closed") {
			t.Errorf("MCP server error: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(500 * time.Millisecond)

	// 7. Verify the protected resource metadata endpoint is working
	resp, err := http.Get("http://localhost:8018/.well-known/oauth-protected-resource")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Create an MCP client with OAuth configuration
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
	require.NoError(t, err)

	defer func() {
		err := client.Close()
		require.NoError(t, err, "closing the client should not fail")
	}()

	// 9. Try to initialize - should trigger OAuth flow
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
	require.True(t, mcpclient.IsOAuthAuthorizationRequiredError(err),
		"Expected OAuth authorization required error, got: %v", err)

	// 10. Use direct access grant to get a token from Keycloak
	t.Log("Using direct access grant to obtain OAuth tokens from Keycloak")

	token := performDirectAccessGrant(t, "genmcp-client", "admin", "admin")
	t.Logf("Successfully obtained access token: %s...", token.AccessToken[:20])

	// 11. Store the token in the OAuth client's token store
	err = oauthConfig.TokenStore.SaveToken(token)
	require.NoError(t, err, "Failed to save token to token store")

	// 13. Now try to initialize the MCP client again with the obtained token
	_, err = client.Initialize(ctx, initRequest)
	require.NoError(t, err, "Client should connect successfully with OAuth token")

	t.Log("MCP client initialized successfully with OAuth token")

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
	require.NoError(t, err, "Tool call should succeed with OAuth token")

	// 15. Verify the result
	require.NotNil(t, res)
	require.Len(t, res.Content, 1)
	require.IsType(t, res.Content[0], mcp.TextContent{})

	textResult, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)

	assert.JSONEq(t, `{"status": "ok"}`, textResult.Text)

	t.Log("Complete OAuth integration test passed - full flow from authentication to tool call successful")
}

func TestOAuthIntegrationUnauthorized(t *testing.T) {
	// Test that the MCP server properly rejects unauthorized requests

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer httpServer.Close()

	mcpYAML := fmt.Sprintf(`
mcpFileVersion: 0.0.1
servers:
  - name: test-oauth-server-unauth
    version: "1.0"
    runtime:
      streamableHttpConfig:
        port: 8019
        basePath: "/mcp"
        auth:
          authorizationServers:
            - http://localhost:8080/realms/master
          scopesSupported:
            - "read"
          bearerMethodsSupported:
            - "header"
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
        invocation:
          http:
            url: "%s/users/{userId}"
            method: "GET"
`, httpServer.URL)

	tmpfile, err := os.CreateTemp("", "mcp-oauth-unauth-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(mcpYAML)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	mcpConfig, err := mcpfile.ParseMCPFile(tmpfile.Name())
	require.NoError(t, err)

	go func() {
		err := mcpserver.RunServer(mcpConfig.Servers[0])
		if err != nil && !strings.Contains(err.Error(), "Server closed") {
			t.Errorf("MCP server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Test unauthorized request returns 401 with proper WWW-Authenticate header
	resp, err := http.Get("http://localhost:8019/mcp")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	wwwAuth := resp.Header.Get("WWW-Authenticate")
	assert.Contains(t, wwwAuth, "Bearer resource_metadata=")
	assert.Contains(t, wwwAuth, "http://localhost:8019/.well-known/oauth-protected-resource")

	// Test protected resource metadata endpoint is available
	resp, err = http.Get("http://localhost:8019/.well-known/oauth-protected-resource")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	t.Log("OAuth protection working correctly - unauthorized requests properly rejected")
}

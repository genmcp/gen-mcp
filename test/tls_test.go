package test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var _ = Describe("TLS Integration", Ordered, func() {
	var (
		certFile string
		keyFile  string
	)

	BeforeAll(func() {
		var err error
		certFile, keyFile, err = generateTestCertificates()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		if certFile != "" {
			err := os.Remove(certFile)
			if err != nil {
				fmt.Printf("failed to clean up cert file after test: %s\n", certFile)
			}
		}
		if keyFile != "" {
			err := os.Remove(keyFile)
			if err != nil {
				fmt.Printf("failed to clean up key file after test: %s\n", certFile)
			}
		}
	})

	Describe("MCP Server with TLS", Ordered, func() {
		const (
			mcpServerPort     = 8019
			mcpServerHTTPSURL = "https://localhost:8019/mcp"
			mcpServerHTTPURL  = "http://localhost:8019/mcp"
		)

		var (
			backendServer       *httptest.Server
			mcpConfig           *mcpfile.MCPServer
			mcpServerCancelFunc context.CancelFunc
		)

		BeforeEach(func() {
			backendServer = createMockBackendServer()
			mcpConfig = createTestTLSMCPConfig(backendServer.URL, mcpServerPort, certFile, keyFile)

			By("starting MCP server with TLS")
			ctx := context.Background()
			ctx, mcpServerCancelFunc = context.WithCancel(ctx)

			go func() {
				defer GinkgoRecover()
				err := mcpserver.RunServer(ctx, mcpConfig)
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

		Describe("TLS Connection Tests", func() {
			It("should accept HTTPS connections when TLS is configured", func() {
				By("creating TLS client that accepts self-signed certificates")
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: tr}

				By("making HTTPS request to TLS-enabled server")
				resp, err := client.Get(mcpServerHTTPSURL)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = resp.Body.Close()
				}()

				By("receiving successful response (even if unauthorized for missing auth)")
				// The server should respond (not reject the TLS connection)
				// It may return 401 if auth is required, but the TLS handshake should succeed
				Expect(resp.StatusCode).To(SatisfyAny(
					Equal(http.StatusOK),
					Equal(http.StatusUnauthorized),
					Equal(http.StatusBadRequest),
				))
			})

			It("should reject HTTP connections when TLS is configured", func() {
				By("making HTTP request to TLS-enabled server")
				client := &http.Client{
					Timeout: 2 * time.Second,
				}

				resp, err := client.Get(mcpServerHTTPURL)

				By("failing to connect via HTTP")
				// When making an HTTP request to an HTTPS server, we should get an error
				// OR a response with a redirect to HTTPS (some servers handle this)
				if err == nil {
					defer func() {
						_ = resp.Body.Close()
					}()
					// If no error, check if we got a redirect or error status
					if resp.StatusCode < 400 {
						Fail(fmt.Sprintf("Expected error or 4xx/5xx status but got %d when making HTTP request to HTTPS server", resp.StatusCode))
					}
					// If we got an error status, that's acceptable - the server rejected the HTTP connection
				} else {
					// Expected case: we should get an error
					Expect(err.Error()).To(SatisfyAny(
						ContainSubstring("connection refused"),
						ContainSubstring("connection reset"),
						ContainSubstring("malformed HTTP"),
						ContainSubstring("EOF"),
						ContainSubstring("tls"),
						ContainSubstring("TLS"),
						ContainSubstring("SSL"),
						ContainSubstring("certificate"),
						ContainSubstring("bad record MAC"),
						ContainSubstring("unexpected message"),
					))
				}
			})

			It("should successfully complete MCP handshake over TLS", func() {
				By("creating MCP client with TLS transport")
				httpTransport := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				httpClient := &http.Client{Transport: httpTransport}

				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test tls client",
					Version: "0.0.1",
				}, nil)

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerHTTPSURL,
					HTTPClient: httpClient,
				}

				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				By("successfully initializing MCP connection over TLS")
				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())
			})

			It("should successfully call tools over TLS", func() {
				By("creating MCP client with TLS transport")
				httpTransport := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				httpClient := &http.Client{Transport: httpTransport}

				client := mcp.NewClient(&mcp.Implementation{
					Name:    "test tls client",
					Version: "0.0.1",
				}, nil)

				transport := &mcp.StreamableClientTransport{
					Endpoint:   mcpServerHTTPSURL,
					HTTPClient: httpClient,
				}

				session, err := client.Connect(context.Background(), transport, nil)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					_ = session.Close()
				}()

				By("initializing MCP connection")
				initResult := session.InitializeResult()
				Expect(initResult).NotTo(BeNil())

				By("calling tool successfully over TLS")
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
		})
	})
})

// generateTestCertificates creates a self-signed certificate and private key for testing
func generateTestCertificates() (certFile, keyFile string, err error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Write certificate to temporary file
	certTempFile, err := os.CreateTemp("", "test-cert-*.pem")
	if err != nil {
		return "", "", err
	}
	defer func() {
		err := certTempFile.Close()
		if err != nil {
			fmt.Printf("warning: failed to close cert temp file after writing, may cause issues: %s\n", err.Error())
		}
	}()

	err = pem.Encode(certTempFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	if err != nil {
		removeErr := os.Remove(certTempFile.Name())
		err := errors.Join(err, removeErr)
		return "", "", err
	}

	// Write private key to temporary file
	keyTempFile, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		removeErr := os.Remove(certTempFile.Name())
		err := errors.Join(err, removeErr)
		return "", "", err
	}
	defer func() {
		err := keyTempFile.Close()
		if err != nil {
			fmt.Printf("warning: failed to close key temp file after writing, may cause issues: %s\n", err.Error())
		}
	}()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		removeErr := os.Remove(certTempFile.Name())
		err = errors.Join(err, removeErr)
		removeErr = os.Remove(keyTempFile.Name())
		err = errors.Join(err, removeErr)
		return "", "", err
	}

	err = pem.Encode(keyTempFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})
	if err != nil {
		removeErr := os.Remove(certTempFile.Name())
		err = errors.Join(err, removeErr)
		removeErr = os.Remove(keyTempFile.Name())
		err = errors.Join(err, removeErr)
		return "", "", err
	}

	return certTempFile.Name(), keyTempFile.Name(), nil
}

func createTestTLSMCPConfig(backendURL string, port int, certFile, keyFile string) *mcpfile.MCPServer {
	By("creating test MCP configuration with TLS")

	toolDefsYAML := fmt.Sprintf(`
kind: MCPToolDefinitions
schemaVersion: 0.2.0
name: test-tls-server
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
`, backendURL)

	serverConfigYAML := fmt.Sprintf(`
kind: MCPServerConfig
schemaVersion: 0.2.0
name: test-tls-server
version: "1.0"
runtime:
  streamableHttpConfig:
    port: %d
    basePath: "/mcp"
    tls:
      certFile: %s
      keyFile: %s
  transportProtocol: streamablehttp
`, port, certFile, keyFile)

	toolDefsFile, err := os.CreateTemp("", "mcp-tls-tooldefs-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(toolDefsFile.Name())

	_, err = toolDefsFile.WriteString(toolDefsYAML)
	Expect(err).NotTo(HaveOccurred())
	toolDefsFile.Close()

	serverConfigFile, err := os.CreateTemp("", "mcp-tls-serverconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(serverConfigFile.Name())

	_, err = serverConfigFile.WriteString(serverConfigYAML)
	Expect(err).NotTo(HaveOccurred())
	serverConfigFile.Close()

	toolDefs, err := definitions.ParseMCPFile(toolDefsFile.Name())
	Expect(err).NotTo(HaveOccurred())

	serverConfig, err := serverconfig.ParseMCPFile(serverConfigFile.Name())
	Expect(err).NotTo(HaveOccurred())

	return &mcpfile.MCPServer{
		MCPToolDefinitions: toolDefs.MCPToolDefinitions,
		MCPServerConfig:    serverConfig.MCPServerConfig,
	}
}

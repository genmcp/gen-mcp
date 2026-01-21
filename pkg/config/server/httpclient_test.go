package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientTLSConfig_BuildTLSConfig(t *testing.T) {
	tt := []struct {
		name           string
		config         *ClientTLSConfig
		setupFunc      func(t *testing.T) (cleanup func())
		expectError    bool
		errorContains  string
		validateConfig func(t *testing.T, config *ClientTLSConfig)
	}{
		{
			name:   "nil config returns nil",
			config: nil,
		},
		{
			name:   "empty config returns valid TLS config",
			config: &ClientTLSConfig{},
		},
		{
			name: "insecureSkipVerify is set correctly",
			config: &ClientTLSConfig{
				InsecureSkipVerify: true,
			},
			validateConfig: func(t *testing.T, config *ClientTLSConfig) {
				tlsConfig, err := config.BuildTLSConfig()
				require.NoError(t, err)
				assert.True(t, tlsConfig.InsecureSkipVerify)
			},
		},
		{
			name: "invalid CA cert file path returns error",
			config: &ClientTLSConfig{
				CACertFiles: []string{"/nonexistent/path/to/ca.pem"},
			},
			expectError:   true,
			errorContains: "failed to load CA cert",
		},
		{
			name: "invalid CA cert directory returns error",
			config: &ClientTLSConfig{
				CACertDir: "/nonexistent/directory",
			},
			expectError:   true,
			errorContains: "failed to load CA certs from directory",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				cleanup := tc.setupFunc(t)
				defer cleanup()
			}

			if tc.validateConfig != nil {
				tc.validateConfig(t, tc.config)
				return
			}

			tlsConfig, err := tc.config.BuildTLSConfig()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tc.config != nil {
					assert.NotNil(t, tlsConfig)
				}
			}
		})
	}
}

func TestClientTLSConfig_LoadCACertFiles(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "genmcp-test-certs")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate a test CA certificate
	certPEM := generateTestCACert(t)
	certPath := filepath.Join(tmpDir, "test-ca.pem")
	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	config := &ClientTLSConfig{
		CACertFiles: []string{certPath},
	}

	tlsConfig, err := config.BuildTLSConfig()
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotNil(t, tlsConfig.RootCAs)
}

func TestClientTLSConfig_LoadCACertDir(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "genmcp-test-certs")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate and write test CA certificates
	cert1PEM := generateTestCACert(t)
	cert2PEM := generateTestCACert(t)

	err = os.WriteFile(filepath.Join(tmpDir, "ca1.pem"), cert1PEM, 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "ca2.crt"), cert2PEM, 0644)
	require.NoError(t, err)

	// Also write a non-cert file that should be ignored
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not a cert"), 0644)
	require.NoError(t, err)

	config := &ClientTLSConfig{
		CACertDir: tmpDir,
	}

	tlsConfig, err := config.BuildTLSConfig()
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotNil(t, tlsConfig.RootCAs)
}

func TestServerRuntime_GetHTTPClient(t *testing.T) {
	tt := []struct {
		name        string
		runtime     *ServerRuntime
		expectError bool
	}{
		{
			name:    "nil runtime returns default client",
			runtime: nil,
		},
		{
			name:    "runtime without ClientTLSConfig returns client",
			runtime: &ServerRuntime{},
		},
		{
			name: "runtime with valid ClientTLSConfig returns client",
			runtime: &ServerRuntime{
				ClientTLSConfig: &ClientTLSConfig{
					InsecureSkipVerify: true,
				},
			},
		},
		{
			name: "runtime with invalid CA path returns error",
			runtime: &ServerRuntime{
				ClientTLSConfig: &ClientTLSConfig{
					CACertFiles: []string{"/nonexistent/ca.pem"},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			client, err := tc.runtime.GetHTTPClient()

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestServerRuntime_GetHTTPClient_Caching(t *testing.T) {
	runtime := &ServerRuntime{
		ClientTLSConfig: &ClientTLSConfig{
			InsecureSkipVerify: true,
		},
	}

	// Get client twice
	client1, err1 := runtime.GetHTTPClient()
	client2, err2 := runtime.GetHTTPClient()

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Should return the same instance
	assert.Same(t, client1, client2, "GetHTTPClient should return cached client")
}

// generateTestCACert generates a self-signed CA certificate for testing
func generateTestCACert(t *testing.T) []byte {
	t.Helper()

	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM
}

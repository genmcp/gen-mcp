package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// GetHTTPClient returns a configured HTTP client based on the ClientTLSConfig.
// The client is created once and cached for subsequent calls.
// If ClientTLSConfig is nil, it returns a default HTTP client with reasonable timeouts.
func (sr *ServerRuntime) GetHTTPClient() (*http.Client, error) {
	if sr == nil {
		return http.DefaultClient, nil
	}

	sr.httpClientOnce.Do(func() {
		sr.httpClient, sr.httpClientErr = sr.buildHTTPClient()
	})

	return sr.httpClient, sr.httpClientErr
}

// buildHTTPClient creates an HTTP client with custom TLS configuration if specified.
// If no ClientTLSConfig is provided, returns a basic client matching the original behavior.
// When custom TLS is configured, we clone http.DefaultTransport to preserve important
// defaults like ProxyFromEnvironment, TLSHandshakeTimeout, and HTTP/2 support.
func (sr *ServerRuntime) buildHTTPClient() (*http.Client, error) {
	// If no custom TLS config, return a simple client matching original behavior
	// (nil Transport means http.DefaultTransport is used implicitly)
	if sr.ClientTLSConfig == nil {
		return &http.Client{}, nil
	}

	// Build TLS config from user settings
	tlsConfig, err := sr.ClientTLSConfig.BuildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Clone DefaultTransport to preserve proxy settings, timeouts, connection pooling, and HTTP/2.
	// Guard the type assertion in case a host application replaced DefaultTransport.
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http.DefaultTransport is not *http.Transport; cannot apply custom TLS config")
	}
	transport := defaultTransport.Clone()
	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
	}, nil
}

// BuildTLSConfig creates a tls.Config from the ClientTLSConfig settings.
// It loads CA certificates from the specified files and/or directory.
func (c *ClientTLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	// Start with the system cert pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		// On some systems (e.g., Windows), we may not be able to get the system pool
		// In that case, create an empty pool
		rootCAs = x509.NewCertPool()
	}

	// Load CA certificates from individual files
	for _, certFile := range c.CACertFiles {
		if err := appendCertFromFile(rootCAs, certFile); err != nil {
			return nil, fmt.Errorf("failed to load CA cert from %s: %w", certFile, err)
		}
	}

	// Load CA certificates from directory
	if c.CACertDir != "" {
		if err := appendCertsFromDir(rootCAs, c.CACertDir); err != nil {
			return nil, fmt.Errorf("failed to load CA certs from directory %s: %w", c.CACertDir, err)
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: c.InsecureSkipVerify, //nolint:gosec // User explicitly requested insecure mode
	}

	return tlsConfig, nil
}

// appendCertFromFile reads a PEM-encoded certificate file and appends it to the cert pool.
func appendCertFromFile(pool *x509.CertPool, certFile string) error {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	if !pool.AppendCertsFromPEM(certPEM) {
		return fmt.Errorf("failed to parse certificate from %s", certFile)
	}

	return nil
}

// appendCertsFromDir loads all .pem and .crt files from a directory into the cert pool.
// Unlike caCertFiles which fails on any error, directory loading is lenient:
// - Invalid or unreadable cert files are skipped with a warning
// - This matches the behavior of system CA directories which may contain various file types
//
// Note: Warnings are written to stderr because this function is called during server
// initialization (via GetHTTPClient -> buildHTTPClient -> BuildTLSConfig) before the
// structured logger is fully initialized. This follows the same pattern used in
// GetBaseLogger() for initialization-time errors.
func appendCertsFromDir(pool *x509.CertPool, dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".pem" && ext != ".crt" {
			continue
		}

		certPath := filepath.Join(dirPath, name)
		if err := appendCertFromFile(pool, certPath); err != nil {
			// Warn but continue - directory may contain non-cert files or expired certs
			// Using stderr since logger is not available during initialization
			fmt.Fprintf(os.Stderr, "Warning: skipping CA cert %s: %v\n", certPath, err)
			continue
		}
		loadedCount++
	}

	if loadedCount == 0 {
		fmt.Fprintf(os.Stderr, "Warning: no valid CA certificates found in %s\n", dirPath)
	}

	return nil
}

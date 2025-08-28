package oauth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

const (
	ProtectedResourceMetadataEndpoint = "/.well-known/oauth-protected-resource"
)

// Middleware returns a middleware function that checks if the Authorization Header is set and otherwise returns a 401
// with the WWW-Authenticate header containing information about the Protected Resource Endpoint
func Middleware(config *mcpfile.MCPServer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		httpConfig := config.Runtime.StreamableHTTPConfig

		// Only create OAuth handler if auth configured
		if httpConfig.Auth == nil {
			return next // No OAuth config, just pass through
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if auth header is set
			authHeader, ok := r.Header["Authorization"]
			if !ok || len(authHeader) != 1 || !strings.HasPrefix(authHeader[0], "Bearer ") {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}

				// Check for X-Forwarded-Proto header (common in proxy setups)
				if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				}

				fullWellKnownPath := fmt.Sprintf("%s://%s%s", scheme, r.Host, ProtectedResourceMetadataEndpoint)

				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer resource_metadata=%q", fullWellKnownPath))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid_request","error_description":"Missing access token"}`))
				return
			}

			// Auth header is set -> continue request
			next.ServeHTTP(w, r)
		})
	}
}

func ProtectedResourceMetadataHandler(config *mcpfile.MCPServer) http.HandlerFunc {
	httpConfig := config.Runtime.StreamableHTTPConfig

	// Only create OAuth handler if configured
	if httpConfig.Auth == nil {
		return func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		}
	}

	// Convert mcpfile.AuthConfig to oauth.MetadataConfig
	metadataConfig := MetadataConfig{
		ResourceName:           config.Name,
		AuthorizationServers:   httpConfig.Auth.AuthorizationServers,
		ScopesSupported:        httpConfig.Auth.ScopesSupported,
		BearerMethodsSupported: httpConfig.Auth.BearerMethodsSupported,
		JWKSURI:                httpConfig.Auth.JWKSURI,
		ResourceDocumentation:  httpConfig.Auth.ResourceDocumentation,
	}

	return NewProtectedResourceMetadataHandler(httpConfig.BasePath, metadataConfig)
}

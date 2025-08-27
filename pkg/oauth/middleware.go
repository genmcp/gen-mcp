package oauth

import (
	"net/http"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

// Middleware returns a middleware function that adds OAuth 2.0 Protected Resource Metadata endpoint support
func Middleware(config *mcpfile.MCPServer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		httpConfig := config.Runtime.StreamableHTTPConfig

		// Only create OAuth handler if configured
		if httpConfig.Auth == nil {
			return next // No OAuth config, just pass through
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
		oauthHandler := NewProtectedResourceMetadataHandler(metadataConfig)

		// Normalize base path
		basePath := httpConfig.BasePath
		if basePath == "" {
			basePath = mcpfile.BasePathDefault
		}
		wellKnownPath := basePath + "/.well-known/oauth-protected-resource"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a request for the OAuth well-known endpoint
			if r.Method == http.MethodGet && r.URL.Path == wellKnownPath {
				oauthHandler(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

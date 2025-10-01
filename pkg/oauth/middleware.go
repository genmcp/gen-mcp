package oauth

import (
	"fmt"
	"net/http"
	"slices"
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

		// Create token validator from auth config
		validator := NewTokenValidator(TokenValidatorConfig{
			JWKSURI:              httpConfig.Auth.JWKSURI,
			AuthorizationServers: httpConfig.Auth.AuthorizationServers,
		})

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Check if auth header is set
			authHeader, ok := r.Header["Authorization"]
			if !ok || len(authHeader) != 1 || !strings.HasPrefix(authHeader[0], "Bearer ") {
				write401(w, r, `{"error":"invalid_request","error_description":"Missing access token"}`)
				return
			}

			// Extract token from Bearer header
			tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")

			// Validate the token and extract claims
			claims, err := validator.ValidateToken(r.Context(), tokenString)
			if err != nil {
				write401(w, r, fmt.Sprintf(`{"error":"invalid_token","error_description":"Token validation failed: %s"}`, err.Error()))
				return
			}

			// Add claims to request context for downstream handlers
			ctx := AddClaimsToContext(r.Context(), claims)
			r = r.WithContext(ctx)

			// Token is valid -> continue request
			next.ServeHTTP(w, r)
		})
	}
}

func write401(w http.ResponseWriter, r *http.Request, body string) {
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
	http.Error(w, body, http.StatusUnauthorized)
}

func ProtectedResourceMetadataHandler(config *mcpfile.MCPServer) http.HandlerFunc {
	httpConfig := config.Runtime.StreamableHTTPConfig

	// Only create OAuth handler if configured
	if httpConfig.Auth == nil {
		return func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		}
	}

	// Add the scopes, which are defined in the tools
	var scopes []string
	for _, tool := range config.Tools {
		for _, requiredScope := range tool.RequiredScopes {
			if !slices.Contains(scopes, requiredScope) {
				scopes = append(scopes, requiredScope)
			}
		}
	}

	// Convert mcpfile.AuthConfig to oauth.MetadataConfig
	metadataConfig := MetadataConfig{
		ResourceName:         config.Name,
		AuthorizationServers: httpConfig.Auth.AuthorizationServers,
		JWKSURI:              httpConfig.Auth.JWKSURI,
		ScopesSupported:      scopes,
	}

	return NewProtectedResourceMetadataHandler(httpConfig.BasePath, metadataConfig)
}

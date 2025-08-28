package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProtectedResourceMetadata represents the OAuth 2.0 Protected Resource Metadata
// as defined in RFC 9728
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`                           // REQUIRED: the protected resource's resource identifier URL
	ResourceName           string   `json:"resource_name,omitempty"`            // RECOMMENDED: human-readable name
	AuthorizationServers   []string `json:"authorization_servers,omitempty"`    // OPTIONAL: list of authorization server URLs
	ScopesSupported        []string `json:"scopes_supported,omitempty"`         // OPTIONAL: supported OAuth scopes
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"` // OPTIONAL: supported bearer token methods
	JWKSURI                string   `json:"jwks_uri,omitempty"`                 // OPTIONAL: JSON Web Key Set URI
}

// MetadataConfig holds the configuration for OAuth 2.0 Protected Resource Metadata
type MetadataConfig struct {
	ResourceName           string   `json:"resourceName,omitempty"`
	AuthorizationServers   []string `json:"authorizationServers,omitempty"`
	ScopesSupported        []string `json:"scopesSupported,omitempty"`
	BearerMethodsSupported []string `json:"bearerMethodsSupported,omitempty"`
	JWKSURI                string   `json:"jwksUri,omitempty"`
}

// NewProtectedResourceMetadataHandler creates an HTTP handler for the .well-known/oauth-protected-resource endpoint
// The endpoint will be available at {basePath}/.well-known/oauth-protected-resource
func NewProtectedResourceMetadataHandler(basePath string, config MetadataConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Construct the resource URL from the request
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		// Check for X-Forwarded-Proto header (common in proxy setups)
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		}

		resourceURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, basePath)

		// Build the metadata response
		metadata := ProtectedResourceMetadata{
			Resource: resourceURL,
		}

		// Add optional fields if configured
		if config.ResourceName != "" {
			metadata.ResourceName = config.ResourceName
		}

		if len(config.AuthorizationServers) > 0 {
			metadata.AuthorizationServers = config.AuthorizationServers
		}

		if len(config.ScopesSupported) > 0 {
			metadata.ScopesSupported = config.ScopesSupported
		}

		if len(config.BearerMethodsSupported) > 0 {
			metadata.BearerMethodsSupported = config.BearerMethodsSupported
		}

		if config.JWKSURI != "" {
			metadata.JWKSURI = config.JWKSURI
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)

		// Encode and send the response
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode OAuth metadata", http.StatusInternalServerError)
			return
		}
	}
}

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// OIDCDiscoveryDocument represents the OpenID Connect discovery document
type OIDCDiscoveryDocument struct {
	JWKSURI string `json:"jwks_uri"`
}

// TokenValidatorConfig holds configuration for token validation
type TokenValidatorConfig struct {
	JWKSURI              string        // Explicit JWKS URI
	AuthorizationServers []string      // Authorization servers for discovery
	HTTPTimeout          time.Duration // HTTP client timeout (default: 5s)
}

// TokenValidator handles OAuth 2.0 token validation
type TokenValidator struct {
	config TokenValidatorConfig
	client *http.Client
}

// NewTokenValidator creates a new token validator with the given configuration
func NewTokenValidator(config TokenValidatorConfig) *TokenValidator {
	return &TokenValidator{
		config: config,
		client: http.DefaultClient,
	}
}

// ValidateToken validates a JWT token and returns extracted claims
func (tv *TokenValidator) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	// Determine JWKS URI
	jwksURI := tv.config.JWKSURI

	// If JWKS URI is not configured, try to discover it from all authorization servers
	if jwksURI == "" {
		if len(tv.config.AuthorizationServers) == 0 {
			return nil, fmt.Errorf("no JWKS URI configured and no authorization servers provided for discovery")
		}

		var err error
		jwksURI, err = tv.discoverJWKSURIFromAuthServers(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to discover JWKS URI: %w", err)
		}
	}

	// Fetch JWKS
	keySet, err := jwk.Fetch(ctx, jwksURI, jwk.WithHTTPClient(tv.client))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURI, err)
	}

	// Parse and validate the token
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKeySet(keySet))
	if err != nil {
		return nil, fmt.Errorf("failed to parse/validate JWT token: %w", err)
	}

	claims := tv.extractClaims(token)

	if err := tv.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("failed to validate claims: %w", err)
	}

	return claims, nil
}

func (tv *TokenValidator) validateClaims(claims *TokenClaims) error {
	if !slices.Contains(tv.config.AuthorizationServers, claims.Issuer) {
		return fmt.Errorf("invalid token claims: %s is not a valid issuer", claims.Issuer)
	}

	// TODO: add more (e.g. audience?!?)

	return nil
}

// discoverJWKSURIFromAuthServers tries to discover JWKS URI from the authorization servers
func (tv *TokenValidator) discoverJWKSURIFromAuthServers(ctx context.Context) (string, error) {
	var jwksURI string
	var lastErr error

	for _, authServer := range tv.config.AuthorizationServers {
		uri, err := tv.discoverJWKSURI(ctx, authServer)
		if err != nil {
			lastErr = err
			continue // Try next authorization server
		}
		jwksURI = uri
		break // Found a working JWKS URI
	}

	if jwksURI == "" {
		return jwksURI, fmt.Errorf("failed to discover JWKS URI from any authorization server, last error: %w", lastErr)
	}

	return jwksURI, nil
}

// extractClaims extracts claims from a validated JWT token
func (tv *TokenValidator) extractClaims(token jwt.Token) *TokenClaims {
	claims := &TokenClaims{}

	// Standard JWT claims
	if sub, ok := token.Subject(); ok && sub != "" {
		claims.Subject = sub
	}

	if iss, ok := token.Issuer(); ok && iss != "" {
		claims.Issuer = iss
	}

	if aud, ok := token.Audience(); ok && len(aud) > 0 {
		claims.Audience = aud
	}

	if exp, ok := token.Expiration(); ok && !exp.IsZero() {
		claims.Expiry = &exp
	}

	if iat, ok := token.IssuedAt(); ok && !iat.IsZero() {
		claims.IssuedAt = &iat
	}

	if nbf, ok := token.NotBefore(); ok && !nbf.IsZero() {
		claims.NotBefore = &nbf
	}

	// OAuth-specific claims
	var scope string
	if err := token.Get("scope", &scope); err == nil {
		claims.Scope = scope
	}

	var clientID string
	if err := token.Get("client_id", &clientID); err == nil {
		claims.ClientID = clientID
	}

	var username string
	if err := token.Get("username", &username); err == nil {
		claims.Username = username
	}

	var email string
	if err := token.Get("email", &email); err == nil {
		claims.Email = email
	}

	return claims
}

// discoverJWKSURI attempts to discover the JWKS URI using multiple fallback strategies
func (tv *TokenValidator) discoverJWKSURI(ctx context.Context, authServerURL string) (string, error) {
	// Strategy 1: Try common OAuth 2.0 JWKS patterns
	commonPaths := []string{
		"/jwks",
		"/.well-known/jwks.json",
		"/oauth/jwks",
		"/auth/jwks",
	}

	for _, path := range commonPaths {
		jwksURL := strings.TrimSuffix(authServerURL, "/") + path
		if tv.isValidJWKSEndpoint(ctx, jwksURL) {
			return jwksURL, nil
		}
	}

	// Strategy 2: Try OIDC discovery as fallback
	oidcDiscoveryURL := strings.TrimSuffix(authServerURL, "/") + "/.well-known/openid-configuration"
	if jwksURI, err := tv.discoverFromOIDC(ctx, oidcDiscoveryURL); err == nil {
		return jwksURI, nil
	}

	return "", fmt.Errorf("could not discover JWKS URI for authorization server: %s", authServerURL)
}

// isValidJWKSEndpoint checks if the given URL returns a valid JWKS response
func (tv *TokenValidator) isValidJWKSEndpoint(ctx context.Context, jwksURL string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return false
	}

	resp, err := tv.client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Try to parse as JWKS to validate it's a proper JWKS endpoint
	_, err = jwk.ParseReader(resp.Body)
	return err == nil
}

// discoverFromOIDC attempts to discover JWKS URI from OIDC discovery document
func (tv *TokenValidator) discoverFromOIDC(ctx context.Context, discoveryURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := tv.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery endpoint returned status %d", resp.StatusCode)
	}

	var doc OIDCDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("failed to parse OIDC discovery document: %w", err)
	}

	if doc.JWKSURI == "" {
		return "", fmt.Errorf("OIDC discovery document does not contain jwks_uri")
	}

	return doc.JWKSURI, nil
}

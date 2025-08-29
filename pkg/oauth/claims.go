package oauth

import (
	"context"
	"time"
)

// TokenClaims represents extracted token claims
type TokenClaims struct {
	Subject   string     `json:"sub,omitempty"`
	Issuer    string     `json:"iss,omitempty"`
	Audience  []string   `json:"aud,omitempty"`
	Expiry    *time.Time `json:"exp,omitempty"`
	IssuedAt  *time.Time `json:"iat,omitempty"`
	NotBefore *time.Time `json:"nbf,omitempty"`
	Scope     string     `json:"scope,omitempty"`
	ClientID  string     `json:"client_id,omitempty"`
	Username  string     `json:"username,omitempty"`
	Email     string     `json:"email,omitempty"`
}

type claimKey struct{}

// GetClaimsFromContext returns the claims (if set) from the given context
func GetClaimsFromContext(ctx context.Context) *TokenClaims {
	val := ctx.Value(claimKey{})
	if val != nil {
		if claims, ok := val.(*TokenClaims); ok {
			return claims
		}
	}

	return nil
}

// AddClaimsToContext returns a new context with the given claims added, which can be retrieved via GetClaimsFromContext
func AddClaimsToContext(ctx context.Context, claims *TokenClaims) context.Context {
	return context.WithValue(ctx, claimKey{}, claims)
}

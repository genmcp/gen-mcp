package oauth

import (
	"context"
	"time"
)

// TokenClaims represents extracted token claims
type TokenClaims struct {
	Subject   string
	Issuer    string
	Audience  []string
	Expiry    *time.Time
	IssuedAt  *time.Time
	NotBefore *time.Time
	Scope     string
	ClientID  string
	Username  string
	Email     string
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

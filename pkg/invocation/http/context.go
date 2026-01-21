package http

import (
	"context"
	"net/http"
)

type httpClientKey struct{}

// WithHTTPClient stores an HTTP client in the context.
// This client will be used by HTTP invokers for outbound requests.
func WithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	return context.WithValue(ctx, httpClientKey{}, client)
}

// HTTPClientFromContext retrieves the HTTP client from the context.
// If no client is found, it returns http.DefaultClient.
func HTTPClientFromContext(ctx context.Context) *http.Client {
	client := ctx.Value(httpClientKey{})
	if client == nil {
		return http.DefaultClient
	}

	httpClient, ok := client.(*http.Client)
	if !ok {
		return http.DefaultClient
	}

	return httpClient
}

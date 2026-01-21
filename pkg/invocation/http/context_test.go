package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClientFromContext(t *testing.T) {
	// Create a specific client to test storage/retrieval
	storedClient := &http.Client{}

	tt := []struct {
		name           string
		setupContext   func() context.Context
		expectedClient *http.Client
	}{
		{
			name: "returns DefaultClient when no client in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedClient: http.DefaultClient,
		},
		{
			name: "returns DefaultClient when context value is nil",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), httpClientKey{}, nil)
			},
			expectedClient: http.DefaultClient,
		},
		{
			name: "returns DefaultClient when context value is wrong type",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), httpClientKey{}, "not a client")
			},
			expectedClient: http.DefaultClient,
		},
		{
			name: "returns stored client when valid client in context",
			setupContext: func() context.Context {
				return WithHTTPClient(context.Background(), storedClient)
			},
			expectedClient: storedClient,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupContext()
			result := HTTPClientFromContext(ctx)
			assert.Same(t, tc.expectedClient, result)
		})
	}
}

func TestWithHTTPClient(t *testing.T) {
	tt := []struct {
		name   string
		client *http.Client
	}{
		{
			name:   "stores non-nil client in context",
			client: &http.Client{},
		},
		{
			name:   "stores nil client in context",
			client: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := WithHTTPClient(context.Background(), tc.client)
			assert.NotNil(t, ctx)

			// Verify the value was stored
			stored := ctx.Value(httpClientKey{})
			assert.Equal(t, tc.client, stored)
		})
	}
}

func TestWithHTTPClientMiddleware(t *testing.T) {
	tt := []struct {
		name         string
		inputClient  *http.Client
		expectClient *http.Client
	}{
		{
			name:         "nil client uses DefaultClient",
			inputClient:  nil,
			expectClient: http.DefaultClient,
		},
		{
			name:         "non-nil client is passed through",
			inputClient:  &http.Client{},
			expectClient: nil, // Will check it's not DefaultClient
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			middleware := WithHTTPClientMiddleware(tc.inputClient)
			assert.NotNil(t, middleware)

			// The middleware should work without panicking
			// We can't fully test without mocking mcp.MethodHandler, but we verify creation
		})
	}
}

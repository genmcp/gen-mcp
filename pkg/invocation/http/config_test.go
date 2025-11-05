package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpInvocationConfig_Validate(t *testing.T) {
	tt := []struct {
		name        string
		config      *HttpInvocationConfig
		expectError bool
	}{
		{
			name: "valid GET method",
			config: &HttpInvocationConfig{
				URL:    "/api/users",
				Method: "GET",
			},
			expectError: false,
		},
		{
			name: "valid POST method",
			config: &HttpInvocationConfig{
				URL:    "/api/users/{id}",
				Method: "POST",
			},
			expectError: false,
		},
		{
			name: "invalid method",
			config: &HttpInvocationConfig{
				URL:    "/api/users",
				Method: "INVALID",
			},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err, "config validation should return an error")
			} else {
				assert.NoError(t, err, "config validation should not return an error")
			}
		})
	}
}

func TestIsValidHttpMethod(t *testing.T) {
	tt := []struct {
		name     string
		method   string
		expected bool
	}{
		{
			name:     "valid GET method",
			method:   "GET",
			expected: true,
		},
		{
			name:     "valid POST method lowercase",
			method:   "post",
			expected: true,
		},
		{
			name:     "invalid method",
			method:   "INVALID",
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := IsValidHttpMethod(tc.method)
			assert.Equal(t, tc.expected, result, "method validation should match expected result")
		})
	}
}

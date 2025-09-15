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
			name: "valid config with no path parameters",
			config: &HttpInvocationConfig{
				PathTemplate: "/api/users",
				PathIndices:  map[string]int{},
				Method:       "GET",
			},
			expectError: false,
		},
		{
			name: "valid config with path parameters",
			config: &HttpInvocationConfig{
				PathTemplate: "/api/users/%d/posts/%s",
				PathIndices: map[string]int{
					"id":     0,
					"postId": 1,
				},
				Method: "POST",
			},
			expectError: false,
		},
		{
			name: "invalid method",
			config: &HttpInvocationConfig{
				PathTemplate: "/api/users",
				PathIndices:  map[string]int{},
				Method:       "INVALID",
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
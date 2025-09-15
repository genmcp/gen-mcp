package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCliInvocationConfig_Validate(t *testing.T) {
	tt := []struct {
		name        string
		config      *CliInvocationConfig
		expectError bool
	}{
		{
			name: "valid config with no parameters",
			config: &CliInvocationConfig{
				Command:          "echo hello",
				ParameterIndices: map[string]int{},
			},
			expectError: false,
		},
		{
			name: "valid config with parameters",
			config: &CliInvocationConfig{
				Command: "echo %s %s",
				ParameterIndices: map[string]int{
					"param1": 0,
					"param2": 1,
				},
			},
			expectError: false,
		},
		{
			name: "invalid config with mismatched parameters",
			config: &CliInvocationConfig{
				Command: "echo %s",
				ParameterIndices: map[string]int{
					"param1": 0,
					"param2": 1,
				},
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

func TestTemplateVariable_FormatValue(t *testing.T) {
	tt := []struct {
		name     string
		tv       *TemplateVariable
		value    any
		expected string
	}{
		{
			name: "simple template without formatting",
			tv: &TemplateVariable{
				Template:     "--verbose",
				shouldFormat: false,
			},
			value:    true,
			expected: "--verbose",
		},
		{
			name: "template with formatting",
			tv: &TemplateVariable{
				Template:     "--file %s",
				shouldFormat: true,
			},
			value:    "test.txt",
			expected: "--file test.txt",
		},
		{
			name: "omit if false when value is false",
			tv: &TemplateVariable{
				Template:    "--debug",
				OmitIfFalse: true,
			},
			value:    false,
			expected: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.tv.FormatValue(tc.value)
			assert.Equal(t, tc.expected, result, "formatted value should match expected")
		})
	}
}
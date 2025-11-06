package extends

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test struct for testing operations
type testConfig struct {
	Name      string            `json:"name"`
	Env       map[string]string `json:"env"`
	Args      []string          `json:"args"`
	Tags      []string          `json:"tags"`
	Count     int               `json:"count"`
	Timeout   int               `json:"timeout"`
	EnableSSL bool              `json:"enable_ssl"`
}

func TestApplyExtend(t *testing.T) {
	tt := []struct {
		name        string
		base        *testConfig
		ext         *testConfig
		expected    *testConfig
		expectError bool
	}{
		{
			name: "extend strings - concatenate",
			base: &testConfig{
				Name: "hello",
			},
			ext: &testConfig{
				Name: " world",
			},
			expected: &testConfig{
				Name: "hello world",
			},
			expectError: false,
		},
		{
			name: "extend maps - merge keys",
			base: &testConfig{
				Env: map[string]string{
					"FOO": "bar",
				},
			},
			ext: &testConfig{
				Env: map[string]string{
					"BAZ": "qux",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "qux",
				},
			},
			expectError: false,
		},
		{
			name: "extend maps - overwrite existing keys",
			base: &testConfig{
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "original",
				},
			},
			ext: &testConfig{
				Env: map[string]string{
					"BAZ": "updated",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "updated",
				},
			},
			expectError: false,
		},
		{
			name: "extend slices - append items",
			base: &testConfig{
				Args: []string{"--verbose", "--debug"},
			},
			ext: &testConfig{
				Args: []string{"--output", "file.txt"},
			},
			expected: &testConfig{
				Args: []string{"--verbose", "--debug", "--output", "file.txt"},
			},
			expectError: false,
		},
		{
			name: "extend with nil base map - initialize and merge",
			base: &testConfig{
				Env: nil,
			},
			ext: &testConfig{
				Env: map[string]string{
					"NEW": "value",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"NEW": "value",
				},
			},
			expectError: false,
		},
		{
			name: "nil base should error",
			base: nil,
			ext: &testConfig{
				Name: "test",
			},
			expectError: true,
		},
		{
			name: "nil ext should error",
			base: &testConfig{
				Name: "test",
			},
			ext:         nil,
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := applyExtend(tc.base, tc.ext)

			if tc.expectError {
				assert.Error(t, err, "applyExtend should return an error")
			} else {
				assert.NoError(t, err, "applyExtend should not return an error")
				assert.Equal(t, tc.expected.Name, tc.base.Name)
				assert.Equal(t, tc.expected.Env, tc.base.Env)
				assert.Equal(t, tc.expected.Args, tc.base.Args)
			}
		})
	}
}

func TestApplyOverride(t *testing.T) {
	tt := []struct {
		name        string
		base        *testConfig
		override    *testConfig
		expected    *testConfig
		expectError bool
	}{
		{
			name: "override string field",
			base: &testConfig{
				Name: "original",
			},
			override: &testConfig{
				Name: "replaced",
			},
			expected: &testConfig{
				Name: "replaced",
			},
			expectError: false,
		},
		{
			name: "override map completely",
			base: &testConfig{
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "qux",
				},
			},
			override: &testConfig{
				Env: map[string]string{
					"NEW": "value",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"NEW": "value",
				},
			},
			expectError: false,
		},
		{
			name: "override slice completely",
			base: &testConfig{
				Args: []string{"--old", "--args"},
			},
			override: &testConfig{
				Args: []string{"--new", "--different"},
			},
			expected: &testConfig{
				Args: []string{"--new", "--different"},
			},
			expectError: false,
		},
		{
			name: "override int field",
			base: &testConfig{
				Count: 10,
			},
			override: &testConfig{
				Count: 25,
			},
			expected: &testConfig{
				Count: 25,
			},
			expectError: false,
		},
		{
			name: "override with zero values - should skip",
			base: &testConfig{
				Name:  "keep",
				Count: 10,
			},
			override: &testConfig{
				Name: "",
			},
			expected: &testConfig{
				Name:  "keep",
				Count: 10,
			},
			expectError: false,
		},
		{
			name:        "nil base should error",
			base:        nil,
			override:    &testConfig{Name: "test"},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := applyOverride(tc.base, tc.override)

			if tc.expectError {
				assert.Error(t, err, "applyOverride should return an error")
			} else {
				assert.NoError(t, err, "applyOverride should not return an error")
				assert.Equal(t, tc.expected.Name, tc.base.Name)
				assert.Equal(t, tc.expected.Env, tc.base.Env)
				assert.Equal(t, tc.expected.Args, tc.base.Args)
				assert.Equal(t, tc.expected.Count, tc.base.Count)
			}
		})
	}
}

func TestApplyRemove(t *testing.T) {
	tt := []struct {
		name        string
		base        *testConfig
		remove      *testConfig
		expected    *testConfig
		expectError bool
	}{
		{
			name: "remove string - sets to empty",
			base: &testConfig{
				Name: "to be removed",
			},
			remove: &testConfig{
				Name: "anything",
			},
			expected: &testConfig{
				Name: "",
			},
			expectError: false,
		},
		{
			name: "remove map keys",
			base: &testConfig{
				Env: map[string]string{
					"KEEP":   "this",
					"REMOVE": "me",
					"DELETE": "also",
				},
			},
			remove: &testConfig{
				Env: map[string]string{
					"REMOVE": "",
					"DELETE": "",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"KEEP": "this",
				},
			},
			expectError: false,
		},
		{
			name: "remove slice items - matching values",
			base: &testConfig{
				Args: []string{"--keep", "--remove", "--debug", "--remove"},
			},
			remove: &testConfig{
				Args: []string{"--remove", "--debug"},
			},
			expected: &testConfig{
				Args: []string{"--keep"},
			},
			expectError: false,
		},
		{
			name: "remove from empty slice - no change",
			base: &testConfig{
				Args: []string{},
			},
			remove: &testConfig{
				Args: []string{"--something"},
			},
			expected: &testConfig{
				Args: []string{},
			},
			expectError: false,
		},
		{
			name: "remove non-existent map key - no error",
			base: &testConfig{
				Env: map[string]string{
					"KEEP": "this",
				},
			},
			remove: &testConfig{
				Env: map[string]string{
					"NONEXISTENT": "",
				},
			},
			expected: &testConfig{
				Env: map[string]string{
					"KEEP": "this",
				},
			},
			expectError: false,
		},
		{
			name:        "nil base should error",
			base:        nil,
			remove:      &testConfig{Name: "test"},
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := applyRemove(tc.base, tc.remove)

			if tc.expectError {
				assert.Error(t, err, "applyRemove should return an error")
			} else {
				assert.NoError(t, err, "applyRemove should not return an error")
				assert.Equal(t, tc.expected.Name, tc.base.Name)
				assert.Equal(t, tc.expected.Env, tc.base.Env)
				assert.Equal(t, tc.expected.Args, tc.base.Args)
			}
		})
	}
}

func TestValidateOperations(t *testing.T) {
	tt := []struct {
		name        string
		extend      *testConfig
		override    *testConfig
		remove      *testConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "no conflicts - different fields",
			extend: &testConfig{
				Name: "extended",
			},
			override: &testConfig{
				Count: 10,
			},
			remove: &testConfig{
				Timeout: 5,
			},
			expectError: false,
		},
		{
			name: "no conflicts - all empty",
			extend:      &testConfig{},
			override:    &testConfig{},
			remove:      &testConfig{},
			expectError: false,
		},
		{
			name: "conflict - extend and override on same field",
			extend: &testConfig{
				Name: "extended",
			},
			override: &testConfig{
				Name: "overridden",
			},
			remove:      &testConfig{},
			expectError: true,
			errorMsg:    "cannot use multiple operations on field 'Name'",
		},
		{
			name: "conflict - extend and remove on same field",
			extend: &testConfig{
				Args: []string{"--flag"},
			},
			override: &testConfig{},
			remove: &testConfig{
				Args: []string{"--other"},
			},
			expectError: true,
			errorMsg:    "cannot use multiple operations on field 'Args'",
		},
		{
			name:   "conflict - all three operations on same field",
			extend: &testConfig{Env: map[string]string{"A": "b"}},
			override: &testConfig{
				Env: map[string]string{"C": "d"},
			},
			remove: &testConfig{
				Env: map[string]string{"E": "f"},
			},
			expectError: true,
			errorMsg:    "cannot use multiple operations on field 'Env'",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateOperations(tc.extend, tc.override, tc.remove)

			if tc.expectError {
				assert.Error(t, err, "validateOperations should return an error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "validateOperations should not return an error")
			}
		})
	}
}

func TestUnmarshalRemoveConfig(t *testing.T) {
	tt := []struct {
		name        string
		data        string
		expected    *testConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "map field with slice of keys - converts to map",
			data: `{
				"env": ["KEY1", "KEY2", "KEY3"]
			}`,
			expected: &testConfig{
				Env: map[string]string{
					"KEY1": "",
					"KEY2": "",
					"KEY3": "",
				},
			},
			expectError: false,
		},
		{
			name: "map field with actual map - uses as-is",
			data: `{
				"env": {
					"KEY1": "",
					"KEY2": ""
				}
			}`,
			expected: &testConfig{
				Env: map[string]string{
					"KEY1": "",
					"KEY2": "",
				},
			},
			expectError: false,
		},
		{
			name: "slice field - unmarshals normally",
			data: `{
				"args": ["--flag1", "--flag2"]
			}`,
			expected: &testConfig{
				Args: []string{"--flag1", "--flag2"},
			},
			expectError: false,
		},
		{
			name: "string field - unmarshals normally",
			data: `{
				"name": "test-name"
			}`,
			expected: &testConfig{
				Name: "test-name",
			},
			expectError: false,
		},
		{
			name: "empty json object",
			data: `{}`,
			expected: &testConfig{
				Env:  nil,
				Args: nil,
				Name: "",
			},
			expectError: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := &testConfig{}
			err := unmarshalRemoveConfig(json.RawMessage(tc.data), config)

			if tc.expectError {
				assert.Error(t, err, "unmarshalRemoveConfig should return an error")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "error message should contain expected text")
				}
			} else {
				require.NoError(t, err, "unmarshalRemoveConfig should not return an error")
				assert.Equal(t, tc.expected.Name, config.Name)
				assert.Equal(t, tc.expected.Env, config.Env)
				assert.Equal(t, tc.expected.Args, config.Args)
			}
		})
	}
}

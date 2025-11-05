package template

import (
	"os"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplate(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"userId":  {Type: "string"},
			"version": {Type: "string"},
			"count":   {Type: "integer"},
		},
	}

	tt := []struct {
		name             string
		template         string
		opts             TemplateParserOptions
		expectErr        bool
		expectedTemplate string
		expectedVarCount int
		expectedIndices  map[string][]int
		expectedVarNames []string
	}{
		{
			name:     "simple template with single parameter",
			template: "https://api.com/users/{userId}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://api.com/users/%s",
			expectedVarCount: 1,
			expectedIndices: map[string][]int{
				"userId": {0},
			},
			expectedVarNames: []string{"userId"},
		},
		{
			name:     "template with multiple parameters",
			template: "https://api.com/{version}/users/{userId}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://api.com/%s/users/%s",
			expectedVarCount: 2,
			expectedIndices: map[string][]int{
				"version": {0},
				"userId":  {1},
			},
			expectedVarNames: []string{"version", "userId"},
		},
		{
			name:     "template with duplicate parameter references",
			template: "api/{version}/users/{userId}/v{version}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "api/%s/users/%s/v%s",
			expectedVarCount: 3,
			expectedIndices: map[string][]int{
				"version": {0, 2},
				"userId":  {1},
			},
			expectedVarNames: []string{"version", "userId", "version"},
		},
		{
			name:     "template with environment variable ${VAR}",
			template: "https://${API_HOST}/users/{userId}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://%s/users/%s",
			expectedVarCount: 2,
			expectedIndices: map[string][]int{
				"API_HOST": {0},
				"userId":   {1},
			},
			expectedVarNames: []string{"API_HOST", "userId"},
		},
		{
			name:     "template with environment variable {env.VAR}",
			template: "https://{env.API_HOST}/users/{userId}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://%s/users/%s",
			expectedVarCount: 2,
			expectedIndices: map[string][]int{
				"API_HOST": {0},
				"userId":   {1},
			},
			expectedVarNames: []string{"API_HOST", "userId"},
		},
		{
			name:     "template with integer parameter",
			template: "https://api.com/items/{count}",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://api.com/items/%d",
			expectedVarCount: 1,
			expectedIndices: map[string][]int{
				"count": {0},
			},
			expectedVarNames: []string{"count"},
		},
		{
			name:     "template with no variables",
			template: "https://api.com/static/path",
			opts: TemplateParserOptions{
				InputSchema: schema,
			},
			expectedTemplate: "https://api.com/static/path",
			expectedVarCount: 0,
			expectedIndices:  map[string][]int{},
			expectedVarNames: []string{},
		},
		{
			name:      "template with unterminated variable",
			template:  "https://api.com/{userId",
			opts:      TemplateParserOptions{InputSchema: schema},
			expectErr: true,
		},
		{
			name:      "template with unmatched closing bracket",
			template:  "https://api.com/userId}",
			opts:      TemplateParserOptions{InputSchema: schema},
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := ParseTemplate(tc.template, tc.opts)

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTemplate, pt.Template)
			assert.Equal(t, tc.expectedVarCount, len(pt.Variables))
			assert.Equal(t, tc.expectedIndices, pt.VariableIndices)

			varNames := make([]string, len(pt.Variables))
			for i, v := range pt.Variables {
				varNames[i] = v.Name
			}
			assert.Equal(t, tc.expectedVarNames, varNames)
		})
	}
}

func TestTemplateBuilder(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"userId":  {Type: "string"},
			"version": {Type: "string"},
			"count":   {Type: "integer"},
			"flag":    {Type: "boolean"},
		},
	}

	tt := []struct {
		name           string
		template       string
		setFields      map[string]any
		expectedResult string
		expectErr      bool
	}{
		{
			name:     "simple template with single value",
			template: "https://api.com/users/{userId}",
			setFields: map[string]any{
				"userId": "123",
			},
			expectedResult: "https://api.com/users/123",
		},
		{
			name:     "template with multiple values",
			template: "https://api.com/{version}/users/{userId}",
			setFields: map[string]any{
				"version": "v2",
				"userId":  "123",
			},
			expectedResult: "https://api.com/v2/users/123",
		},
		{
			name:     "template with duplicate parameter references",
			template: "api/{version}/users/{userId}/{version}",
			setFields: map[string]any{
				"version": "v2",
				"userId":  "123",
			},
			expectedResult: "api/v2/users/123/v2",
		},
		{
			name:     "template with integer parameter",
			template: "https://api.com/items/{count}",
			setFields: map[string]any{
				"count": 42,
			},
			expectedResult: "https://api.com/items/42",
		},
		{
			name:     "template with unused parameter in SetField",
			template: "https://api.com/users/{userId}",
			setFields: map[string]any{
				"userId":  "123",
				"version": "v2",
			},
			expectedResult: "https://api.com/users/123",
		},
		{
			name:     "template with missing required parameter",
			template: "https://api.com/users/{userId}",
			setFields: map[string]any{
				"version": "v2",
			},
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := ParseTemplate(tc.template, TemplateParserOptions{
				InputSchema: schema,
			})
			require.NoError(t, err)

			builder, _ := NewTemplateBuilder(pt, false)

			for path, value := range tc.setFields {
				builder.SetField(path, value)
			}

			result, err := builder.GetResult()

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestEnvVarFormatter(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	os.Setenv("TEST_HOST", "api.example.com")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_HOST")

	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"userId": {Type: "string"},
		},
	}

	tt := []struct {
		name           string
		template       string
		setFields      map[string]any
		expectedResult string
		expectErr      bool
	}{
		{
			name:     "template with ${VAR} syntax",
			template: "https://${TEST_HOST}/users/{userId}",
			setFields: map[string]any{
				"userId": "123",
			},
			expectedResult: "https://api.example.com/users/123",
		},
		{
			name:     "template with {env.VAR} syntax",
			template: "https://{env.TEST_HOST}/users/{userId}",
			setFields: map[string]any{
				"userId": "123",
			},
			expectedResult: "https://api.example.com/users/123",
		},
		{
			name:           "template with multiple env vars",
			template:       "${TEST_API_KEY}@{env.TEST_HOST}",
			setFields:      map[string]any{},
			expectedResult: "secret123@api.example.com",
		},
		{
			name:      "template with missing env var",
			template:  "https://${MISSING_VAR}/users/{userId}",
			setFields: map[string]any{"userId": "123"},
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := ParseTemplate(tc.template, TemplateParserOptions{
				InputSchema: schema,
			})
			require.NoError(t, err)

			builder, _ := NewTemplateBuilder(pt, false)

			for path, value := range tc.setFields {
				builder.SetField(path, value)
			}

			result, err := builder.GetResult()

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestConditionalFormatting(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"flag": {Type: "boolean"},
			"name": {Type: "string"},
		},
	}

	tt := []struct {
		name           string
		templateStr    string
		omitIfFalse    bool
		setFields      map[string]any
		expectedResult string
	}{
		{
			name:        "omitIfFalse with false value",
			templateStr: "--flag",
			omitIfFalse: true,
			setFields: map[string]any{
				"flag": false,
			},
			expectedResult: "",
		},
		{
			name:        "omitIfFalse with true value",
			templateStr: "--flag",
			omitIfFalse: true,
			setFields: map[string]any{
				"flag": true,
			},
			expectedResult: "--flag",
		},
		{
			name:        "omitIfFalse=false with false value",
			templateStr: "--flag",
			omitIfFalse: false,
			setFields: map[string]any{
				"flag": false,
			},
			expectedResult: "--flag",
		},
		{
			name:        "omitIfFalse with non-boolean value",
			templateStr: "--name={name}",
			omitIfFalse: true,
			setFields: map[string]any{
				"name": "test",
			},
			expectedResult: "--name=test",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			formatter, err := NewTemplateFormatter(tc.templateStr, schema, tc.omitIfFalse)
			require.NoError(t, err)

			for path, value := range tc.setFields {
				formatter.SetField(path, value)
			}

			result, err := formatter.GetResult()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestNestedTemplateFormatters(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"user":  {Type: "string"},
			"token": {Type: "string"},
			"url":   {Type: "string"},
		},
	}

	authFormatter, err := NewTemplateFormatter("--auth={user}:{token}", schema, false)
	require.NoError(t, err)

	pt, err := ParseTemplate("curl {url} {auth}", TemplateParserOptions{
		InputSchema: schema,
		Formatters: map[string]VariableFormatter{
			"auth": authFormatter,
		},
	})
	require.NoError(t, err)

	builder, _ := NewTemplateBuilder(pt, false)

	builder.SetField("url", "https://api.com")
	builder.SetField("user", "alice")
	builder.SetField("token", "secret123")

	result, err := builder.GetResult()
	require.NoError(t, err)
	assert.Equal(t, "curl https://api.com --auth=alice:secret123", result)
}

func TestCustomFormatters(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}

	customFormatter := &testFormatter{
		name:         "name",
		formatString: "%s",
		transform: func(v any) any {
			if s, ok := v.(string); ok {
				return "custom-" + s
			}
			return v
		},
	}

	pt, err := ParseTemplate("User: {name}, Age: {age}", TemplateParserOptions{
		InputSchema: schema,
		Formatters: map[string]VariableFormatter{
			"name": customFormatter,
		},
	})
	require.NoError(t, err)

	builder, _ := NewTemplateBuilder(pt, false)
	builder.SetField("name", "alice")
	builder.SetField("age", 30)

	result, err := builder.GetResult()
	require.NoError(t, err)
	assert.Equal(t, "User: custom-alice, Age: 30", result)
}

type testFormatter struct {
	name         string
	formatString string
	value        any
	hasValue     bool
	transform    func(any) any
}

func (tf *testFormatter) SetField(path string, value any) {
	if path == tf.name {
		tf.value = value
		tf.hasValue = true
	}
}

func (tf *testFormatter) GetResult() (any, error) {
	if !tf.hasValue {
		return nil, assert.AnError
	}
	if tf.transform != nil {
		return tf.transform(tf.value), nil
	}
	return tf.value, nil
}

func (tf *testFormatter) FormatString() string {
	return tf.formatString
}

func (tf *testFormatter) VariableNames() []string {
	return []string{tf.name}
}

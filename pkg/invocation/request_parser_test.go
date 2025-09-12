package invocation

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestDynamicJsonParser(t *testing.T) {
	tt := []struct {
		name                   string
		schema                 *jsonschema.Schema
		in                     map[string]any
		expectErr              bool
		builderPaths           map[string][]string
		builderCalls           []string
		expectedBuilderResults map[string]map[string]any
	}{
		{
			name: "simple schema, no builders",
			schema: &jsonschema.Schema{
				Type: JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"city": {
						Type: JsonSchemaTypeString,
					},
					"country": {
						Type: JsonSchemaTypeString,
					},
				},
			},
			in: map[string]any{
				"city":    "Toronto",
				"country": "Canada",
			},
		},
		{
			name: "simple schema, with builders",
			schema: &jsonschema.Schema{
				Type: JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"city": {
						Type: JsonSchemaTypeString,
					},
					"country": {
						Type: JsonSchemaTypeString,
					},
				},
			},
			in: map[string]any{
				"city":    "Toronto",
				"country": "Canada",
			},
			builderPaths: map[string][]string{
				"builder1": {"city"},
				"builder2": {"city", "country"},
			},
			builderCalls: []string{"city", "country"},
			expectedBuilderResults: map[string]map[string]any{
				"builder1": {
					"city": "Toronto",
				},
				"builder2": {
					"city":    "Toronto",
					"country": "Canada",
				},
			},
		},
		{
			name: "schema with all types",
			schema: &jsonschema.Schema{
				Type: JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"number": {
						Type: JsonSchemaTypeNumber,
					},
					"integer": {
						Type: JsonSchemaTypeInteger,
					},
					"boolean": {
						Type: JsonSchemaTypeBoolean,
					},
					"string": {
						Type: JsonSchemaTypeString,
					},
					"array": {
						Type:  JsonSchemaTypeArray,
						Items: &jsonschema.Schema{Type: JsonSchemaTypeInteger},
					},
					"object": {
						Type: JsonSchemaTypeObject,
						Properties: map[string]*jsonschema.Schema{
							"boolean": {
								Type: JsonSchemaTypeBoolean,
							},
						},
					},
				},
			},
			in: map[string]any{
				"number":  4.2,
				"integer": 4,
				"boolean": true,
				"string":  "hello, world",
				"array":   []any{1, 2, 3, 4, 5},
				"object": map[string]any{
					"boolean": false,
				},
			},
			builderCalls: []string{"number", "integer", "boolean", "string", "array[0]", "array[1]", "array[2]", "array[3]", "array[4]", "object.boolean"},
			builderPaths: map[string][]string{
				"nested":    {"boolean", "object.boolean"},
				"arrayItem": {"array[0]"},
			},
			expectedBuilderResults: map[string]map[string]any{
				"nested": {
					"boolean":        true,
					"object.boolean": false,
				},
				"arrayItem": {
					"array[0]": 1,
				},
			},
		},
		{
			name: "missing required field, still parses whole object",
			schema: &jsonschema.Schema{
				Type: JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"city": {
						Type: JsonSchemaTypeString,
					},
					"country": {
						Type: JsonSchemaTypeString,
					},
				},
				Required: []string{"city", "country"},
			},
			in: map[string]any{
				"city": "Toronto",
			},
			expectErr:    true,
			builderCalls: []string{"city"},
			builderPaths: map[string][]string{
				"builder1": {"city"},
			},
			expectedBuilderResults: map[string]map[string]any{
				"builder1": {
					"city": "Toronto",
				},
			},
		},
		{
			name: "parse additional parameters",
			schema: &jsonschema.Schema{
				Type: JsonSchemaTypeObject,
				Properties: map[string]*jsonschema.Schema{
					"city": {
						Type: JsonSchemaTypeString,
					},
					"country": {
						Type: JsonSchemaTypeString,
					},
				},
				AdditionalProperties: &jsonschema.Schema{},
			},
			in: map[string]any{
				"city":    "Toronto",
				"country": "Canada",
				"nested": map[string]any{
					"something": "else",
					"float":     3.2,
				},
			},
			builderCalls: []string{"city", "country", "nested"},
			builderPaths: map[string][]string{
				"definedProperties":    {"city", "country"},
				"additionalProperties": {"nested"},
			},
			expectedBuilderResults: map[string]map[string]any{
				"definedProperties": {
					"city":    "Toronto",
					"country": "Canada",
				},
				"additionalProperties": {
					"nested": map[string]any{
						"something": "else",
						"float":     3.2,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			builders := make(map[string]Builder, len(tc.builderPaths))
			for builderName, builderPaths := range tc.builderPaths {
				builders[builderName] = &testBuilder{interestPaths: builderPaths}
			}
			dj := &DynamicJson{
				Builders: slices.Collect(maps.Values(builders)),
			}

			inJson, err := json.Marshal(tc.in)
			assert.NoError(t, err, "marshalling input map to JSON should not fail")

			out, err := dj.parseObject(inJson, tc.schema, "")
			if tc.expectErr {
				assert.Error(t, err, "parsing the json object should fail")
			} else {
				assert.NoError(t, err, "parsing the json object should not fail")
			}

			assert.Equal(t, tc.in, out, "parsing the map should get the same values")

			for buidlerName, expectedBuilderResult := range tc.expectedBuilderResults {
				actualResult, _ := builders[buidlerName].GetResult()
				assert.Equal(t, expectedBuilderResult, actualResult)
				assert.ElementsMatch(t, tc.builderCalls, builders[buidlerName].(*testBuilder).calls)
			}
		})
	}
}

type testBuilder struct {
	interestPaths []string
	results       map[string]any
	calls         []string
}

func (tb *testBuilder) SetField(path string, value any) {
	if tb.calls == nil {
		tb.calls = []string{}
	}
	tb.calls = append(tb.calls, path)
	if tb.results == nil {
		tb.results = make(map[string]any)
	}
	if slices.Contains(tb.interestPaths, path) {
		tb.results[path] = value
	}
}

func (tb *testBuilder) GetResult() (any, error) {
	return tb.results, nil
}

var _ Builder = &testBuilder{}

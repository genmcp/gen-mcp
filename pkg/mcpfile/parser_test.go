package mcpfile

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var exampleWeatherUrl, _ = url.Parse("http://example.com/weather")

func TestParseMcpFile(t *testing.T) {
	tt := map[string]struct {
		testFileName string
		expected     *MCPFile
		wantErr      bool
	}{
		"no servers": {
			testFileName: "no-servers.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
			},
		},
		"one server, no tools": {
			testFileName: "one-server-no-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
					},
				},
			},
		},
		"one server, with tools": {
			testFileName: "one-server-tools.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Tools: []*Tool{
							{
								Name: "get_weather",
								Title: "Weather Information Provider",
								Description: "Get current weather information for a location",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"location": {
											Type: JsonSchemaTypeString,
											Description: "City name or zip code",
										},
									},
									Required: []string{"location"},
								},
								URL: *exampleWeatherUrl,
							},
						},
					},
				},
			},
		},
	}

	for testName, testCase := range tt {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			mcpFile, err := ParseMCPFile(fmt.Sprintf("./testdata/%s", testCase.testFileName))
			if testCase.wantErr {
				assert.Error(t, err, "parsing mcp file should cause an error")
			} else {
				assert.NoError(t, err, "parsing mcp file should succeed")
			}

			assert.Equal(t, testCase.expected, mcpFile)
		})

	}
}

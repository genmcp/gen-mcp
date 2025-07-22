package mcpfile

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"companyName": {
											Type:        JsonSchemaTypeString,
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								Invocation: &HttpInvocation{
									URL:    "http://localhost:5000",
									Method: http.MethodPost,
								},
							},
						},
					},
				},
			},
		},
		"one server, with tools and http params": {
			testFileName: "one-server-tools-http-params.yaml",
			expected: &MCPFile{
				FileVersion: MCPFileVersion,
				Servers: []*MCPServer{
					{
						Name:    "test-server",
						Version: "1.0.0",
						Tools: []*Tool{
							{
								Name:        "get_user_by_company",
								Title:       "Users Provider",
								Description: "Get list of users from a given company",
								InputSchema: &JsonSchema{
									Type: JsonSchemaTypeObject,
									Properties: map[string]*JsonSchema{
										"companyName": {
											Type:        JsonSchemaTypeString,
											Description: "Name of the company",
										},
									},
									Required: []string{"companyName"},
								},
								Invocation: &HttpInvocation{
									URL:            "http://localhost:5000/{}/users",
									Method:         http.MethodGet,
									pathParameters: []string{"companyName"},
								},
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

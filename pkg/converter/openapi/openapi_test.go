package openapi

import (
	"fmt"
	"os"
	"testing"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConvertFromOpenApiSpec(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/petstorev3.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")
	assert.Error(t, err, "creating the GenMCP config files from the openapi model should have errors on endpoints genmcp does not support")
	assert.NotNil(t, convertedFiles)
	assert.NotNil(t, convertedFiles.ToolDefinitions)
	assert.NotNil(t, convertedFiles.ServerConfig)

	toolDefYaml, err := yaml.Marshal(convertedFiles.ToolDefinitions)
	assert.NoError(t, err, "marshalling tool definitions to yaml should not cause an error")

	serverConfigYaml, err := yaml.Marshal(convertedFiles.ServerConfig)
	assert.NoError(t, err, "marshalling server config to yaml should not cause an error")

	fmt.Printf("Tool Definitions:\n%s\n\nServer Config:\n%s", toolDefYaml, serverConfigYaml)
}

func TestDefaultPort8080InOpenAPIV3Conversion(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/petstorev3.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")
	assert.Error(t, err, "creating the GenMCP config files from the openapi model should have errors on endpoints genmcp does not support")
	assert.NotNil(t, convertedFiles)
	assert.NotNil(t, convertedFiles.ServerConfig)

	assert.Equal(t, serverconfig.DefaultPort, convertedFiles.ServerConfig.Runtime.StreamableHTTPConfig.Port, "OpenAPI v3 conversion should default to port 8080")
}

func TestInvalidToolsAreSkippedButValidOnesIncluded(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v3_with_invalid_tools.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")

	// With summary fallback, all tools should now be valid (summary fills in missing description)
	assert.NoError(t, err, "conversion should not report errors when summary fallback is available")
	assert.NotNil(t, convertedFiles, "GenMCP config files should still be generated")
	assert.NotNil(t, convertedFiles.ToolDefinitions, "tool definitions should be generated")

	assert.NotNil(t, convertedFiles.ToolDefinitions.Tools, "tool definitions should have tools")

	// All 3 tools should now be valid (the previously-invalid one gets summary as description)
	assert.Len(t, convertedFiles.ToolDefinitions.Tools, 3, "should have exactly 3 valid tools")

	// Check that all tools are present
	toolNames := make([]string, len(convertedFiles.ToolDefinitions.Tools))
	for i, tool := range convertedFiles.ToolDefinitions.Tools {
		toolNames[i] = tool.Name
		assert.NotEmpty(t, tool.Description, "all included tools should have descriptions")
	}

	assert.Contains(t, toolNames, "get_valid-endpoint", "should include the valid GET endpoint")
	assert.Contains(t, toolNames, "get_invalid-endpoint", "should include the endpoint that fell back to summary")
	assert.Contains(t, toolNames, "post_another-valid-endpoint", "should include the valid POST endpoint")
}

func TestAllToolsInvalidStillReturnsEmptyMcpFile(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v3_all_invalid_tools.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")

	// Should get an error about all invalid tools
	assert.Error(t, err, "conversion should report errors about all invalid tools")
	assert.NotNil(t, convertedFiles, "GenMCP config files should still be generated")
	assert.NotNil(t, convertedFiles.ToolDefinitions, "tool definitions should be generated")

	assert.Empty(t, convertedFiles.ToolDefinitions.Tools, "tool definitions should have no tools when all are invalid")

	// Check that error mentions both skipped tools
	assert.Contains(t, err.Error(), "get_no-description-1", "error should mention first skipped tool")
	assert.Contains(t, err.Error(), "post_no-description-2", "error should mention second skipped tool")
}

func TestSummaryFallbackWhenDescriptionAbsent(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v3_summary_only.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")
	assert.NoError(t, err, "conversion should not produce errors")
	assert.NotNil(t, convertedFiles, "GenMCP config files should be generated")
	assert.NotNil(t, convertedFiles.ToolDefinitions, "tool definitions should be generated")

	// All 3 tools should be generated (none skipped)
	assert.Len(t, convertedFiles.ToolDefinitions.Tools, 3, "should have exactly 3 valid tools")

	// Build a map of tool name -> tool for easier assertions
	toolsByName := make(map[string]*definitions.Tool)
	for _, tool := range convertedFiles.ToolDefinitions.Tools {
		toolsByName[tool.Name] = tool
	}

	// Summary-only operations should use summary as description
	listUsers := toolsByName["get_users"]
	assert.NotNil(t, listUsers, "should have get_users tool")
	assert.Equal(t, "List all users", listUsers.Description, "summary-only tool should use summary as description")
	assert.Equal(t, "List all users", listUsers.Title, "title should still be the summary")

	getUser := toolsByName["get_users-userId"]
	assert.NotNil(t, getUser, "should have get_users-userId tool")
	assert.Equal(t, "Get a user by ID", getUser.Description, "summary-only tool should use summary as description")

	// Operation with both summary and description should use description
	health := toolsByName["get_health"]
	assert.NotNil(t, health, "should have get_health tool")
	assert.Equal(t, "Returns the health status of the service", health.Description, "tool with both fields should use description")
	assert.Equal(t, "Health check endpoint", health.Title, "title should be the summary")
}

func TestOpenAPIV2BodyParameterHandling(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v2_body_param.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")
	assert.NoError(t, err, "conversion should not produce errors")
	assert.NotNil(t, convertedFiles, "GenMCP config files should be generated")
	assert.NotNil(t, convertedFiles.ToolDefinitions, "tool definitions should be generated")

	assert.Len(t, convertedFiles.ToolDefinitions.Tools, 1, "should have exactly 1 tool")

	tool := convertedFiles.ToolDefinitions.Tools[0]
	assert.Equal(t, "post_features-vote", tool.Name)
	assert.Equal(t, "Vote for feature", tool.Title)

	// The body parameter name should be ignored, and properties should be merged directly
	assert.NotNil(t, tool.InputSchema.Properties, "input schema should have properties")
	assert.NotContains(t, tool.InputSchema.Properties, "body", "should not have 'body' wrapper property")
	assert.Contains(t, tool.InputSchema.Properties, "id", "should have 'id' property from body schema")

	// Verify the id property is correctly typed
	idProp := tool.InputSchema.Properties["id"]
	assert.Equal(t, "integer", idProp.Type)
	assert.Equal(t, "ID of the feature to vote for", idProp.Description)
}

func TestOpenAPIV2BodyParameterWithPathParameters(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v2_body_and_path.json")

	convertedFiles, err := DocumentToMcpFile(docBytes, "")
	assert.NoError(t, err, "conversion should not produce errors")
	assert.NotNil(t, convertedFiles, "GenMCP config files should be generated")
	assert.NotNil(t, convertedFiles.ToolDefinitions, "tool definitions should be generated")

	assert.Len(t, convertedFiles.ToolDefinitions.Tools, 1, "should have exactly 1 tool")

	tool := convertedFiles.ToolDefinitions.Tools[0]
	assert.Equal(t, "post_users-userId-posts", tool.Name)

	// Should have both path parameter and body schema properties
	assert.Contains(t, tool.InputSchema.Properties, "userId", "should have path parameter")
	assert.Contains(t, tool.InputSchema.Properties, "title", "should have body property 'title'")
	assert.Contains(t, tool.InputSchema.Properties, "content", "should have body property 'content'")
	assert.Contains(t, tool.InputSchema.Properties, "tags", "should have body property 'tags'")

	// Verify required fields from both path parameter and body schema
	assert.Contains(t, tool.InputSchema.Required, "userId", "userId should be required (path param)")
	assert.Contains(t, tool.InputSchema.Required, "title", "title should be required (from body schema)")
	assert.Contains(t, tool.InputSchema.Required, "content", "content should be required (from body schema)")
}

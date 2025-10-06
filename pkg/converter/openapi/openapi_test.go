package openapi

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConvertFromOpenApiSpec(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/petstorev3.json")

	mcpfile, err := DocumentToMcpFile(docBytes, "")
	assert.Error(t, err, "creating the mcp file from the openapi model should have errors on endpoints genmcp does not support")
	assert.NotNil(t, mcpfile)

	mcpYaml, err := yaml.Marshal(mcpfile)
	assert.NoError(t, err, "marshalling mcpfile to yaml should not cause an error")

	fmt.Printf("%s", mcpYaml)
}

func TestDefaultPort8080InOpenAPIV3Conversion(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/petstorev3.json")

	mcpfile, err := DocumentToMcpFile(docBytes, "")
	assert.Error(t, err, "creating the mcp file from the openapi model should have errors on endpoints genmcp does not support")
	assert.NotNil(t, mcpfile)

	assert.Equal(t, 8080, mcpfile.Runtime.StreamableHTTPConfig.Port, "OpenAPI v3 conversion should default to port 8080")
}

func TestInvalidToolsAreSkippedButValidOnesIncluded(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_with_invalid_tools.json")

	mcpfile, err := DocumentToMcpFile(docBytes, "")

	// We should get an error about the invalid tool but still get a valid MCP file
	assert.Error(t, err, "conversion should report errors about invalid tools")
	assert.NotNil(t, mcpfile, "MCP file should still be generated")

	assert.NotNil(t, mcpfile.Tools, "server should have tools")

	// Should have exactly 2 valid tools (the ones with descriptions)
	assert.Len(t, mcpfile.Tools, 2, "should have exactly 2 valid tools")

	// Check that the valid tools are present
	toolNames := make([]string, len(mcpfile.Tools))
	for i, tool := range mcpfile.Tools {
		toolNames[i] = tool.Name
		assert.NotEmpty(t, tool.Description, "all included tools should have descriptions")
	}

	assert.Contains(t, toolNames, "get_valid-endpoint", "should include the valid GET endpoint")
	assert.Contains(t, toolNames, "post_another-valid-endpoint", "should include the valid POST endpoint")

	// Check that the error message contains information about the skipped tool
	assert.Contains(t, err.Error(), "get_invalid-endpoint", "error should mention the skipped tool")
	assert.Contains(t, err.Error(), "description is required", "error should mention why the tool was skipped")
}

func TestAllToolsInvalidStillReturnsEmptyMcpFile(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_all_invalid_tools.json")

	mcpfile, err := DocumentToMcpFile(docBytes, "")

	// Should get an error about all invalid tools
	assert.Error(t, err, "conversion should report errors about all invalid tools")
	assert.NotNil(t, mcpfile, "MCP file should still be generated")

	assert.Empty(t, mcpfile.Tools, "server should have no tools when all are invalid")

	// Check that error mentions both skipped tools
	assert.Contains(t, err.Error(), "get_no-description-1", "error should mention first skipped tool")
	assert.Contains(t, err.Error(), "post_no-description-2", "error should mention second skipped tool")
}

func TestOpenAPIV2BodyParameterHandling(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/openapi_v2_body_param.json")

	mcpfile, err := DocumentToMcpFile(docBytes, "")
	assert.NoError(t, err, "conversion should not produce errors")
	assert.NotNil(t, mcpfile, "MCP file should be generated")

	assert.Len(t, mcpfile.Tools, 1, "should have exactly 1 tool")

	tool := mcpfile.Tools[0]
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

	mcpfile, err := DocumentToMcpFile(docBytes, "")
	assert.NoError(t, err, "conversion should not produce errors")
	assert.NotNil(t, mcpfile, "MCP file should be generated")

	assert.Len(t, mcpfile.Tools, 1, "should have exactly 1 tool")

	tool := mcpfile.Tools[0]
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

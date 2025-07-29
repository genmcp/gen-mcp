package openapi

import (
	"fmt"
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestConvertFromOpenApiSpec(t *testing.T) {
	docBytes, _ := os.ReadFile("testdata/petstorev3.json")

	doc, err := ParseDocument(docBytes)
	assert.NoError(t, err, "parsing the openapi document should succeed")

	mcpfile, err := McpFileFromOpenApiModel(&doc.Model)
	assert.NoError(t, err, "creating the mcp file from the openapi model should succeed")

	mcpYaml, err := yaml.Marshal(mcpfile)
	assert.NoError(t, err, "marshalling mcpfile to yaml should not cause an error")

	fmt.Printf("%s", mcpYaml)
}

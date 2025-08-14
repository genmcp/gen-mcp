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

	mcpfile, err := DocumentToMcpFile(docBytes, "")
	assert.Error(t, err, "creating the mcp file from the openapi model should have errors on endpoints genmcp does not support")
	assert.NotNil(t, mcpfile)

	mcpYaml, err := yaml.Marshal(mcpfile)
	assert.NoError(t, err, "marshalling mcpfile to yaml should not cause an error")

	fmt.Printf("%s", mcpYaml)
}

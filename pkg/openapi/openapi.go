package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Cali0707/AutoMCP/pkg/mcpfile"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func ParseDocument(document []byte) (*libopenapi.DocumentModel[v3high.Document], error) {
	doc, err := libopenapi.NewDocument(document)
	if err != nil {
		return nil, fmt.Errorf("failed to create openapi document: %w", err)
	}

	docModel, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI V3 model: %w", err)
	}

	return docModel, nil
}

func McpFileFromOpenApiModel(model *v3high.Document) (*mcpfile.MCPFile, error) {
	// 1. Set top level MCP file info
	// 2. Create a server in the MCP file, default to streamablehttp transport w. port 7007
	// 3 for each (path, operation) in the document, add one tool to the server w. http invoke
	res := &mcpfile.MCPFile{
		FileVersion: mcpfile.MCPFileVersion,
	}
	server := &mcpfile.MCPServer{
		Runtime: &mcpfile.ServerRuntime{
			TransportProtocol: mcpfile.TransportProtocolStreamableHttp,
			StreamableHTTPConfig: &mcpfile.StreamableHTTPConfig{
				Port: 7007,
			},
		},
		Tools: []*mcpfile.Tool{},
	}

	baseUrl := ""
	if len(model.Servers) > 0 {
		baseUrl = model.Servers[0].URL

	}

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			tool := &mcpfile.Tool{
				InputSchema: &mcpfile.JsonSchema{
					Type:       mcpfile.JsonSchemaTypeObject,
					Properties: make(map[string]*mcpfile.JsonSchema),
					Required:   []string{},
				},
				Invocation: &mcpfile.HttpInvocation{
					URL:    fmt.Sprintf("%s%s", baseUrl, pathName),
					Method: strings.ToUpper(operationMethod),
				},
			}

			for _, param := range operation.Parameters {
				if param == nil {
					continue
				}

				tool.InputSchema.Properties[param.Name] = convertSchema(param.Schema)

				if (param.Required != nil && *param.Required) || strings.ToLower(param.In) == "path" {
					tool.InputSchema.Required = append(tool.InputSchema.Required, param.Name)
				}
			}

			// hack to make sure everything is parsed correctly
			toolJson, err := json.Marshal(tool)
			if err != nil {
				return nil, fmt.Errorf("converted tool failed to serialize to mcpfile spec: %w", err)
			}

			t := &mcpfile.Tool{}
			err = json.Unmarshal(toolJson, t)
			if err != nil {
				return nil, fmt.Errorf("converted tool failed to deserialize with mcpfile spec: %w", err)
			}

			server.Tools = append(server.Tools, t)
		}
	}

	err := server.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid converted server: %w", err)
	}

	res.Servers = []*mcpfile.MCPServer{server}

	return res, nil
}

func convertSchema(proxy *highbase.SchemaProxy) *mcpfile.JsonSchema {
	schema := proxy.Schema()
	s := &mcpfile.JsonSchema{
		Type:        strings.ToLower(schema.Type[0]),
		Description: schema.Description,
	}

	switch s.Type {
	case mcpfile.JsonSchemaTypeArray:
		if schema.Items.IsA() {
			s.Items = convertSchema(schema.Items.A)
		}
	case mcpfile.JsonSchemaTypeObject:
		s.Properties = map[string]*mcpfile.JsonSchema{}
		for k, v := range schema.Properties.FromOldest() {
			s.Properties[k] = convertSchema(v)
		}
	}

	return s
}

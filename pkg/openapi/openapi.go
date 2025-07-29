package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
		Tools:   []*mcpfile.Tool{},
		Version: "0.0.1",
	}

	title := "mcpfile-generated"
	if model.Info != nil && model.Info.Title != "" {
		title = model.Info.Title
	}

	server.Name = title

	baseUrl := ""
	if len(model.Servers) > 0 {
		baseUrl = model.Servers[0].URL
	}

	var err error

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			tool := &mcpfile.Tool{
				Name:        fmt.Sprintf("%s.%s", pathName, operationMethod),
				Title:       operation.Summary,
				Description: operation.Description,
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

			if operation.RequestBody != nil {
				jsonSchema, ok := operation.RequestBody.Content.Get("application/json")
				if !ok {
					err = errors.Join(err, fmt.Errorf("no JSON schema defined on request body for %s, skipping tool", tool.Name))
					continue
				}

				reqSchema := convertSchema(jsonSchema.Schema)

				if reqSchema.Type != mcpfile.JsonSchemaTypeObject {
					// TODO: we probably want better error handling here
					err = errors.Join(err, fmt.Errorf("JSON schema defined on request body for %s is not an object, skipping tool", tool.Name))
					continue
				}

				maps.Copy(tool.InputSchema.Properties, reqSchema.Properties)

				tool.InputSchema.Required = append(tool.InputSchema.Required, reqSchema.Required...)

				tool.InputSchema.AdditionalProperties = reqSchema.AdditionalProperties

			}

			// hack to make sure everything is parsed correctly
			toolJson, jsonErr := json.Marshal(tool)
			if jsonErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to serialize tool %s into json, skipping tool: %w", tool.Name, jsonErr))
			}

			t := &mcpfile.Tool{}
			jsonErr = json.Unmarshal(toolJson, t)
			if jsonErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to deserialize tool %s from json, skipping tool: %w", tool.Name, jsonErr))
			}

			server.Tools = append(server.Tools, t)
		}
	}

	validationErr := server.Validate()
	if validationErr != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to validate converted server: %w", validationErr))
	}

	res.Servers = []*mcpfile.MCPServer{server}

	return res, err
}

func convertSchema(proxy *highbase.SchemaProxy) *mcpfile.JsonSchema {
	schema := proxy.Schema()
	s := &mcpfile.JsonSchema{
		Type:        strings.ToLower(schema.Type[0]),
		Description: schema.Description,
	}

	switch schema.Type[0] {
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

	if schema.AdditionalProperties != nil && (schema.AdditionalProperties.IsA() || (schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B)) {
		val := true
		s.AdditionalProperties = &val
	}

	return s
}

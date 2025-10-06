package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	ihttps "github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func DocumentToMcpFile(document []byte, host string) (*mcpfile.MCPFile, error) {
	doc, err := libopenapi.NewDocument(document)
	if err != nil {
		return nil, fmt.Errorf("failed to create openapi document: %w", err)
	}

	if strings.HasPrefix(doc.GetVersion(), "3") {
		docModel, err := doc.BuildV3Model()
		if err != nil {
			return nil, fmt.Errorf("failed to build OpenAPI V3 model: %w", err)
		}
		return McpFileFromOpenApiV3Model(&docModel.Model, host)
	}

	docModel, err := doc.BuildV2Model()
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI V2 model: %w", err)
	}
	return McpFileFromOpenApiV2Model(&docModel.Model, host)
}

func McpFileFromOpenApiV2Model(model *v2high.Swagger, host string) (*mcpfile.MCPFile, error) {
	if model.Host == "" && host == "" {
		return nil, fmt.Errorf("no host provided in the swagger file, unable to construct valid URLs")
	}
	// 1. Set top level MCP file info
	// 2. Create server in the MCP file, default to streamablehttp transport w. port 8080
	// 3 for each (path, operation) in the document, add one tool to the server w. http invoke
	server := &mcpfile.MCPFile{
		FileVersion: mcpfile.MCPFileVersion,
		MCPServer: mcpfile.MCPServer{
			Runtime: &mcpfile.ServerRuntime{
				TransportProtocol: mcpfile.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &mcpfile.StreamableHTTPConfig{
					Port: 8080,
				},
			},
			Tools:   []*mcpfile.Tool{},
			Version: "0.0.1",
		},
	}

	title := "mcpfile-generated"
	if model.Info != nil && model.Info.Title != "" {
		title = model.Info.Title
	}

	server.Name = title

	var err error
	var scheme string
	if model.Schemes == nil {
		scheme = "http"
		err = fmt.Errorf("no schemes set on swagger document, defaulting to http")
	} else if slices.Contains(model.Schemes, "https") {
		scheme = "https"
	} else if slices.Contains(model.Schemes, "http") {
		scheme = "http"
	} else {
		return nil, fmt.Errorf("no valid scheme set on swagger document: gen-mcp requires one of (http, https)")
	}

	urlHost := model.Host
	if host != "" {
		urlHost = host
	}

	baseUrl := fmt.Sprintf("%s://%s%s", scheme, urlHost, model.BasePath)

	if model.Paths == nil || model.Paths.PathItems == nil {
		return nil, fmt.Errorf("no valid paths on the openapi document")
	}

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			if !ihttps.IsValidHttpMethod(operationMethod) {
				err = errors.Join(err, fmt.Errorf("%s is not a supported http method, skipping %s", operationMethod, toolName(pathName, operationMethod)))
				continue
			}

			invocationData, marshalErr := json.Marshal(map[string]any{
				"url":    fmt.Sprintf("%s%s", baseUrl, pathName),
				"method": strings.ToUpper(operationMethod),
			})
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal http invocation config for %s: %w", toolName(pathName, operationMethod), marshalErr))
				continue
			}

			tool := &mcpfile.Tool{
				Name:        toolName(pathName, operationMethod),
				Title:       operation.Summary,
				Description: operation.Description,
				InputSchema: &jsonschema.Schema{
					Type:       invocation.JsonSchemaTypeObject,
					Properties: make(map[string]*jsonschema.Schema),
					Required:   []string{},
				},
				InvocationData: invocationData,
				InvocationType: mcpfile.InvocationTypeHttp,
			}

			numPathParams := 0
			visited := make(map[*highbase.SchemaProxy]*jsonschema.Schema)
			for _, param := range operation.Parameters {
				// In OpenAPI 2.0, body parameters are special - the name is ignored
				// and the schema properties should be merged directly into the input schema
				if strings.ToLower(param.In) == "body" && param.Schema != nil {
					bodySchema := convertSchema(param.Schema, visited)
					if bodySchema.Type == invocation.JsonSchemaTypeObject {
						// Merge properties from body schema
						maps.Copy(tool.InputSchema.Properties, bodySchema.Properties)
						// Merge required fields
						tool.InputSchema.Required = append(tool.InputSchema.Required, bodySchema.Required...)
						// Preserve additional properties if set
						if bodySchema.AdditionalProperties != nil {
							tool.InputSchema.AdditionalProperties = bodySchema.AdditionalProperties
						}
					}
				} else {
					tool.InputSchema.Properties[param.Name] = convertV2Parameter(param, visited)

					if (param.Required != nil && *param.Required) || strings.ToLower(param.In) == "path" {
						tool.InputSchema.Required = append(tool.InputSchema.Required, param.Name)
						numPathParams++
					}
				}
			}

			for _, param := range pathItem.Parameters {
				// In OpenAPI 2.0, body parameters are special - the name is ignored
				// and the schema properties should be merged directly into the input schema
				if strings.ToLower(param.In) == "body" && param.Schema != nil {
					bodySchema := convertSchema(param.Schema, visited)
					if bodySchema.Type == invocation.JsonSchemaTypeObject {
						// Merge properties from body schema
						maps.Copy(tool.InputSchema.Properties, bodySchema.Properties)
						// Merge required fields
						tool.InputSchema.Required = append(tool.InputSchema.Required, bodySchema.Required...)
						// Preserve additional properties if set
						if bodySchema.AdditionalProperties != nil {
							tool.InputSchema.AdditionalProperties = bodySchema.AdditionalProperties
						}
					}
				} else {
					tool.InputSchema.Properties[param.Name] = convertV2Parameter(param, visited)

					if (param.Required != nil && *param.Required) || strings.ToLower(param.In) == "path" {
						tool.InputSchema.Required = append(tool.InputSchema.Required, param.Name)
						numPathParams++
					}
				}
			}

			consumes := model.Consumes
			if len(operation.Consumes) > 0 {
				consumes = operation.Consumes
			}

			if len(tool.InputSchema.Properties) > numPathParams &&
				(strings.ToUpper(operationMethod) == http.MethodPost ||
					strings.ToUpper(operationMethod) == http.MethodPut ||
					strings.ToUpper(operationMethod) == http.MethodPut) &&
				(!slices.Contains(consumes, "application/json") && !slices.Contains(consumes, "*/*")) {
				err = errors.Join(err, fmt.Errorf("endpoint for %s does not consume application/json, skipping tool", tool.Name))
				continue
			}

			server.Tools = append(server.Tools, tool)
		}
	}

	// the only errors we should see at this point are from the tools themselves - let's validate them and filter out invalid tools
	validTools := make([]*mcpfile.Tool, 0, len(server.Tools))
	for _, t := range server.Tools {
		toolErr := t.Validate(invocation.InvocationValidator)
		if toolErr != nil {
			err = errors.Join(err, fmt.Errorf("skipping tool %s: %w", t.Name, toolErr))
		} else {
			validTools = append(validTools, t)
		}
	}

	server.Tools = validTools

	return server, err
}
func McpFileFromOpenApiV3Model(model *v3high.Document, host string) (*mcpfile.MCPFile, error) {
	// 1. Set top level MCP file info
	// 2. Create server in the MCP file, default to streamablehttp transport w. port 8080
	// 3 for each (path, operation) in the document, add one tool to the server w. http invoke
	server := &mcpfile.MCPFile{
		FileVersion: mcpfile.MCPFileVersion,
		MCPServer: mcpfile.MCPServer{
			Runtime: &mcpfile.ServerRuntime{
				TransportProtocol: mcpfile.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &mcpfile.StreamableHTTPConfig{
					Port: 8080,
				},
			},
			Tools:   []*mcpfile.Tool{},
			Version: "0.0.1",
		},
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

	if host != "" {
		baseUrl = host
	}

	var err error

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			if !ihttps.IsValidHttpMethod(operationMethod) {
				err = errors.Join(err, fmt.Errorf("%s is not a supported http method, skipping %s", operationMethod, toolName(pathName, operationMethod)))
				continue
			}

			invocationData, marshalErr := json.Marshal(map[string]any{
				"url":    fmt.Sprintf("%s%s", baseUrl, pathName),
				"method": strings.ToUpper(operationMethod),
			})
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal http invocation config for %s: %w", toolName(pathName, operationMethod), marshalErr))
				continue
			}

			tool := &mcpfile.Tool{
				Name:        toolName(pathName, operationMethod),
				Title:       operation.Summary,
				Description: operation.Description,
				InputSchema: &jsonschema.Schema{
					Type:       invocation.JsonSchemaTypeObject,
					Properties: make(map[string]*jsonschema.Schema),
					Required:   []string{},
				},
				InvocationData: invocationData,
				InvocationType: mcpfile.InvocationTypeHttp,
			}

			visited := make(map[*highbase.SchemaProxy]*jsonschema.Schema)
			for _, param := range operation.Parameters {
				tool.InputSchema.Properties[param.Name] = convertSchema(param.Schema, visited)

				if (param.Required != nil && *param.Required) || strings.ToLower(param.In) == "path" {
					tool.InputSchema.Required = append(tool.InputSchema.Required, param.Name)
				}
			}

			for _, param := range pathItem.Parameters {
				tool.InputSchema.Properties[param.Name] = convertSchema(param.Schema, visited)

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

				reqSchema := convertSchema(jsonSchema.Schema, visited)

				if reqSchema.Type != invocation.JsonSchemaTypeObject {
					// TODO: we probably want better error handling here
					err = errors.Join(err, fmt.Errorf("JSON schema defined on request body for %s is not an object, skipping tool", tool.Name))
					continue
				}

				maps.Copy(tool.InputSchema.Properties, reqSchema.Properties)

				tool.InputSchema.Required = append(tool.InputSchema.Required, reqSchema.Required...)

				tool.InputSchema.AdditionalProperties = reqSchema.AdditionalProperties

			}

			server.Tools = append(server.Tools, tool)
		}
	}

	// the only errors we should see at this point are from the tools themselves - let's validate them and filter out invalid tools
	validTools := make([]*mcpfile.Tool, 0, len(server.Tools))
	for _, t := range server.Tools {
		toolErr := t.Validate(invocation.InvocationValidator)
		if toolErr != nil {
			err = errors.Join(err, fmt.Errorf("skipping tool %s: %w", t.Name, toolErr))
		} else {
			validTools = append(validTools, t)
		}
	}

	server.Tools = validTools

	return server, err
}

func convertSchema(proxy *highbase.SchemaProxy, visited map[*highbase.SchemaProxy]*jsonschema.Schema) *jsonschema.Schema {
	if proxy == nil {
		return nil
	}

	if s, ok := visited[proxy]; ok {
		js := &jsonschema.Schema{
			Type:                 s.Type,
			Description:          s.Description,
			AdditionalProperties: s.AdditionalProperties,
		}

		// hacks to break the cyclical JSON rendering
		if s.Type == invocation.JsonSchemaTypeArray && s.Items != nil {
			js.Items = &jsonschema.Schema{
				Type:        s.Items.Type,
				Description: s.Items.Description,
			}
		} else if s.Type == invocation.JsonSchemaTypeObject {
			js.AdditionalProperties = &jsonschema.Schema{}
		}

		return js
	}

	schema := proxy.Schema()
	schemaType := ""
	if len(schema.Type) > 0 {
		schemaType = schema.Type[0]
	}

	s := &jsonschema.Schema{
		Type:        strings.ToLower(schemaType),
		Description: schema.Description,
	}
	visited[proxy] = s

	switch schemaType {
	case invocation.JsonSchemaTypeArray:
		if schema.Items != nil && schema.Items.IsA() {
			s.Items = convertSchema(schema.Items.A, visited)
		}
	case invocation.JsonSchemaTypeObject:
		s.Properties = map[string]*jsonschema.Schema{}
		if schema.Properties != nil {
			for k, v := range schema.Properties.FromOldest() {
				s.Properties[k] = convertSchema(v, visited)
			}
		}
		// Add required fields for object schemas
		if schema.Required != nil && len(schema.Required) > 0 {
			s.Required = schema.Required
		}
	}

	if schema.AdditionalProperties != nil && (schema.AdditionalProperties.IsA() || (schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B)) {
		s.AdditionalProperties = &jsonschema.Schema{}
	}

	return s
}

func convertV2Parameter(param *v2high.Parameter, visited map[*highbase.SchemaProxy]*jsonschema.Schema) *jsonschema.Schema {
	if param.Schema != nil {
		return convertSchema(param.Schema, visited)
	}

	s := &jsonschema.Schema{
		Type:        strings.ToLower(param.Type),
		Description: param.Description,
	}
	if s.Type == invocation.JsonSchemaTypeArray {
		if param.Items != nil {
			s.Items = &jsonschema.Schema{
				Type: strings.ToLower(param.Items.Type),
			}
		}
	}

	return s
}

func toolName(path, operation string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	pathParts := strings.Split(path, "/")
	return fmt.Sprintf("%s_%s", operation, strings.Join(pathParts, "-"))
}

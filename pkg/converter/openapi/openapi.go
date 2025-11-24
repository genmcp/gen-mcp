package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/config"
	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	serverconfig "github.com/genmcp/gen-mcp/pkg/config/server"
	"github.com/genmcp/gen-mcp/pkg/invocation"
	"github.com/genmcp/gen-mcp/pkg/invocation/extends"
	ihttps "github.com/genmcp/gen-mcp/pkg/invocation/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

const (
	baseApiInvocationName = "baseApi"
)

// ConvertedMCPFiles contains both the tool definitions and server config files
type ConvertedMCPFiles struct {
	ToolDefinitions *definitions.MCPToolDefinitionsFile
	ServerConfig    *serverconfig.MCPServerConfigFile
}

func DocumentToMcpFile(document []byte, host string) (*ConvertedMCPFiles, error) {
	doc, err := libopenapi.NewDocument(document)
	if err != nil {
		return nil, fmt.Errorf("failed to create openapi document: %w", err)
	}

	if strings.HasPrefix(doc.GetVersion(), "3") {
		docModel, err := doc.BuildV3Model()
		if err != nil {
			return nil, fmt.Errorf("failed to build OpenAPI V3 model: %w", err)
		}
		return McpFilesFromOpenApiV3Model(&docModel.Model, host)
	}

	docModel, err := doc.BuildV2Model()
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI V2 model: %w", err)
	}
	return McpFilesFromOpenApiV2Model(&docModel.Model, host)
}

func McpFilesFromOpenApiV2Model(model *v2high.Swagger, host string) (*ConvertedMCPFiles, error) {
	if model.Host == "" && host == "" {
		return nil, fmt.Errorf("no host provided in the swagger file, unable to construct valid URLs")
	}
	// 1. Set top level GenMCP config file info
	// 2. Create server config file with runtime configuration
	// 3. Create MCP file with tools
	// 4. For each (path, operation) in the document, add one tool w. http invoke

	title := "mcpfile-generated"
	if model.Info != nil && model.Info.Title != "" {
		title = model.Info.Title
	}

	version := "0.0.1"
	if model.Info != nil && model.Info.Version != "" {
		version = model.Info.Version
	}

	// Create server config file
	serverConfig := &serverconfig.MCPServerConfigFile{
		Kind:          serverconfig.KindMCPServerConfig,
		SchemaVersion: config.SchemaVersion,
		MCPServerConfig: serverconfig.MCPServerConfig{
			Runtime: &serverconfig.ServerRuntime{
				TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
					Port: 8080,
				},
			},
		},
	}

	// Create MCP file
	toolDefinitions := &definitions.MCPToolDefinitionsFile{
		Kind:          definitions.KindMCPToolDefinitions,
		SchemaVersion: config.SchemaVersion,
		MCPToolDefinitions: definitions.MCPToolDefinitions{
			Name:            title,
			Version:         version,
			Tools:           []*definitions.Tool{},
			InvocationBases: map[string]*invocation.InvocationConfigWrapper{},
		},
	}

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

	baseInvocation := &invocation.InvocationConfigWrapper{
		Type: ihttps.InvocationType,
		Config: &ihttps.HttpInvocationConfig{
			URL: baseUrl,
		},
	}

	// Set invocation bases in MCP file
	toolDefinitions.InvocationBases[baseApiInvocationName] = baseInvocation

	if model.Paths == nil || model.Paths.PathItems == nil {
		return nil, fmt.Errorf("no valid paths on the openapi document")
	}

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			if !ihttps.IsValidHttpMethod(operationMethod) {
				err = errors.Join(err, fmt.Errorf("%s is not a supported http method, skipping %s", operationMethod, toolName(pathName, operationMethod)))
				continue
			}

			extend := &ihttps.HttpInvocationConfig{
				URL: pathName,
			}

			extendRaw, marshalErr := json.Marshal(extend)
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal tool extension: %w", marshalErr))
				continue
			}

			override := &ihttps.HttpInvocationConfig{
				Method: strings.ToUpper(operationMethod),
			}

			overrideRaw, marshalErr := json.Marshal(override)
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal tool override: %w", marshalErr))
				continue
			}

			tool := &definitions.Tool{
				Name:        toolName(pathName, operationMethod),
				Title:       operation.Summary,
				Description: operation.Description,
				InputSchema: &jsonschema.Schema{
					Type:       invocation.JsonSchemaTypeObject,
					Properties: make(map[string]*jsonschema.Schema),
					Required:   []string{},
				},
				InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
					Type: extends.InvocationType,
					Config: &extends.ExtendsConfig{
						From:     baseApiInvocationName,
						Extend:   extendRaw,
						Override: overrideRaw,
					},
				},
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

			toolDefinitions.Tools = append(toolDefinitions.Tools, tool)
		}
	}

	// the only errors we should see at this point are from the tools themselves - let's validate them and filter out invalid tools
	extends.SetBases(toolDefinitions.InvocationBases)
	validTools := make([]*definitions.Tool, 0, len(toolDefinitions.Tools))
	for _, t := range toolDefinitions.Tools {
		toolErr := t.Validate(invocation.InvocationValidator)
		if toolErr != nil {
			err = errors.Join(err, fmt.Errorf("skipping tool %s: %w", t.Name, toolErr))
		} else {
			validTools = append(validTools, t)
		}
	}

	toolDefinitions.Tools = validTools

	return &ConvertedMCPFiles{
		ToolDefinitions: toolDefinitions,
		ServerConfig:    serverConfig,
	}, err
}
func McpFilesFromOpenApiV3Model(model *v3high.Document, host string) (*ConvertedMCPFiles, error) {
	// 1. Set top level GenMCP config file info
	// 2. Create server config file with runtime configuration
	// 3. Create MCP file with tools
	// 4. For each (path, operation) in the document, add one tool w. http invoke

	title := "mcpfile-generated"
	if model.Info != nil && model.Info.Title != "" {
		title = model.Info.Title
	}

	version := "0.0.1"
	if model.Info != nil && model.Info.Version != "" {
		version = model.Info.Version
	}

	// Create server config file
	serverConfig := &serverconfig.MCPServerConfigFile{
		Kind:          serverconfig.KindMCPServerConfig,
		SchemaVersion: config.SchemaVersion,
		MCPServerConfig: serverconfig.MCPServerConfig{
			Runtime: &serverconfig.ServerRuntime{
				TransportProtocol: serverconfig.TransportProtocolStreamableHttp,
				StreamableHTTPConfig: &serverconfig.StreamableHTTPConfig{
					Port: 8080,
				},
			},
		},
	}

	// Create MCP file
	toolDefinitions := &definitions.MCPToolDefinitionsFile{
		Kind:          definitions.KindMCPToolDefinitions,
		SchemaVersion: config.SchemaVersion,
		MCPToolDefinitions: definitions.MCPToolDefinitions{
			Name:            title,
			Version:         version,
			Tools:           []*definitions.Tool{},
			InvocationBases: map[string]*invocation.InvocationConfigWrapper{},
		},
	}

	baseUrl := ""
	if len(model.Servers) > 0 {
		baseUrl = model.Servers[0].URL
	}

	if host != "" {
		baseUrl = host
	}

	baseInvocation := &invocation.InvocationConfigWrapper{
		Type: ihttps.InvocationType,
		Config: &ihttps.HttpInvocationConfig{
			URL: baseUrl,
		},
	}

	// Set invocation bases in MCP file
	toolDefinitions.InvocationBases[baseApiInvocationName] = baseInvocation

	var err error

	for pathName, pathItem := range model.Paths.PathItems.FromOldest() {
		for operationMethod, operation := range pathItem.GetOperations().FromOldest() {
			if !ihttps.IsValidHttpMethod(operationMethod) {
				err = errors.Join(err, fmt.Errorf("%s is not a supported http method, skipping %s", operationMethod, toolName(pathName, operationMethod)))
				continue
			}

			extend := &ihttps.HttpInvocationConfig{
				URL: pathName,
			}

			extendRaw, marshalErr := json.Marshal(extend)
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal tool extension: %w", marshalErr))
				continue
			}

			override := &ihttps.HttpInvocationConfig{
				Method: strings.ToUpper(operationMethod),
			}

			overrideRaw, marshalErr := json.Marshal(override)
			if marshalErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to marshal tool override: %w", marshalErr))
				continue
			}

			tool := &definitions.Tool{
				Name:        toolName(pathName, operationMethod),
				Title:       operation.Summary,
				Description: operation.Description,
				InputSchema: &jsonschema.Schema{
					Type:       invocation.JsonSchemaTypeObject,
					Properties: make(map[string]*jsonschema.Schema),
					Required:   []string{},
				},
				InvocationConfigWrapper: &invocation.InvocationConfigWrapper{
					Type: extends.InvocationType,
					Config: &extends.ExtendsConfig{
						From:     baseApiInvocationName,
						Extend:   extendRaw,
						Override: overrideRaw,
					},
				},
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

			toolDefinitions.Tools = append(toolDefinitions.Tools, tool)
		}
	}

	// the only errors we should see at this point are from the tools themselves - let's validate them and filter out invalid tools
	extends.SetBases(toolDefinitions.InvocationBases)
	validTools := make([]*definitions.Tool, 0, len(toolDefinitions.Tools))
	for _, t := range toolDefinitions.Tools {
		toolErr := t.Validate(invocation.InvocationValidator)
		if toolErr != nil {
			err = errors.Join(err, fmt.Errorf("skipping tool %s: %w", t.Name, toolErr))
		} else {
			validTools = append(validTools, t)
		}
	}

	toolDefinitions.Tools = validTools

	return &ConvertedMCPFiles{
		ToolDefinitions: toolDefinitions,
		ServerConfig:    serverConfig,
	}, err
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
		if len(schema.Required) > 0 {
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

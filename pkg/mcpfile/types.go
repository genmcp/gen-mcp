package mcpfile

import (
	"net/url"
)

const (
	MCPFileVersion        = "0.0.1"
	JsonSchemaTypeArray   = "array"
	JsonSchemaTypeBoolean = "boolean"
	JsonSchemaTypeInteger = "integer"
	JsonSchemaTypeNumber  = "number"
	JsonSchemaTypeNull    = "null"
	JsonSchemaTypeObject  = "object"
	JsonSchemaTypeString  = "string"
)

type JsonSchema struct {
	Type                 string                 `json:"type"`                           // can be array, boolean, integer, number, null, object, or string
	Items                *JsonSchema            `json:"items,omitempty"`                // schema for items of an array
	Properties           map[string]*JsonSchema `json:"properties,omitempty"`           // properties of an object
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"` // allow extra properties for type object
	Required             []string               `json:"required,omitempty"`             // required properties for an object
	Description          string                 `json:"description,omitempty"`          // optional human readable description of the item
}

type Tool struct {
	Name         string      `json:"name"`                   // name of the tool
	Title        string      `json:"title,omitempty"`        // optional human readable name of the tool, for client display
	Description  string      `json:"description"`            // description of the tool
	InputSchema  *JsonSchema `json:"inputSchema"`            // input schema to call the tool
	OutputSchema *JsonSchema `json:"outputSchema,omitempty"` // optional output schema of the tool
	URL          url.URL     `json:"url"`                    // url for where the MCP server should call the tool
}

type MCPServer struct {
	Name    string  `json:"name"`    // name of the server
	Version string  `json:"version"` // version of the server
	Tools   []*Tool `json:"tools,omitempty"`   // set of tools available to the server
}

type MCPFile struct {
	FileVersion string       `json:"mcpFileVersion"`
	Servers     []*MCPServer `json:"servers,omitempty"`
}

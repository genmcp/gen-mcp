package mcpfile

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
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
	InvocationTypeHttp    = "http"
)

type JsonSchema struct {
	Type                 string                 `json:"type"`                           // can be array, boolean, integer, number, null, object, or string
	Items                *JsonSchema            `json:"items,omitempty"`                // schema for items of an array
	Properties           map[string]*JsonSchema `json:"properties,omitempty"`           // properties of an object
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"` // allow extra properties for type object
	Required             []string               `json:"required,omitempty"`             // required properties for an object
	Description          string                 `json:"description,omitempty"`          // optional human readable description of the item
}

type Invocation interface {
	HandleRequest(ctx context.Context, req mcp.CallToolRequest, t *Tool) (*mcp.CallToolResult, error) // handle the relevant tool call request
	Validate(*Tool) error
}

type HttpInvocation struct {
	URL            string   `json:"url"`    // the url to make the request to
	Method         string   `json:"method"` // the request method
	pathParameters []string // parameters to extract from the InputSchema into the URL path
}

var _ Invocation = &HttpInvocation{}

type Tool struct {
	Name         string      `json:"name"`                   // name of the tool
	Title        string      `json:"title,omitempty"`        // optional human readable name of the tool, for client display
	Description  string      `json:"description"`            // description of the tool
	InputSchema  *JsonSchema `json:"inputSchema"`            // input schema to call the tool
	OutputSchema *JsonSchema `json:"outputSchema,omitempty"` // optional output schema of the tool
	Invocation   Invocation  `json:"invocation"`             // how the tool should be invoked
}

type MCPServer struct {
	Name    string  `json:"name"`            // name of the server
	Version string  `json:"version"`         // version of the server
	Tools   []*Tool `json:"tools,omitempty"` // set of tools available to the server
}

type MCPFile struct {
	FileVersion string       `json:"mcpFileVersion"`
	Servers     []*MCPServer `json:"servers,omitempty"`
}

package mcpfile

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	MCPFileVersion                  = "0.0.1"
	JsonSchemaTypeArray             = "array"
	JsonSchemaTypeBoolean           = "boolean"
	JsonSchemaTypeInteger           = "integer"
	JsonSchemaTypeNumber            = "number"
	JsonSchemaTypeNull              = "null"
	JsonSchemaTypeObject            = "object"
	JsonSchemaTypeString            = "string"
	InvocationTypeHttp              = "http"
	InvocationTypeCli               = "cli"
	TransportProtocolStreamableHttp = "streamablehttp"
	TransportProtocolStdio          = "stdio"
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

type TemplateVariable struct {
	Property         string   `json:"property,omitempty"`                    // the property on the input schema
	Format           string   `json:"format,omitempty"`                      // the format to output this variable
	OmitIfFalse      bool     `json:"omitIfFalse,omitempty" default:"false"` // whether to omit the variable if it is false
	formatParameters []string // parameters to place into the variable format from the input property
}

type CliInvocation struct {
	Command           string                       `json:"command"`                     // the terminal command to run
	TemplateVariables map[string]*TemplateVariable `json:"templateVariables,omitempty"` // information on how to map the template variables
	commandParameters []string                     // parameters found in the command, in order
}

var _ Invocation = &HttpInvocation{}

type Tool struct {
	Name           string      `json:"name"`                     // name of the tool
	Title          string      `json:"title,omitempty"`          // optional human readable name of the tool, for client display
	Description    string      `json:"description"`              // description of the tool
	InputSchema    *JsonSchema `json:"inputSchema"`              // input schema to call the tool
	OutputSchema   *JsonSchema `json:"outputSchema,omitempty"`   // optional output schema of the tool
	Invocation     Invocation  `json:"invocation"`               // how the tool should be invoked
	RequiredScopes []string    `json:"requiredScopes,omitempty"` // required OAuth scopes to be able to use the tool
}

type StreamableHTTPConfig struct {
	Port     int         `json:"port"`           // the port to start listening on
	BasePath string      `json:"basePath"`       // the base path for the MCP server
	Auth     *AuthConfig `json:"auth,omitempty"` // OAuth 2.0 configuration for protected resource
	TLS      *TLSConfig  `json:"tls,omitempty"`  // TLS configuration for the http server
}

type TLSConfig struct {
	CertFile string `json:"certFile,omitempty"` // The absolute path to the server's public certificate
	KeyFile  string `json:"keyFile,omitempty"`  // The absolute path to the server's private key
}

type AuthConfig struct {
	AuthorizationServers []string `json:"authorizationServers,omitempty"` // list of authorization server URLs
	JWKSURI              string   `json:"jwksUri,omitempty"`              // JSON Web Key Set URI
}

type StdioConfig struct {
}

type ServerRuntime struct {
	TransportProtocol    string                `json:"transportProtocol"`              // which transport protocol to use
	StreamableHTTPConfig *StreamableHTTPConfig `json:"streamableHttpConfig,omitempty"` // config for the streamable http transport protocol
	StdioConfig          *StdioConfig          `json:"stdioConfig,omitempty"`          // config for the stdio transport protocol
}

type MCPServer struct {
	Name    string         `json:"name"`              // name of the server
	Version string         `json:"version"`           // version of the server
	Runtime *ServerRuntime `json:"runtime,omitempty"` // runtime settings for the server
	Tools   []*Tool        `json:"tools,omitempty"`   // set of tools available to the server
}

type MCPFile struct {
	FileVersion string       `json:"mcpFileVersion"`
	Servers     []*MCPServer `json:"servers,omitempty"`
}

package mcpfile

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	MCPFileVersion                  = "0.1.0"
	InvocationTypeHttp              = "http"
	InvocationTypeCli               = "cli"
	TransportProtocolStreamableHttp = "streamablehttp"
	TransportProtocolStdio          = "stdio"
	PrimitiveTypeTool               = "tool"
	PrimitiveTypePrompt             = "prompt"
	PrimitiveTypeResource           = "resource"
	PrimitiveTypeResourceTemplate   = "resourceTemplate"
)

type Primitive interface {
	GetName() string
	GetDescription() string
	GetInputSchema() *jsonschema.Schema
	GetOutputSchema() *jsonschema.Schema
	GetInvocationData() json.RawMessage
	GetInvocationType() string
	GetRequiredScopes() []string
	GetResolvedInputSchema() *jsonschema.Resolved
	PrimitiveType() string
}

type Tool struct {
	Name           string             `json:"name"`                     // name of the tool
	Title          string             `json:"title,omitempty"`          // optional human readable name of the tool, for client display
	Description    string             `json:"description"`              // description of the tool
	InputSchema    *jsonschema.Schema `json:"inputSchema"`              // input schema to call the tool
	OutputSchema   *jsonschema.Schema `json:"outputSchema,omitempty"`   // optional output schema of the tool
	InvocationData json.RawMessage    `json:"invocation"`               // how the tool should be invoked
	InvocationType string             `json:"-"`                        // which invocation type should be used
	RequiredScopes []string           `json:"requiredScopes,omitempty"` // required OAuth scopes to be able to use the tool

	ResolvedInputSchema *jsonschema.Resolved `json:"-"` // used internally after resolving the schema during validation
}

func (t Tool) GetName() string                              { return t.Name }
func (t Tool) GetDescription() string                       { return t.Description }
func (t Tool) PrimitiveType() string                        { return PrimitiveTypeTool }
func (t Tool) GetInputSchema() *jsonschema.Schema           { return t.InputSchema }
func (t Tool) GetOutputSchema() *jsonschema.Schema          { return t.OutputSchema }
func (t Tool) GetInvocationData() json.RawMessage           { return t.InvocationData }
func (t Tool) GetInvocationType() string                    { return t.InvocationType }
func (t Tool) GetRequiredScopes() []string                  { return t.RequiredScopes }
func (t Tool) GetResolvedInputSchema() *jsonschema.Resolved { return t.ResolvedInputSchema }

type Prompt struct {
	Name           string             `json:"name"`                     // name of the prompt
	Title          string             `json:"title,omitempty"`          // optional human readable name of the prompt, for client display
	Description    string             `json:"description"`              // description of the prompt
	Arguments      []*PromptArgument  `json:"arguments,omitempty"`      // list of arguments to use for templating the prompt.
	InputSchema    *jsonschema.Schema `json:"inputSchema"`              // input schema to call the prompt
	OutputSchema   *jsonschema.Schema `json:"outputSchema,omitempty"`   // optional output schema of the prompt
	InvocationData json.RawMessage    `json:"invocation"`               // how the prompt should be invoked
	InvocationType string             `json:"-"`                        // which invocation type should be used
	RequiredScopes []string           `json:"requiredScopes,omitempty"` // required OAuth scopes to be able to use the prompt

	ResolvedInputSchema *jsonschema.Resolved `json:"-"` // used internally after resolving the schema during validation
}

func (p Prompt) GetName() string                              { return p.Name }
func (p Prompt) GetDescription() string                       { return p.Description }
func (p Prompt) PrimitiveType() string                        { return PrimitiveTypePrompt }
func (p Prompt) GetInputSchema() *jsonschema.Schema           { return p.InputSchema }
func (p Prompt) GetOutputSchema() *jsonschema.Schema          { return p.OutputSchema }
func (p Prompt) GetInvocationData() json.RawMessage           { return p.InvocationData }
func (p Prompt) GetInvocationType() string                    { return p.InvocationType }
func (p Prompt) GetRequiredScopes() []string                  { return p.RequiredScopes }
func (p Prompt) GetResolvedInputSchema() *jsonschema.Resolved { return p.ResolvedInputSchema }

type PromptArgument struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Resource struct {
	Name           string             `json:"name"`                     // name of the resource
	Title          string             `json:"title,omitempty"`          // optional human readable name of the resource, for client display
	Description    string             `json:"description"`              // description of the resource
	MIMEType       string             `json:"mimeType,omitempty"`       // The MIME type of this resource, if known.
	Size           int64              `json:"size,omitempty"`           // The size of the raw resource content, in bytes (i.e., before base64 encoding or any tokenization), if known.
	URI            string             `json:"uri"`                      // The URI of this resource.
	InputSchema    *jsonschema.Schema `json:"inputSchema"`              // input schema to call the resource
	OutputSchema   *jsonschema.Schema `json:"outputSchema,omitempty"`   // optional output schema of the resource
	InvocationData json.RawMessage    `json:"invocation"`               // how the resource should be invoked
	InvocationType string             `json:"-"`                        // which invocation type should be used
	RequiredScopes []string           `json:"requiredScopes,omitempty"` // required OAuth scopes to be able to use the resource

	ResolvedInputSchema *jsonschema.Resolved `json:"-"` // used internally after resolving the schema during validation
}

func (r Resource) GetName() string                              { return r.Name }
func (r Resource) GetDescription() string                       { return r.Description }
func (r Resource) PrimitiveType() string                        { return PrimitiveTypeResource }
func (r Resource) GetInputSchema() *jsonschema.Schema           { return r.InputSchema }
func (r Resource) GetOutputSchema() *jsonschema.Schema          { return r.OutputSchema }
func (r Resource) GetInvocationData() json.RawMessage           { return r.InvocationData }
func (r Resource) GetInvocationType() string                    { return r.InvocationType }
func (r Resource) GetRequiredScopes() []string                  { return r.RequiredScopes }
func (r Resource) GetResolvedInputSchema() *jsonschema.Resolved { return r.ResolvedInputSchema }

type ResourceTemplate struct {
	Name           string             `json:"name"`                     // name of the resource template
	Title          string             `json:"title,omitempty"`          // optional human readable name of the resource template, for client display
	Description    string             `json:"description"`              // description of the resource template
	MIMEType       string             `json:"mimeType,omitempty"`       // The MIME type for all resources that match this template
	URITemplate    string             `json:"uriTemplate"`              // A URI template (according to RFC 6570) that can be used to construct resource URI
	InputSchema    *jsonschema.Schema `json:"inputSchema"`              // input schema to call the resource template
	OutputSchema   *jsonschema.Schema `json:"outputSchema,omitempty"`   // optional output schema of the resource template
	InvocationData json.RawMessage    `json:"invocation"`               // how the resource template should be invoked
	InvocationType string             `json:"-"`                        // which invocation type should be used
	RequiredScopes []string           `json:"requiredScopes,omitempty"` // required OAuth scopes to be able to use the resource template

	ResolvedInputSchema *jsonschema.Resolved `json:"-"` // used internally after resolving the schema during validation
}

func (r ResourceTemplate) GetName() string                              { return r.Name }
func (r ResourceTemplate) GetDescription() string                       { return r.Description }
func (r ResourceTemplate) PrimitiveType() string                        { return PrimitiveTypeResourceTemplate }
func (r ResourceTemplate) GetInputSchema() *jsonschema.Schema           { return r.InputSchema }
func (r ResourceTemplate) GetOutputSchema() *jsonschema.Schema          { return r.OutputSchema }
func (r ResourceTemplate) GetInvocationData() json.RawMessage           { return r.InvocationData }
func (r ResourceTemplate) GetInvocationType() string                    { return r.InvocationType }
func (r ResourceTemplate) GetRequiredScopes() []string                  { return r.RequiredScopes }
func (r ResourceTemplate) GetResolvedInputSchema() *jsonschema.Resolved { return r.ResolvedInputSchema }

type StreamableHTTPConfig struct {
	Port      int         `json:"port"`           // the port to start listening on
	BasePath  string      `json:"basePath"`       // the base path for the MCP server
	Stateless bool        `json:"stateless"`      // whether or not the server will be stateless
	Auth      *AuthConfig `json:"auth,omitempty"` // OAuth 2.0 configuration for protected resource
	TLS       *TLSConfig  `json:"tls,omitempty"`  // TLS configuration for the http server
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
	Name              string              `json:"name"`                        // name of the server
	Version           string              `json:"version"`                     // version of the server
	Runtime           *ServerRuntime      `json:"runtime,omitempty"`           // runtime settings for the server
	Tools             []*Tool             `json:"tools,omitempty"`             // set of tools available to the server
	Prompts           []*Prompt           `json:"prompts,omitempty"`           // set of prompts available to the server
	Resources         []*Resource         `json:"resources,omitempty"`         // set of resources available to the server
	ResourceTemplates []*ResourceTemplate `json:"resourceTemplates,omitempty"` // set of resource templates available to the server
}

type MCPFile struct {
	FileVersion string `json:"mcpFileVersion"`
	MCPServer   `json:",inline"`
}

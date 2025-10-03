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
)

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
	Name    string         `json:"name"`              // name of the server
	Version string         `json:"version"`           // version of the server
	Runtime *ServerRuntime `json:"runtime,omitempty"` // runtime settings for the server
	Tools   []*Tool        `json:"tools,omitempty"`   // set of tools available to the server
}

type MCPFile struct {
	FileVersion string `json:"mcpFileVersion"`
	MCPServer   `json:",inline"`
}

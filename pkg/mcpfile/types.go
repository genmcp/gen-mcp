package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/genmcp/gen-mcp/pkg/observability/logging"
	"github.com/google/jsonschema-go/jsonschema"
	"go.uber.org/zap"
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

// Primitive represents a tool-like entity that can be invoked (Tool, Prompt, Resource, or ResourceTemplate).
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

// Tool represents an executable capability of the MCP server.
type Tool struct {
	// Unique identifier for the tool.
	Name string `json:"name" jsonschema:"required"`

	// Human-readable title for display purposes.
	Title string `json:"title,omitempty" jsonschema:"optional"`

	// Detailed description of the tool's purpose.
	Description string `json:"description" jsonschema:"required"`

	// JSON Schema describing input parameters.
	InputSchema *jsonschema.Schema `json:"inputSchema" jsonschema:"required"`

	// Optional JSON Schema describing output.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty" jsonschema:"optional"`

	// Object describing how to execute the tool.
	InvocationData json.RawMessage `json:"invocation" jsonschema:"required,oneof_ref=#/$defs/HttpInvocationData;#/$defs/CliInvocationData"`

	// Invocation type ("http" or "cli"). Populated internally.
	InvocationType string `json:"-"`

	// OAuth scopes required to invoke this tool.
	RequiredScopes []string `json:"requiredScopes,omitempty" jsonschema:"optional"`

	// Annotations to indicate tool behaviour to the client.
	Annotations *ToolAnnotations `json:"annotations" jsonschema:"optional"`

	// Resolved input schema for validation (internal use only).
	ResolvedInputSchema *jsonschema.Resolved `json:"-"`
}

type ToolAnnotations struct {
	// If true, the tool may perform destructive updates to its environemnt. If
	// false, the tool performs only additive updates
	DesctructiveHint *bool `json:"destructiveHint,omitempty" jsonschema:"optional"`

	// If true, calling the tool repeatedly with the same arguments will have no additional
	// effect on its environment
	IdempotentHint *bool `json:"idempotentHint,omitempty" jsonschema:"optional"`

	// If true, this tool may interact with an "open world" or external entities. If
	// false, this tool's domain of interaction is closed. For example, the world of
	// a web search tool is open, wherease that of a memory tool is not.
	OpenWorldHint *bool `json:"openWorldHint,omitempty" jsonschema:"optional"`

	// If true, the tool does not modify its environment
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty" jsonschema:"optional"`
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

// Prompt represents a natural-language or LLM-style function invocation.
type Prompt struct {
	// Unique identifier for the prompt.
	Name string `json:"name" jsonschema:"required"`

	// Human-readable title for display purposes.
	Title string `json:"title,omitempty" jsonschema:"optional"`

	// Detailed description of what the prompt does.
	Description string `json:"description" jsonschema:"required"`

	// List of template arguments for the prompt.
	Arguments []*PromptArgument `json:"arguments,omitempty" jsonschema:"optional"`

	// Schema describing input parameters.
	InputSchema *jsonschema.Schema `json:"inputSchema" jsonschema:"required"`

	// Optional schema describing prompt output.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty" jsonschema:"optional"`

	// Object describing how to invoke the prompt.
	InvocationData json.RawMessage `json:"invocation" jsonschema:"required,oneof_ref=#/$defs/HttpInvocationData;#/$defs/CliInvocationData"`

	// Invocation type ("http" or "cli"). Determined dynamically.
	InvocationType string `json:"-"`

	// OAuth scopes required to invoke this prompt.
	RequiredScopes []string `json:"requiredScopes,omitempty" jsonschema:"optional"`

	// Resolved input schema for validation (internal use only).
	ResolvedInputSchema *jsonschema.Resolved `json:"-"`
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

// PromptArgument defines a variable that can be substituted into a prompt template.
type PromptArgument struct {
	// Unique identifier for the argument.
	Name string `json:"name" jsonschema:"required"`

	// Human-readable title for display.
	Title string `json:"title,omitempty" jsonschema:"optional"`

	// Detailed explanation of the argument.
	Description string `json:"description,omitempty" jsonschema:"optional"`

	// Indicates if the argument is mandatory.
	Required bool `json:"required,omitempty" jsonschema:"optional"`
}

// Resource represents a retrievable or executable resource.
type Resource struct {
	// Unique identifier for the resource.
	Name string `json:"name" jsonschema:"required"`

	// Human-readable title for display purposes.
	Title string `json:"title,omitempty" jsonschema:"optional"`

	// Detailed description of the resource.
	Description string `json:"description" jsonschema:"required"`

	// The MIME type of this resource, if known.
	MIMEType string `json:"mimeType,omitempty" jsonschema:"optional"`

	// The size of the raw resource content in bytes, if known.
	Size int64 `json:"size,omitempty" jsonschema:"optional"`

	// The URI of this resource.
	URI string `json:"uri" jsonschema:"required"`

	// Schema describing input parameters.
	InputSchema *jsonschema.Schema `json:"inputSchema" jsonschema:"required"`

	// Optional schema describing resource output.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty" jsonschema:"optional"`

	// Object describing how to invoke the resource.
	InvocationData json.RawMessage `json:"invocation" jsonschema:"required,oneof_ref=#/$defs/HttpInvocationData;#/$defs/CliInvocationData"`

	// Invocation type ("http" or "cli"). Determined dynamically.
	InvocationType string `json:"-"`

	// OAuth scopes required to access this resource.
	RequiredScopes []string `json:"requiredScopes,omitempty" jsonschema:"optional"`

	// Resolved input schema for validation (internal use only).
	ResolvedInputSchema *jsonschema.Resolved `json:"-"`
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

// ResourceTemplate represents a reusable URI-based template for resources.
type ResourceTemplate struct {
	// Unique identifier for the resource template.
	Name string `json:"name" jsonschema:"required"`

	// Human-readable title for display purposes.
	Title string `json:"title,omitempty" jsonschema:"optional"`

	// Detailed description of the resource template.
	Description string `json:"description" jsonschema:"required"`

	// MIME type for resources matching this template.
	MIMEType string `json:"mimeType,omitempty" jsonschema:"optional"`

	// URI template (RFC 6570) used to construct resource URIs.
	URITemplate string `json:"uriTemplate" jsonschema:"required"`

	// Schema describing input parameters.
	InputSchema *jsonschema.Schema `json:"inputSchema" jsonschema:"required"`

	// Optional schema describing resource output.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty" jsonschema:"optional"`

	// Object describing how to invoke the resource template.
	InvocationData json.RawMessage `json:"invocation" jsonschema:"required,oneof_ref=#/$defs/HttpInvocationData;#/$defs/CliInvocationData"`

	// Invocation type ("http" or "cli"). Determined dynamically.
	InvocationType string `json:"-"`

	// OAuth scopes required to access this resource template.
	RequiredScopes []string `json:"requiredScopes,omitempty" jsonschema:"optional"`

	// Resolved input schema for validation (internal use only).
	ResolvedInputSchema *jsonschema.Resolved `json:"-"`
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

// StreamableHTTPConfig defines configuration for the HTTP-based runtime.
type StreamableHTTPConfig struct {
	// Port number to listen on.
	Port int `json:"port" jsonschema:"required"`

	// Base path for the MCP server (default: /mcp).
	BasePath string `json:"basePath,omitempty" jsonschema:"optional"`

	// Indicates whether the server is stateless.
	Stateless bool `json:"stateless" jsonschema:"required"`

	// OAuth 2.0 configuration for protected resources.
	Auth *AuthConfig `json:"auth,omitempty" jsonschema:"optional"`

	// TLS configuration for HTTPS.
	TLS *TLSConfig `json:"tls,omitempty" jsonschema:"optional"`
}

// TLSConfig defines paths to TLS certificate and private key files.
type TLSConfig struct {
	// Absolute path to the server's public certificate.
	CertFile string `json:"certFile,omitempty" jsonschema:"optional"`

	// Absolute path to the server's private key.
	KeyFile string `json:"keyFile,omitempty" jsonschema:"optional"`
}

// AuthConfig defines OAuth 2.0 authorization settings.
type AuthConfig struct {
	// List of authorization server URLs for token validation.
	AuthorizationServers []string `json:"authorizationServers,omitempty" jsonschema:"optional"`

	// URI for the JSON Web Key Set (JWKS) used for token verification.
	JWKSURI string `json:"jwksUri,omitempty" jsonschema:"optional"`
}

// StdioConfig defines configuration for stdio transport protocol.
type StdioConfig struct{}

// ServerRuntime defines transport protocol and associated configuration.
type ServerRuntime struct {
	// Transport protocol to use (streamablehttp or stdio).
	TransportProtocol string `json:"transportProtocol" jsonschema:"required"`

	// Configuration for streamable HTTP transport protocol.
	StreamableHTTPConfig *StreamableHTTPConfig `json:"streamableHttpConfig,omitempty" jsonschema:"optional"`

	// Configuration for stdio transport protocol.
	StdioConfig *StdioConfig `json:"stdioConfig,omitempty" jsonschema:"optional"`

	// Configuration for the server logging
	LoggingConfig *logging.LoggingConfig `json:"loggingConfig" jsonschema:"optional"`

	baseLogger     *zap.Logger
	initLoggerOnce sync.Once
}

// GetBaseLogger returns the base logger for the server, defaulting to noop if either the runtime or
// any of the logging config is nil or has errors
func (sr *ServerRuntime) GetBaseLogger() *zap.Logger {
	if sr == nil {
		return zap.NewNop()
	}

	sr.initLoggerOnce.Do(func() {
		if sr.LoggingConfig != nil {
			logger, err := sr.LoggingConfig.BuildBase()
			if err != nil || logger == nil {
				// Surface the error to stderr before falling back
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: Failed to build base logger, using no-op logger: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "ERROR: BuildBase returned nil logger, using no-op logger\n")
				}
				logger = zap.NewNop()
			}
			sr.baseLogger = logger
		} else {
			sr.baseLogger = zap.NewNop()
		}
	})

	return sr.baseLogger
}

// MCPServer defines the metadata and capabilities of an MCP server.
type MCPServer struct {
	// Name of the MCP server.
	Name string `json:"name" jsonschema:"required"`

	// Semantic version of the MCP server.
	Version string `json:"version" jsonschema:"required"`

	// Runtime configuration for the MCP server.
	Runtime *ServerRuntime `json:"runtime,omitempty" jsonschema:"optional"`

	// List of tools provided by the server.
	Tools []*Tool `json:"tools,omitempty" jsonschema:"optional"`

	// List of prompts provided by the server.
	Prompts []*Prompt `json:"prompts,omitempty" jsonschema:"optional"`

	// Set of resources available to the server.
	Resources []*Resource `json:"resources,omitempty" jsonschema:"optional"`

	// Set of resource templates available to the server.
	ResourceTemplates []*ResourceTemplate `json:"resourceTemplates,omitempty" jsonschema:"optional"`
}

// MCPFile is the root structure of an MCP configuration file.
type MCPFile struct {
	// Version of the MCP file format.
	FileVersion string `json:"mcpFileVersion" jsonschema:"required"`

	// MCP server definition.
	MCPServer `json:",inline"`
}

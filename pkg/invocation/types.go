package invocation

import (
	"context"
	"encoding/json"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	JsonSchemaTypeObject  = "object"
	JsonSchemaTypeNumber  = "number"
	JsonSchemaTypeInteger = "integer"
	JsonSchemaTypeString  = "string"
	JsonSchemaTypeArray   = "array"
	JsonSchemaTypeBoolean = "boolean"
	JsonSchemaTypeNull    = "null"
)

type Invoker interface {
	Invoke(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	InvokePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
	InvokeResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
	InvokeResourceTemplate(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
}

type InvocationConfig interface {
	Validate() error
}

type InvocationConfigParser interface {
	Parse(data json.RawMessage, tool *mcpfile.Tool) (InvocationConfig, error)
	ParsePrompt(data json.RawMessage, prompt *mcpfile.Prompt) (InvocationConfig, error)
	ParseResource(data json.RawMessage, resource *mcpfile.Resource) (InvocationConfig, error)
	ParseResourceTemplate(data json.RawMessage, resourceTemplate *mcpfile.ResourceTemplate) (InvocationConfig, error)
}

type InvokerFactory interface {
	CreateInvoker(config InvocationConfig, schema *jsonschema.Resolved) (Invoker, error)
}

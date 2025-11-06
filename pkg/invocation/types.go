package invocation

import (
	"context"
	"encoding/json"
	"fmt"

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

type ConfigMetadata interface {
	NewConfig() InvocationConfig
}

type InvocationConfig interface {
	Validate() error
}

type Primitive interface {
	GetName() string
	GetDescription() string
	GetInputSchema() *jsonschema.Schema
	GetOutputSchema() *jsonschema.Schema
	GetInvocationConfig() InvocationConfig
	GetInvocationType() string
	GetRequiredScopes() []string
	GetResolvedInputSchema() *jsonschema.Resolved
	GetURITemplate() string
	PrimitiveType() string
}

type InvokerFactory interface {
	NewConfig() InvocationConfig
	CreateInvoker(config InvocationConfig, primitive Primitive) (Invoker, error)
}

type InvocationConfigWrapper struct {
	Type   string           `json:"-"`
	Config InvocationConfig `json:"-"`
}

func (w *InvocationConfigWrapper) UnmarshalJSON(data []byte) error {
	var typeMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &typeMap); err != nil {
		return fmt.Errorf("failed to unmarshal invocation config: %w", err)
	}

	if len(typeMap) != 1 {
		return fmt.Errorf("invocation config must have exactly one type key, got %d", len(typeMap))
	}

	for invocationType, configData := range typeMap {
		w.Type = invocationType

		factory, exists := globalRegistry.factories[invocationType]
		if !exists {
			return fmt.Errorf("unknown invocation type: '%s'", invocationType)
		}

		config := factory.NewConfig()

		if err := json.Unmarshal(configData, config); err != nil {
			return fmt.Errorf("failed to unmarshal %s config: %w", invocationType, err)
		}

		w.Config = config
		return nil
	}

	return fmt.Errorf("no invocation type found")
}

func (w InvocationConfigWrapper) MarshalJSON() ([]byte, error) {
	if w.Config == nil {
		return nil, fmt.Errorf("cannot marshal wrapper with nil config")
	}

	wrapper := map[string]InvocationConfig{
		w.Type: w.Config,
	}

	return json.Marshal(wrapper)
}

func (w InvocationConfigWrapper) GetType() string {
	return w.Type
}

func (w InvocationConfigWrapper) GetConfig() InvocationConfig {
	return w.Config
}

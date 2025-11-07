package invocation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	invopopschema "github.com/invopop/jsonschema"
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
	DeepCopy() InvocationConfig
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

// InvocationConfigWrapper wraps an invocation configuration with its type.
// In JSON, this is represented as an object with a single key indicating the type
// (one of "http", "cli", or "extends") and the value being the configuration.
// Example: {"http": {...}} or {"cli": {...}} or {"extends": {...}}
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

// JSONSchema provides a custom JSON Schema for InvocationConfigWrapper.
// This defines it as a oneOf discriminated union where exactly one invocation type key must be present.
func (w InvocationConfigWrapper) JSONSchema() *invopopschema.Schema {
	httpProps := invopopschema.NewProperties()
	httpProps.Set("http", &invopopschema.Schema{
		Ref: "#/$defs/HttpInvocationConfig",
	})

	cliProps := invopopschema.NewProperties()
	cliProps.Set("cli", &invopopschema.Schema{
		Ref: "#/$defs/CliInvocationConfig",
	})

	extendsProps := invopopschema.NewProperties()
	extendsProps.Set("extends", &invopopschema.Schema{
		Ref: "#/$defs/ExtendsConfig",
	})

	return &invopopschema.Schema{
		OneOf: []*invopopschema.Schema{
			{
				Type:                 "object",
				Properties:           httpProps,
				Required:             []string{"http"},
				AdditionalProperties: invopopschema.FalseSchema,
				Description:          "An invocation configuration using HTTP.",
			},
			{
				Type:                 "object",
				Properties:           cliProps,
				Required:             []string{"cli"},
				AdditionalProperties: invopopschema.FalseSchema,
				Description:          "An invocation configuration using CLI.",
			},
			{
				Type:                 "object",
				Properties:           extendsProps,
				Required:             []string{"extends"},
				AdditionalProperties: invopopschema.FalseSchema,
				Description:          "An invocation configuration that extends a base configuration.",
			},
		},
		Description: "A wrapper for invocation configurations. Must contain exactly one invocation type key (http, cli, or extends) with its corresponding configuration.",
	}
}

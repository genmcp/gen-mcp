package extends

import (
	"encoding/json"
	"fmt"

	"github.com/genmcp/gen-mcp/pkg/invocation"
)

const InvocationType = "extends"

// ExtendsConfig allows extending an invocation base with modifications.
type ExtendsConfig struct {
	// From specifies the invocation base to extend from.
	From string `json:"from" jsonschema:"required"`
	// Extend adds or merges new fields into the base configuration (e.g., append to arrays, merge maps).
	Extend json.RawMessage `json:"extend,omitempty" jsonschema:"optional"`
	// Override replaces fields in the base configuration with new values.
	Override json.RawMessage `json:"override,omitempty" jsonschema:"optional"`
	// Remove deletes specific fields from the base configuration (e.g., remove map keys, clear strings).
	Remove json.RawMessage `json:"remove,omitempty" jsonschema:"optional"`
}

func (ec *ExtendsConfig) Validate() error {
	if ec.From == "" {
		return fmt.Errorf("extends requires 'from' field")
	}

	return nil
}

func (ec *ExtendsConfig) DeepCopy() invocation.InvocationConfig {
	return &ExtendsConfig{
		From:     ec.From,
		Extend:   ec.Extend,
		Override: ec.Override,
		Remove:   ec.Remove,
	}
}

func (ec *ExtendsConfig) resolve() (*invocation.InvocationConfigWrapper, error) {
	baseInfo, ok := getBase(ec.From)
	if !ok {
		return nil, fmt.Errorf("failed to get base invocation config '%s'", ec.From)
	}

	factory, ok := invocation.GetFactory(baseInfo.Type)
	if !ok {
		return nil, fmt.Errorf("no matching invocation type found for invocation type '%s' from invocation base '%s'", baseInfo.Type, ec.From)
	}

	result := baseInfo.Config.DeepCopy()

	hasExtend := len(ec.Extend) > 0
	hasOverride := len(ec.Override) > 0
	hasRemove := len(ec.Remove) > 0

	if !hasExtend && !hasOverride && !hasRemove {
		return &invocation.InvocationConfigWrapper{
			Type:   baseInfo.Type,
			Config: result,
		}, nil
	}

	extendConfig := factory.NewConfig()
	overrideConfig := factory.NewConfig()
	removeConfig := factory.NewConfig()

	if hasExtend {
		if err := json.Unmarshal(ec.Extend, extendConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal extend: %w", err)
		}
	}

	if hasOverride {
		if err := json.Unmarshal(ec.Override, overrideConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal override: %w", err)
		}
	}

	if hasRemove {
		if err := unmarshalRemoveConfig(ec.Remove, removeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal remove: %w", err)
		}
	}

	if err := validateOperations(extendConfig, overrideConfig, removeConfig); err != nil {
		return nil, err
	}

	if hasRemove {
		if err := applyRemove(result, removeConfig); err != nil {
			return nil, fmt.Errorf("remove failed: %w", err)
		}
	}

	if hasExtend {
		if err := applyExtend(result, extendConfig); err != nil {
			return nil, fmt.Errorf("extend failed: %w", err)
		}
	}

	if hasOverride {
		if err := applyOverride(result, overrideConfig); err != nil {
			return nil, fmt.Errorf("override failed: %w", err)
		}
	}

	return &invocation.InvocationConfigWrapper{
		Type:   baseInfo.Type,
		Config: result,
	}, nil
}

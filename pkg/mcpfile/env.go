package mcpfile

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	genmcpEnvPrefix = "GENMCP"
)

type RuntimeOverrider interface {
	ApplyOverrides(runtime *ServerRuntime) error
}

type envRuntimeOverrider struct{}

func NewEnvRuntimeOverrider() RuntimeOverrider {
	return &envRuntimeOverrider{}
}

func (e *envRuntimeOverrider) ApplyOverrides(runtime *ServerRuntime) error {
	reflectRuntime := reflect.ValueOf(runtime).Elem()
	_, err := processStruct(reflectRuntime, genmcpEnvPrefix)
	return err
}

func processStruct(val reflect.Value, prefix string) (bool, error) {
	typ := val.Type()

	madeUpdate := false
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		if fieldVal.Kind() == reflect.Ptr && fieldVal.Elem().Kind() == reflect.Struct {
			wasNil := false
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				wasNil = true
			}

			updated, err := processStruct(fieldVal.Elem(), buildEnvKey(prefix, fieldTyp.Name))
			if updated {
				madeUpdate = updated
			} else if wasNil {
				// re-zero the type, we don't want to set anything to non nil if there was no override
				fieldVal.Set(reflect.Zero(fieldVal.Type()))
			}

			if err != nil {
				return madeUpdate, err
			}

		}

		if fieldVal.Kind() == reflect.Struct {
			// for embedded structs, the name is the type name. we don't want to add this to the prefix
			keyPrefix := prefix
			if !fieldTyp.Anonymous {
				keyPrefix = buildEnvKey(prefix, fieldTyp.Name)
			}
			updated, err := processStruct(fieldVal, keyPrefix)
			if updated {
				madeUpdate = updated
			}

			if err != nil {
				return madeUpdate, err
			}
		}

		envKey := buildEnvKey(prefix, fieldTyp.Name)
		envVal, found := os.LookupEnv(envKey)
		if !found {
			continue
		}

		if err := setField(fieldVal, envVal); err != nil {
			return madeUpdate, fmt.Errorf("error setting field %s from env var %s: %w", fieldTyp.Name, envKey, err)
		}

		madeUpdate = true
	}

	return madeUpdate, nil
}

func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// special case: time.Duration is an alias to int64
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			intVal, err := strconv.ParseInt(value, 10, getIntBitSize(field.Kind()))
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		}
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, getFloatBitSize(field.Kind()))
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(value, ",")
			field.Set(reflect.ValueOf(parts))
		} else {
			return fmt.Errorf("unsupported field type: %v", field.Kind())
		}
	case reflect.Ptr:
		// Handle pointers to basic types (e.g., *bool, *string, *int)
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setField(field.Elem(), value)
	case reflect.Map:
		// Handle maps, particularly map[string]interface{} by parsing as JSON
		if field.Type().Key().Kind() == reflect.String {
			var mapValue map[string]interface{}
			if err := json.Unmarshal([]byte(value), &mapValue); err != nil {
				return fmt.Errorf("failed to parse map value as JSON: %w", err)
			}
			field.Set(reflect.ValueOf(mapValue))
		} else {
			return fmt.Errorf("unsupported map key type: %v", field.Type().Key().Kind())
		}
	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}

func getIntBitSize(kind reflect.Kind) int {
	switch kind {
	case reflect.Int:
		return 0
	case reflect.Int8:
		return 8
	case reflect.Int16:
		return 16
	case reflect.Int32:
		return 32
	case reflect.Int64:
		return 64
	default:
		return -1
	}
}

func getFloatBitSize(kind reflect.Kind) int {
	switch kind {
	case reflect.Float32:
		return 32
	case reflect.Float64:
		return 64
	default:
		return -1
	}
}

func buildEnvKey(prefix, name string) string {
	if prefix == "" {
		return strings.ToUpper(name)
	}

	return strings.ToUpper(fmt.Sprintf("%s_%s", prefix, name))
}

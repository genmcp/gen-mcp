package extends

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// applyExtend extends base with values from ext using reflection
// strings are appended
// maps are merged
// slices are appended
func applyExtend(base, ext any) error {
	if base == nil || ext == nil {
		return fmt.Errorf("both base and ext must be non nil")
	}

	baseVal := reflect.ValueOf(base)
	extVal := reflect.ValueOf(ext)

	if baseVal.Kind() != reflect.Ptr || baseVal.IsNil() {
		return fmt.Errorf("base must be a non-nil pointer")
	}
	if extVal.Kind() != reflect.Ptr || extVal.IsNil() {
		return fmt.Errorf("ext must be a non-nil pointer")
	}

	baseVal = baseVal.Elem()
	extVal = extVal.Elem()
	baseType := baseVal.Type()
	extType := extVal.Type()

	if baseType != extType {
		return fmt.Errorf("both base and ext must be the same underlying type")
	}

	for i := 0; i < baseType.NumField(); i++ {
		field := baseType.Field(i)
		baseField := baseVal.Field(i)
		extField := extVal.Field(i)

		// skip as the intended extension is empty
		if extField.IsZero() {
			continue
		}

		switch baseField.Kind() {
		case reflect.String:
			// String: append
			baseStr := baseField.String()
			extStr := extField.String()
			baseField.SetString(baseStr + extStr)

		case reflect.Map:
			// Map: merge keys
			if baseField.IsNil() {
				baseField.Set(reflect.MakeMap(baseField.Type()))
			}
			iter := extField.MapRange()
			for iter.Next() {
				baseField.SetMapIndex(iter.Key(), iter.Value())
			}

		case reflect.Slice:
			// Slice: append items
			baseField.Set(reflect.AppendSlice(baseField, extField))

		default:
			return fmt.Errorf("field '%s' has unsupported type for extend: %s", field.Name, baseField.Kind())
		}
	}

	return nil
}

// applyOverride replaces base values with values from override using reflection
func applyOverride(base, override any) error {
	if base == nil || override == nil {
		return fmt.Errorf("both base and override must be non nil")
	}

	baseVal := reflect.ValueOf(base)
	overrideVal := reflect.ValueOf(override)

	if baseVal.Kind() != reflect.Ptr || baseVal.IsNil() {
		return fmt.Errorf("base must be a non-nil pointer")
	}
	if overrideVal.Kind() != reflect.Ptr || overrideVal.IsNil() {
		return fmt.Errorf("override must be a non-nil pointer")
	}

	baseVal = baseVal.Elem()
	overrideVal = overrideVal.Elem()
	baseType := baseVal.Type()
	overrideType := overrideVal.Type()

	if baseType != overrideType {
		return fmt.Errorf("both base and override must be the same underlying type")
	}

	for i := 0; i < baseType.NumField(); i++ {
		overrideField := overrideVal.Field(i)

		// skip as the intended override is empty - nothing to override
		if overrideField.IsZero() {
			continue
		}

		baseField := baseVal.Field(i)
		baseField.Set(overrideField)
	}

	return nil
}

// applyRemove removes specified items from base
// strings: sets to empty
// slices: removes matching values
// maps: removes any matching keys
func applyRemove(base, remove any) error {
	if base == nil || remove == nil {
		return fmt.Errorf("both base and remove must be non nil")
	}

	baseVal := reflect.ValueOf(base)
	removeVal := reflect.ValueOf(remove)

	if baseVal.Kind() != reflect.Ptr || baseVal.IsNil() {
		return fmt.Errorf("base must be a non-nil pointer")
	}
	if removeVal.Kind() != reflect.Ptr || removeVal.IsNil() {
		return fmt.Errorf("remove must be a non-nil pointer")
	}

	baseVal = baseVal.Elem()
	removeVal = removeVal.Elem()
	baseType := baseVal.Type()
	removeType := removeVal.Type()

	if baseType != removeType {
		return fmt.Errorf("both base and remove must be the same underlying type")
	}

	for i := 0; i < baseType.NumField(); i++ {
		field := baseType.Field(i)
		baseField := baseVal.Field(i)
		removeField := removeVal.Field(i)

		// skip if there is nothing to remove
		if removeField.IsZero() || baseField.IsZero() {
			continue
		}

		switch baseField.Kind() {
		case reflect.String:
			baseField.SetString("")

		case reflect.Map:
			if removeField.Kind() != reflect.Map {
				return fmt.Errorf("remove for map field '%s' must be a map of keys", field.Name)
			}

			iter := removeField.MapRange()
			for iter.Next() {
				baseField.SetMapIndex(iter.Key(), reflect.Value{})
			}

		case reflect.Slice:
			if removeField.Kind() != reflect.Slice {
				return fmt.Errorf("remove for slice field '%s' must be a slice", field.Name)
			}
			newSlice := reflect.MakeSlice(baseField.Type(), 0, baseField.Len())
			for j := 0; j < baseField.Len(); j++ {
				item := baseField.Index(j)
				shouldRemove := false
				for k := 0; k < removeField.Len(); k++ {
					if reflect.DeepEqual(item.Interface(), removeField.Index(k).Interface()) {
						shouldRemove = true
						break
					}
				}

				if !shouldRemove {
					newSlice = reflect.Append(newSlice, item)
				}
			}
			baseField.Set(newSlice)

		default:
			return fmt.Errorf("field '%s' has unsupported type for remove: %s", field.Name, baseField.Kind())
		}

	}

	return nil
}

func validateOperations(extend, override, remove any) error {
	if extend == nil || override == nil || remove == nil {
		return fmt.Errorf("extend, override, and remove must be non nil")
	}

	extendVal := reflect.ValueOf(extend)
	overrideVal := reflect.ValueOf(override)
	removeVal := reflect.ValueOf(remove)

	if extendVal.Kind() != reflect.Ptr || extendVal.IsNil() {
		return fmt.Errorf("extend must be a non-nil pointer")
	}
	if overrideVal.Kind() != reflect.Ptr || overrideVal.IsNil() {
		return fmt.Errorf("override must be a non-nil pointer")
	}
	if removeVal.Kind() != reflect.Ptr || removeVal.IsNil() {
		return fmt.Errorf("remove must be a non-nil pointer")
	}

	extendVal = extendVal.Elem()
	overrideVal = overrideVal.Elem()
	removeVal = removeVal.Elem()
	extendType := extendVal.Type()
	overrideType := overrideVal.Type()
	removeType := removeVal.Type()

	if extendType != overrideType || extendType != removeType {
		return fmt.Errorf("extend, override, and remove must be the same underlying type")
	}

	for i := 0; i < extendType.NumField(); i++ {
		field := extendType.Field(i)
		hasExtend := !extendVal.Field(i).IsZero()
		hasOverride := !overrideVal.Field(i).IsZero()
		hasRemove := !removeVal.Field(i).IsZero()

		ops := []string{}
		if hasExtend {
			ops = append(ops, "extend")
		}
		if hasOverride {
			ops = append(ops, "override")
		}
		if hasRemove {
			ops = append(ops, "remove")
		}

		if len(ops) > 1 {
			return fmt.Errorf("cannot use multiple operations on field '%s': %s", field.Name, strings.Join(ops, ", "))
		}
	}

	return nil
}

// UnmarshalRemoveConfig handles special unmarshalling for remove operations
// For map fields, it accepts []string and converts to appropriate format
func unmarshalRemoveConfig(data json.RawMessage, config any) error {
	if config == nil {
		return fmt.Errorf("config must be non nil")
	}

	configVal := reflect.ValueOf(config)

	if configVal.Kind() != reflect.Ptr || configVal.IsNil() {
		return fmt.Errorf("config must be a non-nil pointer")
	}

	configVal = configVal.Elem()
	configType := configVal.Type()

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		configField := configVal.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// extract just the field name from tag (before comma)
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		rawData, ok := rawMap[jsonTag]
		if !ok {
			continue
		}

		// if we have a map, first to try unmarshal as a slice
		// this allows users to provide a slice of keys to remove, which is more intuitive
		if configField.Kind() == reflect.Map {
			mapType := configField.Type()
			keyType := mapType.Key()
			elemType := mapType.Elem()

			sliceType := reflect.SliceOf(keyType)
			keysPtr := reflect.New(sliceType)
			if err := json.Unmarshal(rawData, keysPtr.Interface()); err == nil {
				keysSlice := keysPtr.Elem()

				newMap := reflect.MakeMap(mapType)

				elemZeroVal := reflect.Zero(elemType)

				for j := 0; j < keysSlice.Len(); j++ {
					key := keysSlice.Index(j)
					newMap.SetMapIndex(key, elemZeroVal)
				}

				configField.Set(newMap)
				continue
			}
		}

		if err := json.Unmarshal(rawData, configField.Addr().Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal remove field '%s': %w", field.Name, err)
		}
	}

	return nil
}

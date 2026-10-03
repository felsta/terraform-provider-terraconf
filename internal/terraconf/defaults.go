package terraconf

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ApplyDefaults walks the given value alongside the compiled schema and, for
// every object, adds the properties that are missing but have a default value
// defined in the schema. The value is modified in place. References ($ref,
// $recursiveRef, $dynamicRef) and allOf are followed, since they always apply
// to the value. Conditional keywords (anyOf, oneOf, if/then/else) and
// patternProperties are ignored, since it is ambiguous which of their
// subschemas should provide the default. The value should be validated
// against the schema before applying defaults.
func ApplyDefaults(schema *jsonschema.Schema, value any) error {
	schemas := collectApplicableSchemas(schema)

	switch value := value.(type) {
	case map[string]any:
		for _, s := range schemas {
			for key, propertySchema := range s.Properties {
				if _, ok := value[key]; !ok {
					defaultValue, ok, err := findDefault(propertySchema)
					if err != nil {
						return fmt.Errorf("default for property %s: %w", key, err)
					}
					if !ok {
						continue
					}

					value[key] = defaultValue
				}

				if err := ApplyDefaults(propertySchema, value[key]); err != nil {
					return err
				}
			}
		}
	case []any:
		for _, s := range schemas {
			for i, item := range value {
				itemSchema := findItemSchema(s, i)
				if itemSchema == nil {
					continue
				}

				if err := ApplyDefaults(itemSchema, item); err != nil {
					return err
				}
			}
		}
	default:
		// primitives: nothing to do
	}

	return nil
}

// collectApplicableSchemas returns the given schema along with all schemas
// that unconditionally apply to the same value, which are the targets of
// references and the subschemas of allOf. Each schema is only returned once,
// so recursive schemas don't cause an infinite loop.
func collectApplicableSchemas(schema *jsonschema.Schema) []*jsonschema.Schema {
	var schemas []*jsonschema.Schema
	visited := map[*jsonschema.Schema]bool{}

	var visit func(s *jsonschema.Schema)
	visit = func(s *jsonschema.Schema) {
		if s == nil || visited[s] {
			return
		}
		visited[s] = true
		schemas = append(schemas, s)

		visit(s.Ref)
		visit(s.RecursiveRef)
		if s.DynamicRef != nil {
			visit(s.DynamicRef.Ref)
		}
		for _, sub := range s.AllOf {
			visit(sub)
		}
	}

	visit(schema)
	return schemas
}

// findDefault returns a copy of the first default value found in the
// given schema or the schemas that apply alongside it. The second return
// value reports whether a default was found.
func findDefault(schema *jsonschema.Schema) (any, bool, error) {
	for _, s := range collectApplicableSchemas(schema) {
		if s.Default == nil {
			continue
		}

		value, err := copyDefault(*s.Default)
		if err != nil {
			return nil, false, err
		}

		return value, true, nil
	}

	return nil, false, nil
}

// findItemSchema returns the schema that applies to the array item at the
// given index, or nil if there is none. Both the draft 2020-12 keywords
// (prefixItems, items) and the keywords of older drafts (items as array or
// schema, additionalItems) are supported.
func findItemSchema(schema *jsonschema.Schema, index int) *jsonschema.Schema {
	if index < len(schema.PrefixItems) {
		return schema.PrefixItems[index]
	}
	if schema.Items2020 != nil {
		return schema.Items2020
	}

	switch items := schema.Items.(type) {
	case *jsonschema.Schema:
		return items
	case []*jsonschema.Schema:
		if index < len(items) {
			return items[index]
		}
		if additionalItems, ok := schema.AdditionalItems.(*jsonschema.Schema); ok {
			return additionalItems
		}
	}

	return nil
}

// copyDefault deep copies a default value from the schema, so that values
// added to different documents don't share state with each other or the
// schema. Numbers are decoded as json.Number by the schema loader and are
// converted to int64 or float64, which are supported by the rest of the
// provider.
func copyDefault(value any) (any, error) {
	switch value := value.(type) {
	case nil, bool, string:
		return value, nil
	case json.Number:
		if i, err := value.Int64(); err == nil {
			return i, nil
		}

		f, err := value.Float64()
		if err != nil {
			return nil, fmt.Errorf("converting number %s: %w", value, err)
		}

		return f, nil
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			copied, err := copyDefault(item)
			if err != nil {
				return nil, err
			}

			result[i] = copied
		}

		return result, nil
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			copied, err := copyDefault(item)
			if err != nil {
				return nil, err
			}

			result[key] = copied
		}

		return result, nil
	default:
		return nil, fmt.Errorf("unsupported default value type: %T", value)
	}
}

package terraconf

import (
	"path/filepath"
	"testing"
)

func TestApplyDefaults(t *testing.T) {
	cases := []struct {
		name       string
		schemaFile string
		input      map[string]any
		expected   map[string]any
	}{
		{
			name:       "empty object gets all top-level defaults",
			schemaFile: "schema.json",
			input:      map[string]any{},
			expected: map[string]any{
				"enabled":  false,
				"replicas": 3,
				"ratio":    0.5,
				"tags":     []any{},
				"network":  map[string]any{"mtu": 1500},
				"owner":    "platform",
			},
		},
		{
			name:       "existing values are not overwritten",
			schemaFile: "schema.json",
			input: map[string]any{
				"name":     "shared",
				"enabled":  true,
				"replicas": 1,
				"ratio":    1.5,
				"tags":     []any{"foo"},
				"network":  map[string]any{"mtu": 9000},
				"owner":    "me",
			},
			expected: map[string]any{
				"name":     "shared",
				"enabled":  true,
				"replicas": 1,
				"ratio":    1.5,
				"tags":     []any{"foo"},
				"network":  map[string]any{"mtu": 9000},
				"owner":    "me",
			},
		},
		{
			name:       "nested object without default only gets defaults if present",
			schemaFile: "schema.json",
			input: map[string]any{
				"labels": map[string]any{},
			},
			expected: map[string]any{
				"enabled":  false,
				"replicas": 3,
				"ratio":    0.5,
				"tags":     []any{},
				"network":  map[string]any{"mtu": 1500},
				"owner":    "platform",
				"labels":   map[string]any{"env": "dev"},
			},
		},
		{
			name:       "array items via items and prefixItems",
			schemaFile: "schema.json",
			input: map[string]any{
				"disks": []any{map[string]any{}, map[string]any{"size": 20}},
				"tuple": []any{map[string]any{}, map[string]any{}},
			},
			expected: map[string]any{
				"enabled":  false,
				"replicas": 3,
				"ratio":    0.5,
				"tags":     []any{},
				"network":  map[string]any{"mtu": 1500},
				"owner":    "platform",
				"disks":    []any{map[string]any{"size": 10}, map[string]any{"size": 20}},
				"tuple":    []any{map[string]any{"first": "a"}, map[string]any{}},
			},
		},
		{
			name:       "recursive schema",
			schemaFile: "schema.json",
			input: map[string]any{
				"tree": map[string]any{
					"children": []any{
						map[string]any{},
						map[string]any{"kind": "branch", "children": []any{map[string]any{}}},
					},
				},
			},
			expected: map[string]any{
				"enabled":  false,
				"replicas": 3,
				"ratio":    0.5,
				"tags":     []any{},
				"network":  map[string]any{"mtu": 1500},
				"owner":    "platform",
				"tree": map[string]any{
					"kind": "leaf",
					"children": []any{
						map[string]any{"kind": "leaf"},
						map[string]any{"kind": "branch", "children": []any{map[string]any{"kind": "leaf"}}},
					},
				},
			},
		},
		{
			name:       "draft-07 items and additionalItems",
			schemaFile: "draft7.json",
			input: map[string]any{
				"list":  []any{map[string]any{}, map[string]any{"a": 2}},
				"tuple": []any{map[string]any{}, map[string]any{}},
			},
			expected: map[string]any{
				"list":  []any{map[string]any{"a": 1}, map[string]any{"a": 2}},
				"tuple": []any{map[string]any{"first": "x"}, map[string]any{"rest": "y"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := CompileSchema(filepath.Join("testdata", "schema-defaults", tc.schemaFile))
			if err != nil {
				t.Fatalf("CompileSchema: %v", err)
			}

			if err := ApplyDefaults(schema, tc.input); err != nil {
				t.Fatalf("ApplyDefaults: %v", err)
			}

			if err := compareValues(tc.expected, tc.input, NewJsonPointerRoot()); err != nil {
				t.Fatalf("mismatch: %v", err)
			}
		})
	}
}

func TestApplyDefaultsNumberTypes(t *testing.T) {
	schema, err := CompileSchema(filepath.Join("testdata", "schema-defaults", "schema.json"))
	if err != nil {
		t.Fatalf("CompileSchema: %v", err)
	}

	value := map[string]any{}
	if err := ApplyDefaults(schema, value); err != nil {
		t.Fatalf("ApplyDefaults: %v", err)
	}

	if _, ok := value["replicas"].(int64); !ok {
		t.Errorf("expected integer default to be int64, got %T", value["replicas"])
	}
	if _, ok := value["ratio"].(float64); !ok {
		t.Errorf("expected number default to be float64, got %T", value["ratio"])
	}
}

func TestApplyDefaultsNoSharedState(t *testing.T) {
	schema, err := CompileSchema(filepath.Join("testdata", "schema-defaults", "schema.json"))
	if err != nil {
		t.Fatalf("CompileSchema: %v", err)
	}

	first := map[string]any{}
	if err := ApplyDefaults(schema, first); err != nil {
		t.Fatalf("ApplyDefaults: %v", err)
	}

	// Mutate the defaults added to the first value, like AddAnnotations does
	network, ok := first["network"].(map[string]any)
	if !ok {
		t.Fatalf("expected network default to be an object, got %T", first["network"])
	}
	network["_terraconf"] = "mutated"

	second := map[string]any{}
	if err := ApplyDefaults(schema, second); err != nil {
		t.Fatalf("ApplyDefaults: %v", err)
	}

	expected := map[string]any{"mtu": 1500}
	if err := compareValues(expected, second["network"], NewJsonPointer("/network")); err != nil {
		t.Fatalf("default was shared between values: %v", err)
	}
}

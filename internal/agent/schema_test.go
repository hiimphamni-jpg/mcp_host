package agent

import (
	"strings"
	"testing"
)

func TestValidateArgs_ok(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"mode":    map[string]any{"type": "string", "enum": []any{"read", "write"}},
			"count":   map[string]any{"type": "integer"},
			"ratio":   map[string]any{"type": "number"},
			"enabled": map[string]any{"type": "boolean"},
			"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"config": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"retries": map[string]any{"type": "integer"},
				},
			},
		},
		"required": []string{"path", "config"},
	}

	cases := []map[string]any{
		{"path": "a.txt", "config": map[string]any{}},
		{"path": "a.txt", "config": map[string]any{"retries": 3}, "mode": "read", "count": 2, "ratio": 0.5, "enabled": true, "tags": []any{"x", "y"}},
		{"path": "/x", "config": map[string]any{"retries": 1}, "extra": "allowed"},
	}
	for _, args := range cases {
		if err := validateArgs(schema, args); err != nil {
			t.Errorf("validateArgs(%v) returned an error: %v", args, err)
		}
	}
}

func TestValidateArgs_missingRequired(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"path": map[string]any{"type": "string"}},
		"required":   []string{"path"},
	}
	err := validateArgs(schema, map[string]any{})
	if err == nil {
		t.Fatal("validateArgs accepted missing required field")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Errorf("error %q does not mention the missing field", err.Error())
	}
}

func TestValidateArgs_wrongType(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"s": map[string]any{"type": "string"},
			"i": map[string]any{"type": "integer"},
			"b": map[string]any{"type": "boolean"},
		},
	}
	cases := []map[string]any{
		{"s": 42},
		{"i": "not-an-int"},
		{"i": 1.5},
		{"b": "yes"},
	}
	for _, args := range cases {
		if err := validateArgs(schema, args); err == nil {
			t.Errorf("validateArgs(%v) accepted a wrong type", args)
		}
	}
}

func TestValidateArgs_enumMismatch(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"mode": map[string]any{"type": "string", "enum": []any{"fast", "slow"}}},
		"required":   []string{"mode"},
	}
	if err := validateArgs(schema, map[string]any{"mode": "medium"}); err == nil {
		t.Fatal("validateArgs accepted an out-of-enum value")
	}
	if err := validateArgs(schema, map[string]any{"mode": "fast"}); err != nil {
		t.Fatalf("validateArgs rejected an in-enum value: %v", err)
	}
}

func TestValidateArgs_nestedObject(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"config": map[string]any{
				"type":       "object",
				"properties": map[string]any{"timeout": map[string]any{"type": "number"}},
				"required":   []string{"timeout"},
			},
		},
	}
	if err := validateArgs(schema, map[string]any{"config": map[string]any{"timeout": 1.5}}); err != nil {
		t.Errorf("validateArgs rejected a valid nested object: %v", err)
	}
	if err := validateArgs(schema, map[string]any{"config": map[string]any{}}); err == nil {
		t.Error("validateArgs accepted a nested object missing its required field")
	}
	if err := validateArgs(schema, map[string]any{"config": "not-an-object"}); err == nil {
		t.Error("validateArgs accepted a non-object for an object property")
	}
}

func TestValidateArgs_arrayItems(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
	if err := validateArgs(schema, map[string]any{"tags": []any{"a", "b"}}); err != nil {
		t.Errorf("validateArgs rejected a valid array: %v", err)
	}
	if err := validateArgs(schema, map[string]any{"tags": []any{"a", 42}}); err == nil {
		t.Error("validateArgs accepted an array with a bad item")
	}
	if err := validateArgs(schema, map[string]any{"tags": "not-array"}); err == nil {
		t.Error("validateArgs accepted a non-array for an array property")
	}
}

func TestValidateArgs_emptySchemaNoArgs(t *testing.T) {
	if err := validateArgs(map[string]any{}, nil); err != nil {
		t.Errorf("validateArgs(nil) returned an error: %v", err)
	}
	if err := validateArgs(map[string]any{}, map[string]any{"any": "thing"}); err != nil {
		t.Errorf("validateArgs({}) returned an error: %v", err)
	}
	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	if err := validateArgs(schema, map[string]any{"anything": true}); err != nil {
		t.Errorf("validateArgs(no-props) returned an error: %v", err)
	}
}

func TestValidateArgs_integerEnumMatchesFloat(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"code": map[string]any{"type": "integer", "enum": []any{200, 404}}},
		"required":   []string{"code"},
	}
	// Gemini may decode 200 as float64(200) in args.
	if err := validateArgs(schema, map[string]any{"code": float64(200)}); err != nil {
		t.Errorf("validateArgs rejected integer enum as float64: %v", err)
	}
}

func TestValidateArgs_requiredAsJSONDecodedArray(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"path": map[string]any{"type": "string"}},
		"required":   []any{"path"},
	}
	if err := validateArgs(schema, map[string]any{}); err == nil {
		t.Error("validateArgs accepted missing required field with []any required list")
	}
	if err := validateArgs(schema, map[string]any{"path": "a"}); err != nil {
		t.Errorf("validateArgs rejected present required field: %v", err)
	}
}

func TestValidateArgs_enumAsStringSlice(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"mode": map[string]any{"type": "string", "enum": []string{"fast", "slow"}}},
		"required":   []string{"mode"},
	}
	if err := validateArgs(schema, map[string]any{"mode": "fast"}); err != nil {
		t.Errorf("validateArgs rejected a []string enum member: %v", err)
	}
	if err := validateArgs(schema, map[string]any{"mode": "nope"}); err == nil {
		t.Error("validateArgs accepted a value outside a []string enum")
	}
}

func TestValidateArgs_numberNumericKinds(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"n": map[string]any{"type": "number"}, "i": map[string]any{"type": "integer"}},
	}
	// JSON decodes all numbers as float64; integer args must also accept
	// int-family values (llm.ToFloat covers them).
	for _, n := range []any{int(1), int64(1), float64(1), float32(1), uint(1), int32(1)} {
		if err := validateArgs(schema, map[string]any{"n": n, "i": n}); err != nil {
			t.Errorf("validateArgs rejected numeric kind %T: %v", n, err)
		}
	}
	if err := validateArgs(schema, map[string]any{"n": "x"}); err == nil {
		t.Error("validateArgs accepted a string for a number property")
	}
}

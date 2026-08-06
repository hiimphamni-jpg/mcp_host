package agent

import (
	"fmt"
	"math"
	"reflect"
)

// This file holds the stdlib-only argument validator used by the agent loop
// (business-logic §5: "Tool arguments must conform to the discovered schema
// before they are sent to MCP"). It validates against the neutral schema
// registry passed via Options.Schemas (ADR-0001 D1) so internal/agent stays
// free of mcp-go and genai. It deliberately mirrors the schema subset that
// internal/mapping supports (object/nested object, string/number/integer/
// boolean/array, enum, required, array items) without importing it.

// validateArgs checks args against a tool's neutral JSON Schema object.
// An empty or untyped schema is treated as unconstrained (no-arg tools pass).
// Unknown argument keys are allowed; only declared properties are validated.
func validateArgs(schema map[string]any, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	if typ, _ := schema["type"].(string); typ == "" {
		return nil
	}
	return validateObject(schema, args, "")
}

// validateObject checks object properties against required and property
// schemas. path is a dot-delimited field path used in error messages; "" is
// the tool root.
func validateObject(schema map[string]any, value map[string]any, path string) error {
	if reqRaw, ok := schema["required"]; ok {
		req := toStrings(reqRaw)
		for _, name := range req {
			if _, present := value[name]; !present {
				return fmt.Errorf("%s: required field %q is missing", displayPath(path), name)
			}
		}
	}

	props, _ := schema["properties"].(map[string]any)
	for name, raw := range value {
		prop, ok := props[name]
		if !ok {
			continue // undeclared keys are allowed (lenient)
		}
		propSchema, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		if err := validateValue(propSchema, raw, joinPath(path, name)); err != nil {
			return err
		}
	}
	return nil
}

// validateValue checks a single value against a property schema, recursing
// into nested objects and array items. path carries the exact field location.
func validateValue(schema map[string]any, value any, path string) error {
	typ, _ := schema["type"].(string)
	switch typ {
	case "string":
		if _, ok := value.(string); !ok {
			return typeMismatch(path, "string", value)
		}
		return validateEnum(schema, value, path)
	case "number":
		if !isNumber(value) {
			return typeMismatch(path, "number", value)
		}
		return validateEnum(schema, value, path)
	case "integer":
		if !isInteger(value) {
			return typeMismatch(path, "integer", value)
		}
		return validateEnum(schema, value, path)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return typeMismatch(path, "boolean", value)
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return typeMismatch(path, "array", value)
		}
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range arr {
				if err := validateValue(items, item, fmt.Sprintf("%s[%d]", displayPath(path), i)); err != nil {
					return err
				}
			}
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return typeMismatch(path, "object", value)
		}
		return validateObject(schema, obj, path)
	}
	return nil
}

// validateEnum rejects a value that is not among the declared string/number
// members. The message never embeds the value so a potentially sensitive
// argument cannot leak into a tool result (rules/security.md).
func validateEnum(schema map[string]any, value any, path string) error {
	raw, ok := schema["enum"]
	if !ok {
		return nil
	}
	for _, member := range toSlice(raw) {
		if looseEqual(member, value) {
			return nil
		}
	}
	return fmt.Errorf("%s: value is not in the allowed set", displayPath(path))
}

func typeMismatch(path, want string, got any) error {
	return fmt.Errorf("%s: expected %s, got %s", displayPath(path), want, reflect.TypeOf(got))
}

func displayPath(path string) string {
	if path == "" {
		return "$"
	}
	return path
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

// toStrings normalizes a required list, which may be []string or a
// JSON-decoded []any of strings, dropping non-strings defensively.
func toStrings(raw any) []string {
	var out []string
	switch v := raw.(type) {
	case []string:
		out = append(out, v...)
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// toSlice normalizes an enum list, which may be []string or []any.
func toSlice(raw any) []any {
	switch v := raw.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i, s := range v {
			out[i] = s
		}
		return out
	}
	return nil
}

// looseEqual compares JSON-decoded values, treating numbers across Go numeric
// types as equal so integer enum members match float64 arguments.
func looseEqual(a, b any) bool {
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func isNumber(v any) bool {
	_, ok := toFloat(v)
	return ok
}

func isInteger(v any) bool {
	f, ok := toFloat(v)
	if !ok {
		return false
	}
	return f == math.Trunc(f)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

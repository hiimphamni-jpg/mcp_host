package mapping

import (
	"google.golang.org/genai"
)

// maxRecursionDepth bounds recursive descent into nested objects and array
// items. It exists so a malformed cyclic schema (which can be constructed
// from in-memory map[string]any values) is rejected with an error instead of
// overflowing the stack (TDD §6).
const maxRecursionDepth = 64

// mapType converts a JSON Schema type string from the supported MVP subset to
// a genai.Type. Anything else (including null, any, or an empty/missing type)
// is rejected with a path-bearing ErrUnsupportedType.
func mapType(tool, path string, raw any) (genai.Type, error) {
	t, ok := raw.(string)
	if !ok {
		return genai.TypeUnspecified, malformed(tool, path, "type must be a string")
	}
	switch t {
	case "object":
		return genai.TypeObject, nil
	case "string":
		return genai.TypeString, nil
	case "number":
		return genai.TypeNumber, nil
	case "integer":
		return genai.TypeInteger, nil
	case "boolean":
		return genai.TypeBoolean, nil
	case "array":
		return genai.TypeArray, nil
	default:
		return genai.TypeUnspecified, unsupportedType(tool, path, t)
	}
}

// mapSchemaValue converts one property schema (a map[string]any) into a
// *genai.Schema, recursing into object properties and array items. depth is
// incremented on every descent and checked against maxRecursionDepth.
func mapSchemaValue(tool, path string, m map[string]any, depth int) (*genai.Schema, error) {
	if depth > maxRecursionDepth {
		return nil, malformed(tool, path, "schema nesting exceeds the recursion limit")
	}

	typ, err := mapType(tool, path, m["type"])
	if err != nil {
		return nil, err
	}
	s := &genai.Schema{Type: typ}

	if desc, ok := m["description"].(string); ok && desc != "" {
		s.Description = desc
	}

	switch typ {
	case genai.TypeObject:
		if err := mapObject(tool, path, m, s, depth); err != nil {
			return nil, err
		}
	case genai.TypeArray:
		if err := mapArray(tool, path, m, s, depth); err != nil {
			return nil, err
		}
	default:
		if raw, ok := m["enum"]; ok {
			es, err := enumStrings(tool, path, raw)
			if err != nil {
				return nil, err
			}
			s.Enum = es
		}
		if err := mapNumericBounds(tool, path, m, s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// mapObject fills an object schema with its Properties and nested Required.
// The root tool parameters schema is built through mapSchemaForTool, which
// calls mapProperty for each top-level property.
func mapObject(tool, path string, m map[string]any, s *genai.Schema, depth int) error {
	if rawProps, ok := m["properties"]; ok {
		props, ok := rawProps.(map[string]any)
		if !ok {
			return malformed(tool, path, "properties must be an object")
		}
		s.Properties = make(map[string]*genai.Schema, len(props))
		for name, raw := range props {
			childPath := joinPath(path, name)
			child, err := mapProperty(tool, raw, childPath, depth+1)
			if err != nil {
				return err
			}
			s.Properties[name] = child
		}
	}
	if rawReq, ok := m["required"]; ok {
		req, err := requiredStrings(tool, path, rawReq)
		if err != nil {
			return err
		}
		s.Required = req
	}
	return nil
}

// mapArray fills an array schema with its Items, if declared. The item path
// uses the "[]" suffix so a rejection points at the exact array member field.
func mapArray(tool, path string, m map[string]any, s *genai.Schema, depth int) error {
	rawItems, ok := m["items"]
	if !ok {
		return nil
	}
	itemPath := path + "[]"
	item, err := mapProperty(tool, rawItems, itemPath, depth+1)
	if err != nil {
		return err
	}
	s.Items = item
	return nil
}

// mapProperty coerces a property value into a schema map. Every property value
// in mcp-go Tool schemas is itself a map[string]any; anything else is malformed.
func mapProperty(tool string, raw any, path string, depth int) (*genai.Schema, error) {
	if depth > maxRecursionDepth {
		return nil, malformed(tool, path, "schema nesting exceeds the recursion limit")
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, malformed(tool, path, "property schema must be an object")
	}
	return mapSchemaValue(tool, path, m, depth)
}

// requiredStrings accepts either a []string or a JSON-decoded []any of strings
// (nested required inside a map[string]any property decodes to an interface
// slice) and rejects non-string entries with path context.
func requiredStrings(tool, path string, raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...), nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			sv, ok := item.(string)
			if !ok {
				return nil, malformed(tool, path, "required must be an array of strings")
			}
			out = append(out, sv)
		}
		return out, nil
	default:
		return nil, malformed(tool, path, "required must be an array of strings")
	}
}

// enumStrings accepts either a []string or a JSON-decoded []any of strings and
// rejects non-stringable enum members with path context (plan risk: enum as
// []any vs []string).
func enumStrings(tool, path string, raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...), nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			sv, ok := item.(string)
			if !ok {
				return nil, malformed(tool, path, "enum members must be strings")
			}
			out = append(out, sv)
		}
		return out, nil
	default:
		return nil, malformed(tool, path, "enum must be an array of strings")
	}
}

// mapNumericBounds carries minimum/maximum numeric constraints (mcp-go Min/Max
// helpers emit "minimum"/"maximum" keys as int/int64/float64) into the schema.
func mapNumericBounds(tool, path string, m map[string]any, s *genai.Schema) error {
	if minRaw, ok := m["minimum"]; ok {
		min, ok := asFloat64(minRaw)
		if !ok {
			return malformed(tool, path, "minimum must be numeric")
		}
		s.Minimum = &min
	}
	if maxRaw, ok := m["maximum"]; ok {
		max, ok := asFloat64(maxRaw)
		if !ok {
			return malformed(tool, path, "maximum must be numeric")
		}
		s.Maximum = &max
	}
	return nil
}

func asFloat64(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

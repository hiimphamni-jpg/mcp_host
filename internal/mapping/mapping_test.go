package mapping_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"

	"mcp-host/internal/mapping"
)

func tool(name, desc string, props map[string]any, required []string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Description: desc,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: props,
			Required:   required,
		},
	}
}

func TestMapTools_FullSupportedSubsetLossless(t *testing.T) {
	props := map[string]any{
		"path":    map[string]any{"type": "string", "description": "file name"},
		"mode":    map[string]any{"type": "string", "enum": []any{"append", "write", "read"}},
		"modeS":   map[string]any{"type": "string", "enum": []string{"auto", "manual"}},
		"count":   map[string]any{"type": "integer"},
		"ratio":   map[string]any{"type": "number"},
		"enabled": map[string]any{"type": "boolean"},
		"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"config": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timeout": map[string]any{"type": "number"},
				"retries": map[string]any{"type": "integer"},
			},
			"required": []any{"timeout"},
		},
	}
	tools := []mcp.Tool{tool("file_ops", "operate on files", props, []string{"path", "count", "config"})}

	decls, err := mapping.MapTools(tools)
	if err != nil {
		t.Fatalf("MapTools returned an error: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("MapTools returned %d declarations, want 1", len(decls))
	}

	want := &genai.FunctionDeclaration{
		Name:        "file_ops",
		Description: "operate on files",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"path":    {Type: genai.TypeString, Description: "file name"},
				"mode":    {Type: genai.TypeString, Enum: []string{"append", "write", "read"}},
				"modeS":   {Type: genai.TypeString, Enum: []string{"auto", "manual"}},
				"count":   {Type: genai.TypeInteger},
				"ratio":   {Type: genai.TypeNumber},
				"enabled": {Type: genai.TypeBoolean},
				"tags":    {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
				"config": {
					Type:     genai.TypeObject,
					Required: []string{"timeout"},
					Properties: map[string]*genai.Schema{
						"timeout": {Type: genai.TypeNumber},
						"retries": {Type: genai.TypeInteger},
					},
				},
			},
			Required: []string{"path", "count", "config"},
		},
	}

	if !reflect.DeepEqual(decls[0], want) {
		t.Errorf("declaration mismatch:\n got %+v\nwant %+v", decls[0], want)
	}
}

func TestMapTools_emptyPropertiesNoArgTool(t *testing.T) {
	tools := []mcp.Tool{tool("ping", "no args", map[string]any{}, nil)}

	decls, err := mapping.MapTools(tools)
	if err != nil {
		t.Fatalf("MapTools returned an error: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("MapTools returned %d declarations, want 1", len(decls))
	}
	if decls[0].Parameters == nil || decls[0].Parameters.Properties == nil || len(decls[0].Parameters.Properties) != 0 {
		t.Errorf("expected empty non-nil Parameters.Properties, got %+v", decls[0].Parameters)
	}
}

func TestMapTools_rejectsUnsupportedType(t *testing.T) {
	cases := []struct {
		name     string
		props    map[string]any
		wantSub  string
		wantKind error
	}{
		{"null type", map[string]any{"bad": map[string]any{"type": "null"}}, "bad", mapping.ErrUnsupportedType},
		{"any type", map[string]any{"bad": map[string]any{"type": "any"}}, "bad", mapping.ErrUnsupportedType},
		{"empty type", map[string]any{"bad": map[string]any{"type": ""}}, "bad", mapping.ErrUnsupportedType},
		{"missing type", map[string]any{"bad": map[string]any{"description": "x"}}, "bad", mapping.ErrMalformed},
		{"unknown type", map[string]any{"bad": map[string]any{"type": "undefined"}}, "bad", mapping.ErrUnsupportedType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tools := []mcp.Tool{tool("my_tool", "", tc.props, nil)}
			_, err := mapping.MapTools(tools)
			if err == nil {
				t.Fatal("MapTools returned nil error, want unsupported-type error")
			}
			if !errors.Is(err, tc.wantKind) {
				t.Errorf("errors.Is(err, %v) = false, err=%v", tc.wantKind, err)
			}
			for _, sub := range []string{"my_tool", tc.wantSub} {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q does not contain %q", err.Error(), sub)
				}
			}
		})
	}
}

func TestMapTools_rejectsNestedUnsupportedWithPath(t *testing.T) {
	props := map[string]any{
		"config": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"inner": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"x": map[string]any{"type": "void"},
					},
				},
			},
		},
	}
	tools := []mcp.Tool{tool("myTool", "", props, nil)}
	_, err := mapping.MapTools(tools)
	if err == nil {
		t.Fatal("MapTools returned nil error, want nested unsupported-type error")
	}
	if !errors.Is(err, mapping.ErrUnsupportedType) {
		t.Errorf("errors.Is(err, ErrUnsupportedType) = false, err=%v", err)
	}
	for _, sub := range []string{"myTool", "config", "inner", "x"} {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("error %q does not contain %q", err.Error(), sub)
		}
	}
}

func TestMapTools_rejectsMalformedPropertySchema(t *testing.T) {
	cases := []struct {
		name  string
		props map[string]any
	}{
		{"property not an object", map[string]any{"bad": "not-a-map"}},
		{"properties not an object", map[string]any{"obj": map[string]any{"type": "object", "properties": 42}}},
		{"required not a list", map[string]any{"obj": map[string]any{"type": "object", "required": "yes"}}},
		{"required non-string member", map[string]any{"obj": map[string]any{"type": "object", "required": []any{"a", 1}}}},
		{"array items malformed", map[string]any{"arr": map[string]any{"type": "array", "items": 42}}},
		{"array items unsupported", map[string]any{"arr": map[string]any{"type": "array", "items": map[string]any{"type": "null"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tools := []mcp.Tool{tool("myTool", "", tc.props, nil)}
			_, err := mapping.MapTools(tools)
			if err == nil {
				t.Fatal("MapTools returned nil error, want malformed-schema error")
			}
			if !strings.Contains(err.Error(), "myTool") {
				t.Errorf("error %q does not contain tool name", err.Error())
			}
		})
	}
}

func TestMapTools_rejectsNonStringEnumMember(t *testing.T) {
	props := map[string]any{
		"mode": map[string]any{"type": "string", "enum": []any{"fast", 42, "slow"}},
	}
	tools := []mcp.Tool{tool("myTool", "", props, nil)}
	_, err := mapping.MapTools(tools)
	if err == nil {
		t.Fatal("MapTools returned nil error, want malformed enum error")
	}
	if !strings.Contains(err.Error(), "myTool") || !strings.Contains(err.Error(), "mode") {
		t.Errorf("error %q does not include tool/field path", err.Error())
	}
}

func TestMapTools_rejectsDuplicateToolNames(t *testing.T) {
	base := tool("dup", "", map[string]any{}, nil)
	tools := []mcp.Tool{base, base}

	_, err := mapping.MapTools(tools)
	if err == nil {
		t.Fatal("MapTools returned nil error, want duplicate-name error")
	}
	if !errors.Is(err, mapping.ErrDuplicateName) {
		t.Errorf("errors.Is(err, ErrDuplicateName) = false, err=%v", err)
	}
	if !strings.Contains(err.Error(), "dup") {
		t.Errorf("error %q does not contain duplicate tool name", err.Error())
	}
}

func TestMapTools_recursionGuardOnCyclicSchema(t *testing.T) {
	node := map[string]any{"type": "object", "properties": map[string]any{}}
	node["properties"] = map[string]any{"self": node}

	tools := []mcp.Tool{tool("cyc", "", map[string]any{"node": node}, nil)}
	_, err := mapping.MapTools(tools)
	if err == nil {
		t.Fatal("MapTools returned nil error for a cyclic schema, want a recursion error")
	}
	if !strings.Contains(err.Error(), "cyc") {
		t.Errorf("error %q does not contain tool name", err.Error())
	}
}

func TestMapTools_recursionGuardDeepNesting(t *testing.T) {
	depth := 300
	props := map[string]any{}
	cur := props
	for i := 0; i < depth; i++ {
		child := map[string]any{"type": "object", "properties": map[string]any{}}
		cur["n"] = child
		cur = child["properties"].(map[string]any)
	}
	tools := []mcp.Tool{tool("deep", "", props, nil)}
	_, err := mapping.MapTools(tools)
	if err == nil {
		t.Fatal("MapTools returned nil error for over-deep nesting, want a recursion error")
	}
}

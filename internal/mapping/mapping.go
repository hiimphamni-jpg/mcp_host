// Package mapping converts discovered MCP tool definitions into Gemini
// function declarations. It depends only on mcp-go types (input) and the
// genai SDK types (output), so both internal/agent and internal/llm remain
// SDK-free (TDD §2.1, AC5).
package mapping

import (
	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"
)

// MapTools converts a slice of discovered MCP tools into Gemini function
// declarations. It supports the MVP JSON Schema subset (objects, nested
// objects, string/number/integer/boolean/array, enum, required, and array
// items) losslessly, rejects unsupported or malformed schemas with the tool
// name and field path in the error, and rejects duplicate tool names rather
// than silently overwriting (TDD §6).
func MapTools(tools []mcp.Tool) ([]*genai.FunctionDeclaration, error) {
	declarations := make([]*genai.FunctionDeclaration, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		if tool.Name == "" {
			return nil, malformed("", "", "tool name is empty")
		}
		if _, dup := seen[tool.Name]; dup {
			return nil, duplicateName(tool.Name)
		}
		seen[tool.Name] = struct{}{}

		params, err := mapSchemaForTool(tool)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		})
	}

	return declarations, nil
}

// mapSchemaForTool builds the root parameters schema (an object) from a tool's
// InputSchema, preserving required fields at the root.
func mapSchemaForTool(tool mcp.Tool) (*genai.Schema, error) {
	s := &genai.Schema{Type: genai.TypeObject}

	props := tool.InputSchema.Properties
	s.Properties = make(map[string]*genai.Schema, len(props))
	for name, raw := range props {
		child, err := mapProperty(tool.Name, raw, name, 1)
		if err != nil {
			return nil, err
		}
		s.Properties[name] = child
	}

	if req := tool.InputSchema.Required; len(req) > 0 {
		s.Required = append([]string(nil), req...)
	}

	return s, nil
}

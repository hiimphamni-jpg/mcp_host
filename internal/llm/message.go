// Package llm defines a provider-neutral generation surface (types + LLM
// interface) plus a Gemini adapter. internal/agent (FEAT-00006) depends only on
// this package's neutral interface, never on the Gemini SDK or mcp-go (TDD
// §2.1, AC5). This file holds only the SDK-free conversation types.
package llm

// Role identifies the producer of a Message in a conversation turn.
type Role string

const (
	// RoleUser marks user/assistant-tool-result turns.
	RoleUser Role = "user"
	// RoleModel marks model-generated turns.
	RoleModel Role = "model"
)

// Message is a single neutral conversation turn. It may carry free text, model
// tool calls, and/or user tool results, depending on Role. It has no provider
// dependency, so it stays stable regardless of the underlying LLM.
type Message struct {
	Role        Role
	Text        string
	ToolCalls   []*ToolCall
	ToolResults []*ToolResult
}

// ToolCall is a neutral request from the model to execute a named tool.
// ThoughtSignature carries the provider's opaque thought signature when present
// so a returned tool call can be replayed faithfully in a later turn.
type ToolCall struct {
	Name             string
	Arguments        map[string]any
	ThoughtSignature []byte
}

// ToolResult is the observed outcome of a tool call, fed back to the model.
type ToolResult struct {
	Name     string
	Response map[string]any
}

// Response is a neutral model response: final text and/or tool calls to run.
type Response struct {
	Text      string
	ToolCalls []*ToolCall
}

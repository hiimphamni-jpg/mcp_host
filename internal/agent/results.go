package agent

import (
	"context"
	"encoding/json"
	"errors"

	"mcp-host/internal/llm"
)

// This file bounds tool results and maps tool-call failures into safe, bounded,
// non-secret context for the LLM (business-logic §5/§7, ADR-0001 D2/D3). The
// agent never sees mcp-go types here: cmd/server already converted the raw
// result into a neutral llm.ToolResult via the ToolCaller seam, and failures
// arrive as opaque error values classified only by stdlib sentinels.

// truncationMarker signals that a tool result exceeded MCP_MAX_RESULT_BYTES and
// was cut with an explicit marker (business-logic §3, TDD §7).
const truncationMarker = "...[truncated]"

// BoundedResult returns a copy of tr with its Response serialized to JSON and
// truncated to maxBytes. Excess bytes are dropped with the explicit marker so
// the model understands the result was bounded. The bounded text is surfaced as
// a single "content" field; a nil/empty result is a safe empty marker.
func BoundedResult(tr *llm.ToolResult, maxBytes int) *llm.ToolResult {
	if tr == nil {
		tr = &llm.ToolResult{}
	}
	b, err := json.Marshal(tr.Response)
	if err != nil {
		b = []byte("{}")
	}
	text := string(b)
	if len(text) > maxBytes {
		keep := maxBytes - len(truncationMarker)
		if keep < 0 {
			keep = 0
		}
		text = text[:keep] + truncationMarker
	}
	return &llm.ToolResult{
		Name:     tr.Name,
		Response: map[string]any{"content": text},
	}
}

// MessageResult builds a neutral tool result that carries only a short,
// non-sensitive message as context for the model (e.g. "action unavailable" or
// "arguments failed validation"). It is bounded by construction.
func MessageResult(name, message string) *llm.ToolResult {
	return &llm.ToolResult{
		Name:     name,
		Response: map[string]any{"content": message},
	}
}

// ErrorResult maps a tool-call error into a bounded, non-secret tool result so
// the loop can continue and let the model explain or recover (AC4, D3). The
// raw error string is never embedded, preventing secret/path leaks; only a
// fixed category is produced. Caller cancellation is deliberately NOT handled
// here — the loop treats it as terminal before this is reached.
func ErrorResult(name string, err error) *llm.ToolResult {
	message := "tool call failed"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		message = "tool call timed out"
	case errors.Is(err, context.Canceled):
		message = "tool call cancelled"
	}
	return &llm.ToolResult{
		Name:     name,
		Response: map[string]any{"content": message, "error": true},
	}
}

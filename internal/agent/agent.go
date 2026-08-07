// Package agent implements the bounded LLM<->MCP agentic loop (FEAT-00006).
// It depends only on llm.LLM and its own SDK-free ToolCaller seam, never on
// mcp-go or the Gemini SDK (TDD §2.1, AC5). The loop feeds each MCP tool
// result back to the LLM bounded to MCP_MAX_RESULT_BYTES, maps tool failures
// to safe non-secret context, and honours caller cancellation (business-logic
// §3/§5, ADR-0001 D3).
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"mcp-host/internal/llm"
)

// ToolCaller is the narrow, SDK-free execution seam the agent uses to invoke
// tools. cmd/server implements it by wrapping mcpclient.Client and converting
// the mcp-go *mcp.CallToolResult into a neutral llm.ToolResult before the
// agent sees it (ADR-0001 D2). This keeps internal/agent free of mcp-go and
// genai so a fake can drive the whole loop in tests (AC5).
type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error)
}

// Options configures an Agent.
type Options struct {
	// LLM is the generation surface (llm.LLM). Required.
	LLM llm.LLM
	// Tools executes tool calls (ToolCaller). Required.
	Tools ToolCaller
	// Schemas maps each available tool name to its neutral JSON Schema object
	// (map[string]any). It is the agent's own registry, built by cmd/server
	// from discovered tools (ADR-0001 D1); the agent never imports mcp-go.
	Schemas map[string]any
	// MaxIterations bounds the number of LLM-to-tool iterations per prompt.
	MaxIterations int
	// MaxResultBytes caps the serialized tool result retained in history.
	MaxResultBytes int
	// OnEvent is an optional observer for Agent life events (ADR-00006). It is
	// invoked synchronously on every event; a panic is recovered so a misbehaving
	// observer can never break the loop. nil means no-op.
	OnEvent func(Event)
}

// Agent runs a bounded agentic loop: LLM inference, tool execution, and result
// feedback until the LLM returns final text or the iteration limit is hit.
type Agent struct {
	llm            llm.LLM
	tools          ToolCaller
	schemas        map[string]any
	maxIterations  int
	maxResultBytes int
	onEvent        func(Event)
}

// New validates Options and returns an Agent.
func New(opts Options) (*Agent, error) {
	if opts.LLM == nil {
		return nil, fmt.Errorf("agent: LLM is required")
	}
	if opts.Tools == nil {
		return nil, fmt.Errorf("agent: tools caller is required")
	}
	if opts.MaxIterations <= 0 {
		return nil, fmt.Errorf("agent: MaxIterations must be positive")
	}
	if opts.MaxResultBytes <= 0 {
		return nil, fmt.Errorf("agent: MaxResultBytes must be positive")
	}
	return &Agent{
		llm:            opts.LLM,
		tools:          opts.Tools,
		schemas:        opts.Schemas,
		maxIterations:  opts.MaxIterations,
		maxResultBytes: opts.MaxResultBytes,
		onEvent:        opts.OnEvent,
	}, nil
}

// emit forwards an event to the configured observer, recovering from panics so
// an observer bug can never leak into the loop (ADR-00006).
func (a *Agent) emit(e Event) {
	observer := a.onEvent
	if observer == nil {
		return
	}
	defer func() { _ = recover() }()
	observer(e)
}

// Run executes the loop for one prompt and returns the final LLM text. It
// stops early on caller cancellation (context.Canceled is terminal) and on an
// LLM timeout/provider failure; a tool-call failure is bounded into history so
// the loop can continue to explain it (ADR-0001 D3).
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	a.emit(Event{Type: EventStart})
	history := []*llm.Message{{Role: llm.RoleUser, Text: prompt}}

	for i := 0; i < a.maxIterations; i++ {
		resp, err := a.llm.Generate(ctx, &llm.Request{
			Contents:  history,
			ToolNames: a.toolNames(),
		})
		if err != nil {
			return "", classifyGenerateError(err)
		}
		if resp == nil {
			resp = &llm.Response{}
		}
		if resp.Text != "" && len(resp.ToolCalls) == 0 {
			a.emit(Event{Type: EventFinal, Text: resp.Text})
			return resp.Text, nil
		}

		results := make([]*llm.ToolResult, 0, len(resp.ToolCalls))
		if len(resp.ToolCalls) > 0 {
			if resp.Text != "" {
				a.emit(Event{Type: EventText, Text: resp.Text})
			}
			history = append(history, &llm.Message{
				Role:      llm.RoleModel,
				Text:      resp.Text,
				ToolCalls: resp.ToolCalls,
			})
		}
		for _, tc := range resp.ToolCalls {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			tr, err := a.execute(ctx, tc)
			if err != nil {
				return "", err
			}
			results = append(results, tr)
		}
		history = append(history, &llm.Message{Role: llm.RoleUser, ToolResults: results})
	}
	a.emit(Event{Type: EventFinal, Text: ""})
	return "", iterationLimitError()
}

// execute validates and runs one tool call, returning a bounded tool result.
// A caller-cancelled context is returned as a terminal error; every other tool
// failure is converted to a bounded, non-secret result so the loop continues
// (ADR-0001 D3). Tool calls are executed sequentially in request order (D4).
func (a *Agent) execute(ctx context.Context, tc *llm.ToolCall) (*llm.ToolResult, error) {
	if tc == nil {
		return MessageResult("", "action unavailable: empty tool call"), nil
	}
	rawSchema, ok := a.schemas[tc.Name]
	if !ok {
		return MessageResult(tc.Name, "action unavailable: tool not found"), nil
	}
	schema, ok := rawSchema.(map[string]any)
	if !ok {
		return MessageResult(tc.Name, "arguments failed validation: malformed schema"), nil
	}
	if err := validateArgs(schema, tc.Arguments); err != nil {
		return MessageResult(tc.Name, "arguments failed validation: "+err.Error()), nil
	}
	a.emit(Event{Type: EventTool, ToolName: tc.Name, ToolArgs: tc.Arguments, Phase: ToolPhaseStart})
	tr, err := a.tools.CallTool(ctx, tc.Name, tc.Arguments)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err() // caller cancellation takes precedence (D3)
		}
		return ErrorResult(tc.Name, err), nil
	}
	a.emit(Event{Type: EventTool, ToolName: tc.Name, Phase: ToolPhaseResult, Bytes: resultBytes(tr)})
	return BoundedResult(tr, a.maxResultBytes), nil
}

// resultBytes reports the serialized size of a raw tool result for observation.
func resultBytes(tr *llm.ToolResult) int {
	if tr == nil {
		return 0
	}
	b, err := json.Marshal(tr.Response)
	if err != nil {
		return 0
	}
	return len(b)
}

// classifyGenerateError normalizes an LLM failure. Caller cancellation passes
// through unchanged and is terminal; LLM timeout/provider failures keep their
// typed sentinel (errors.Is) so callers can classify them; any other error is
// wrapped without embedding its raw message (no secret leak).
func classifyGenerateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, llm.ErrTimeout) || errors.Is(err, llm.ErrProvider) {
		return &Error{Cause: err, Operation: "Generate", message: "LLM generation failed"}
	}
	return &Error{Cause: err, Operation: "Generate", message: "LLM generation failed"}
}

// toolNames returns the available tool names in sorted order for the LLM
// request's ToolNames indirection (deterministic config assembly).
func (a *Agent) toolNames() []string {
	out := make([]string, 0, len(a.schemas))
	for name := range a.schemas {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

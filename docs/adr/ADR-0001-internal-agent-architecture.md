# ADR-00001: Internal Agent Loop — Orchestration Boundary and Tool-Result Conversion

- **Status:** Accepted
- **Date:** 2026-08-06
- **Decision makers:** Architect (resolving open design points flagged during FEAT-00005 / FEAT-00006 planning)
- **Related feature:** FEAT-00006 (Bounded agentic loop)

## Context

`internal/agent` (FEAT-00006) must run a bounded LLM↔MCP loop while depending **only** on
`llm.LLM` and `mcpclient.Client` interfaces (TDD §2.1, AC5), keeping it free of `mcp-go` and
the Gemini SDK. Four design points were left open in prior packages and must be settled here:

1. **Tool-declaration hand-off** — flagged at `internal/llm/interface.go:18`: how the agent
   references tools that the Gemini adapter must wire as `genai.FunctionDeclaration`s.
2. **`mcp.CallToolResult` → neutral result** — `internal/agent` must not import `mcp-go`, but
   `mcpclient.Client.CallTool` returns `*mcp.CallToolResult`; who converts it to a neutral
   `llm.ToolResult`?
3. **Deadline vs cancellation precedence** across the loop.
4. **Sequential vs parallel** execution when Gemini requests multiple tool calls in one turn.

## Decisions

### D1. Orchestrator owns discovery + mapping; agent receives neutral inputs

`cmd/server` (composition root) performs MCP discovery and `mapping.MapTools`, then:
- passes `declsByName map[string]*genai.FunctionDeclaration` to the Gemini adapter
  (`llm.New`), and
- passes a neutral tool registry (`map[string]*llm.ToolDecl`, plain name/schema data, no
  `genai`/`mcp-go`) to the agent.

The agent therefore never pre-renders declarations and never owns provider types. This closes
the indirection in `internal/llm/interface.go` (`Request.ToolNames` resolves to pre-built
declarations in the adapter via `selectedDecls`).

**Rationale:** Keeps `internal/agent` provider-SDK-free (AC5) and gives a single place
(cmd/server) for discovery wiring.

**D2. Tool-result conversion lives in the composition root adapter (neutral boundary)**

`mcp-go`'s `mcp.CallToolResult` must not cross into `internal/agent`. So the agent depends on a
narrow, SDK-free execution seam defined next to it:

```go
type ToolCaller interface {
    CallTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error)
}
```

A concrete adapter **in `cmd/server`** wraps `mcpclient.Client` and converts the raw
`*mcp.CallToolResult` content into a neutral `llm.ToolResult{Name, Response map[string]any}`
before the agent sees it. The agent does the bounding/truncation and safe-error mapping on the
already-neutral result.

**Rationale:** `mcpclient` must not import `internal/llm` (declared in its package comment) and
`internal/agent` must not import `mcp-go`; the composition root legitimately owns this glue.

**D3. Caller cancellation takes precedence; per-op timeouts are internal and non-fatal to the loop**

- Each LLM/MCP operation bounds itself by applying `LLM_TIMEOUT` / `MCP_TIMEOUT` via
  `context.WithTimeout` layered on top of the caller context.
- If the caller cancels, the derived context returns `context.Canceled`, which the agent
  treats as the **terminal cancellation status**: it stops the loop, closes MCP, and returns a
  cancellation error (business-logic §5 "Caller cancellation").
- A per-operation **timeout** during an LLM call is a terminal failure (`ErrTimeout`); a
  timeout during a **tool call** becomes a bounded, safe tool result so the loop can continue
  to explain the failure (business-logic §5).
- Precedence: caller cancellation overrides per-operation timeouts.

**D4. Sequential tool execution in the MVP**

When one Gemini turn requests multiple tool calls, the agent executes them **sequentially** in
request order. Parallel/concurrent execution is deferred until multi-server support has an
explicit policy and conflict-resolution design (business-logic §4).

**Rationale:** MVP is single-MCP-server; sequential execution is deterministic, bounded, and
simplifies result ordering in conversation history.

## Consequences

- `internal/agent` gains a new `internal/agent`-local `ToolCaller` interface (implemented in
  `cmd/server`), keeping it free of `mcp-go`/`genai`.
- Discovery + mapping wiring moves into `cmd/server` (already the composition root).
- Independent future providers/transports can implement `llm.LLM` + `ToolCaller` without
  touching the agent.
- MVP agent loop is sequential; concurrency is a later, deliberate capability.
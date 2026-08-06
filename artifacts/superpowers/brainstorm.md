# Brainstorm — FEAT-00006: Bounded agentic loop, tool-result context, cancellation, and error handling

Status: `/think` (Flow 1, Phase: Clarify)
Source: REGISTRY.md (P0 / S1) + docs/tdd/MCP_HOST_TDD.md §7 + business-logic.md §3–5

## Goal
Implement `internal/agent`: a provider-neutral, bounded agentic loop that:
- sends conversation history + mapped tool declarations to the LLM,
- executes validated MCP tool calls, appends bounded results (or safe errors) to history,
- returns final text when the model stops calling tools,
- enforces `AGENT_MAX_ITERATIONS`, `LLM_TIMEOUT`, `MCP_TIMEOUT`, and `MCP_MAX_RESULT_BYTES`,
- honors caller cancellation end-to-end, and
- maps every failure mode to a safe, non-secret result/error.

## Constraints
- `internal/agent` must depend ONLY on the `llm.LLM` and `mcpclient.Client` interfaces (AC5); no `mcp-go`, no Gemini SDK.
- No implementation code until an APPROVED `/plan` exists (Plan Gate).
- One MCP server per invocation; tool names validated against discovered `tools/list` + policy.
- Timeouts mandatory on every LLM/MCP operation; no silent failures; no logging of secrets/raw prompts/stack traces.
- Tool results in history truncated at `MCP_MAX_RESULT_BYTES` with an explicit marker.
- Must follow TDD (Red → Green → Refactor); agent branches target ≥80% statement coverage.

## Known Context
- `internal/llm` already provides neutral types (`Message`, `ToolCall`, `ToolResult`, `Response`) and `LLM.Generate` (interface.go), plus Gemini adapter and mapper (`internal/mapping`) from FEAT-00004/00005.
- `internal/mcpclient.Client` exposes `Initialize/ListTools/CallTool/Close`; CallTool returns `*mcp.CallToolResult`.
- `internal/policy` covers executable, tool, root-path, schema, and result-size policy (ready to be invoked by the agent).
- TDD §7 specifies the loop pseudocode and rules. business-logic.md §5 defines the error-handling contract table.
- ONE open architecture decision flagged in `internal/llm/interface.go`: `Request.ToolNames` indirection — agent lists tool names, adapter maps names → already-built declarations. Resolution belongs to `/architect`.

## Risks
- **Interface leakage (HIGH):** if the agent returns `*mcp.CallToolResult` or provider SDK types, AC5 breaks. Need neutral `llm.ToolResult`.
- **Iteration/timeout interplay:** LLM_TIMEOUT vs MCP_TIMEOUT vs caller context — one deadline must not mask another; cancellation must propagate.
- **Result truncation correctness:** truncation must be deterministic and marked, so the model can recover; must never feed back secrets/unbounded output.
- **Parallel tool calls in one response:** TDD pseudocode says "for each requested tool call" — sequential is MVP; parallel execution is a scope question for the architect.
- **Failure taxonomy:** unknown tool, invalid args, MCP error, crash, timeout, iteration-limit, cancellation must each map to the contract in business-logic.md §5.

## Options
1. **Sequential tool execution, single agent package** — execute one tool call at a time, append result, loop. Simplest, matches TDD §7, easy to unit test with fakes. Recommended for MVP.
2. **Batch/parallel tool execution within one LLM response** — execute multiple tool calls concurrently. Faster but adds concurrency + ordering complexity, harder bounded-context guarantees.
3. **Tool-call → provider-native result embedding** — adapters embed tool results in provider-specific structures. Breaks neutrality; rejected.
4. **Multi-server aware agent** — loop over multiple MCP clients. Out of MVP scope (business-logic.md §7).

## Recommendation
Option 1: a single `internal/agent` package with a sequential, single-server loop driving `llm.LLM` and `mcpclient.Client` interfaces. Tool results become neutral `llm.ToolResult`; MCP-typed results are converted inside a thin, agent-owned boundary. Executes tool calls one at a time to keep bounded-context and failure handling deterministic for MVP.

## Acceptance Criteria (draft, to be finalized by /ba + /plan)
- AC-A: Given a fake LLM returning a tool call then final text, the agent executes the MCP tool via the fake client, appends a bounded result, and returns final text (mirrors business AC3).
- AC-B: Agent refuses calls to unknown/unauthorized tools or schematically invalid arguments without invoking MCP, returning a safe error result (mirrors business AC5, §5).
- AC-C: Agent stops with a clear error when `AGENT_MAX_ITERATIONS` is reached; no background work.
- AC-D: Tool results > `MCP_MAX_RESULT_BYTES` are truncated with an explicit marker; no secret/unbounded content leaks.
- AC-E: Cancelling the parent context cancels active LLM/MCP work and surfaces cancellation, not a generic error.
- AC-F: Agent imports only neutral interfaces — compiles with fakes; no `mcp-go` or Gemini SDK in `internal/agent`.

## Architecture design required?
**YES — `/architect design internal-agent-loop`** is required before `/plan`:
1. Resolve the open `Request.ToolNames` → declaration indirection decision (interface.go comment).
2. Define the neutral tool-result/error shape the agent feeds back and who converts `mcp.CallToolResult` (agent vs adapter).
3. Define deadline/cancellation precedence (parent ctx vs LLM_TIMEOUT vs MCP_TIMEOUT).
4. Decide sequential vs parallel tool execution (recommend sequential for MVP) and record an ADR if the decision has durable impact.

## Next
`/architect design internal-agent-loop` → then `/ba spec` + `/ba ac FEAT-00006` → `/pm registry`/`/pm dor FEAT-00006` → `/plan FEAT-00006` (needs APPROVED) → `/build`.

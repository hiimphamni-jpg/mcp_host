# Plan - FEAT-00006: Bounded agentic loop, tool-result context, cancellation, and error handling

Source of truth: `docs/tdd/MCP_HOST_TDD.md` section 7 (Agentic Loop), 5/6 (validation), 8 (security), 10 (AC3/AC4/AC5); `docs/business-logic.md` section 3 (Agent State), 5 (Error Handling).

## Goal

Create the `internal/agent` package implementing a bounded agentic loop (LLM <-> MCP tool calls) and wire it into `cmd/server`. The loop:

- runs at most `AGENT_MAX_ITERATIONS` LLM-to-tool iterations per prompt and returns a clear failure (`ErrIterationLimit`) when exhausted;
- feeds each MCP tool result back to the LLM bounded to `MCP_MAX_RESULT_BYTES`, truncating excess with an explicit marker;
- represents MCP failures, crashes, timeouts, and unauthorized/invalid tool calls as safe, bounded, non-secret context so Gemini can recover or explain - never crashing the host (AC4);
- cancels the whole invocation through the request `context` (propagates to LLM and MCP);
- depends only on `internal/llm.LLM` and the MCP client interface - no `mcp-go`/Gemini SDK in business logic - so fakes drive it in tests (AC5), and providers/transports can be swapped.

## Assumptions (design decisions resolving the "open decision" markers)

1. **Declaration wiring.** The Gemini adapter is constructed with its tool declarations (`llm.New(client, model, timeout, declsByName)`). The agent must not produce `genai` declarations, so tool discovery and `mapping.MapTools` happen in the orchestrator (`cmd/server`), which builds both `declsByName` (for the Gemini adapter) and a neutral schema registry (for the agent's validation). The agent's `Run` takes that registry instead of re-listing tools.
2. **Agent's MCP surface.** The agent depends on the existing `mcpclient.Client` interface. MCP value types reach the agent only through that interface (same intent as `internal/mcpclient/interface.go`); the agent does not import `mcp-go` for constructs and never touches the Gemini SDK.
3. **Per-request schema-arg validation** is done against the neutral schema registry inside the agent with a small stdlib-only validator (no `mcp-go`/`genai` import), which enables fake-driven tests. It rejects unknown tool names and schema-invalid arguments, producing a bounded tool result instead of calling MCP (business-logic section 5).
4. **Result bounding** extracts a single text from `mcp.CallToolResult` (text content / embedded-resource text / `IsError`), serializes it as a short JSON envelope, and truncates to `MaxResultBytes` with an explicit `...[truncated]` marker. Secrets and host paths are removed.
5. **CLI/exit-code formatting remains FEAT-00007.** `cmd/server` wires the pipeline and, until FEAT-00007, reads one prompt from a `--prompt` flag to exercise the loop.

## Plan

### Step 1: Agent package skeleton + types + sentinel errors
- **Files**: `internal/agent/agent.go`, `internal/agent/errors.go`
- **Agent**: /dev (design consent already recorded in this plan and in the TDD)
- **Change**: Add `internal/agent`. Define neutral `Options`:
  `{ llm.LLM, mcpclient.Client, Schemas map[string]any, MaxIterations, MaxResultBytes int }`
  and `Run(ctx context.Context, prompt string) (string, error)`. Add sentinel
  `ErrIterationLimit` and a typed `agent.Error{Cause, Operation}` mirroring
  `internal/llm/errors.go` style (user-safe message). Skeleton with scaffolding only - no loop body yet.
- **Verify**: `go build ./...`; unit test that construction rejects non-positive limits and nil LLM/MCP.
- **Duration**: ~8 min

### Step 2: Neutral schema-argument validator
- **Files**: `internal/agent/schema.go`, `internal/agent/schema_test.go`
- **Agent**: /dev
- **Change**: stdlib-only `validateArgs(schema map[string]any, args map[string]any) error` for the MVP schema subset (object props, required, primitive, enum, array items), mirroring the `internal/mapping` subset conventions without importing it (deliberate, to keep the agent free of `genai`).
- **Verify**: `go test ./internal/agent/...` - table tests: missing required, wrong type, enum mismatch, nested object, array items, empty-schema/no-args tool.
- **Duration**: ~8 min

### Step 3: Tool-result bounding + safe error mapping
- **Files**: `internal/agent/results.go`, `internal/agent/results_test.go`
- **Agent**: /dev
- **Change**: `ResultFrom(call *mcp.CallToolResult, maxBytes int) map[string]any` producing a bounded JSON/text tool result; `ErrorResult(err error) map[string]any` mapping a `mcpclient` typed error (crash/timeout/invalid/call) and `context.Canceled` into a bounded, non-secret message. Truncate to `MaxResultBytes` with explicit marker. Pure helpers, unit-tested.
- **Verify**: `go test ./internal/agent -run 'Result'` - truncation marker, no-secret, error mapping.
- **Duration**: ~5 min

### Step 4: Agent loop (`Run`) - main logic
- **Files**: `internal/agent/agent.go` (extend)
- **Agent**: /dev
- **Change**:
  ```
  history = [llm.Message{Role: user, Text: prompt}]
  for i in 1..MaxIterations:
    resp = LLM.Generate(ctx, &Request{Contents: history, ToolNames: schema keys})
    if err: return typed err (timeout/cancel/provider)
    if resp.Text != "" and len(resp.ToolCalls) == 0: return resp.Text, nil
    toolResults = []
    for each ToolCall:
      if ctx.Err() != nil: return ctx.Err()
      if unknown tool name: toolResult = bounded "action unavailable"
      else if validateArgs fails: toolResult = bounded "arguments failed validation"
      else:
        r, err := MCP.CallTool(ctx, name, args)
        toolResult = err != nil ? ErrorResult(err) : ResultFrom(r)
    append user llm.Message{ToolResults: toolResults}
  return "", ErrIterationLimit
  ```
  All bounded results capped to `MaxResultBytes`.
- **Verify**: `go test ./internal/agent -run TestAgentRun` with fakes (Step 5).
- **Duration**: ~10 min

### Step 5: Agent unit tests with fakes
- **Files**: `internal/agent/agent_test.go`, `internal/agent/errors_test.go`
- **Agent**: /dev
- **Change**: Tests:
  - final text returned directly;
  - single tool call then final text;
  - multiple tool calls in one turn;
  - iteration limit returns `ErrIterationLimit`;
  - unknown tool name becomes safe bounded result;
  - invalid args become safe bounded result;
  - MCP failure becomes bounded result, host keeps running;
  - cancellation propagates (cancel ctx mid-loop -> `context.Canceled`; fake asserts `CallTool` receives the cancelled context);
  - result is truncated;
  - secret content is not in output.
  - Fakes: `stubLLM` returns scripted `*llm.Response` sequences; `stubMCP` returns fixed results/errors. No real `mcp-go` client or Gemini SDK.
- **Verify**: `go test ./internal/agent/...`; `go vet ./...`.
- **Duration**: ~10 min

### Step 6: Wire the pipeline in `cmd/server`
- **Files**: `cmd/server/main.go`, `cmd/server/main_test.go`
- **Agent**: /dev
- **Change**: In `run()` after config validation: build policy -> `mcpclient.NewStdio` -> `Initialize` -> `ListTools` -> `mapping.MapTools` -> build `declsByName` -> `llm.New` -> construct `internal/agent` with neutral schemas + limits -> `agent.Run(ctx, ...)`. Defer `client.Close()` on every path (AC4: no orphan). Read one prompt from a `--prompt` flag when provided. Full stdout/exit-code contract stays FEAT-00007. Add a smoke test that an unconfigured invocation ends cleanly (no real Gemini/MCP in CI).
- **Verify**: `go build ./...`; `go test ./cmd/server/...`.
- **Duration**: ~8 min

### Step 7: Cross-package quality gates
- **Files**: none new
- **Agent**: /dev
- **Change**: `go vet ./...`, `go test ./...`; confirm the new agent logic coverage >= 80% statement; confirm no new `mcp-go`/`genai` import crosses into `internal/agent` (`go list -deps ./internal/agent`); update `docs/REGISTRY.md` DEV column to `done` for FEAT-00006.
- **Verify**: `go vet ./...` and `go test ./...` clean; `grep -r "genai" internal/agent` returns nothing.
- **Duration**: ~5 min

## Risks & Mitigations
- **Open-decision drift (adapter wiring)**: mitigation = Step 1/6 record the decision (orchestrator maps once, passes decls + neutral schema to agent); keep agent `genai`-free (Step 7 guard).
- **Schema validator diverges from `internal/mapping`**: mitigation = stdlib-only subset with table tests; if divergence grows, extract to `internal/policy` (neutral) and have both consume it.
- **`mcp.CallToolResult` shape variability** (text vs embedded resource): mitigation = `ResultFrom` covered by table tests for both content kinds.
- **Cancellation does not reach in-flight MCP call**: mitigation = agent passes caller ctx through to `CallTool` (already context-aware); verified in Step 5 cancellation test.
- **Secret leakage in results/errors**: mitigation = results/errors carry only short labels/truncated JSON; verified by the no-secret test in Step 5.

## Rollback Plan
- All changes additive. Revert = `git checkout -- internal/agent cmd/server/main.go cmd/server/main_test.go`; delete `internal/agent`; restore `docs/REGISTRY.md` row to backlog.
- No DB migrations and no breaking API change. `mcpclient`/`llm`/`mapping`/`policy` surfaces are unchanged.

## Parallel Opportunities
- Step 2 (validator) and Step 3 (result bounding) are independent - different files, same package, no interdependency; they can run concurrently before Step 4 consumes both.
- Step 5 depends on Step 4 (fakes must run the real loop); Steps 1-3 are sequential.

# Brainstorm — FEAT-00008: HTTP/SSE API Gateway and Next.js UI

> Phase: Flow 1 — Clarify (`/think`). Task ID: FEAT-00008. Dependency: dev, test, qc.
> Registry status on entry: DEFERRED (P2 / S2).

## Goal
Expose the existing headless Go MCP host (Gemini + Filesystem MCP agent loop) to remote
callers over an HTTP/SSE API Gateway, and provide a minimal Next.js web UI that issues
prompts and streams the final response. Scope is one MCP server per request (mirrors MVP
single-server constraint).

## Constraints
- **Plan Gate (mandatory):** no implementation code until `/plan` is built and the user
  replies `APPROVED` (`.gemini/rules/workflow.md §1`).
- **MVP out-of-scope:** HTTP API, SSE, Next.js UI, and remote-caller auth were explicitly
  deferred (business-logic.md §7; TDD §1). This feature is their first full activation, so
  design decisions (auth, SSE semantics, session model) are required before code.
- **Design before build:** spans ≥2 layers (Go transport adapter + TypeScript/Next.js UI) →
  TDD from `/architect` and ADR are required first (`.gemini/rules/workflow` + GEMINI.md §7).
- **Reuse seams / AC5:** must not couple the new gateway to mcp-go or the Gemini SDK.
  Reuse `llm.LLM`, `agent.ToolCaller`, `mapping`, `policy`, `config` via the existing seams.
- Architecture/security rules apply fully: `api-convention.md`, `security.md`,
  `error-handling.md`, `code-style.md` (Go + TS), `database.md` (N/A for MVP scope).
- Endpoint, authentication, event format, reconnection, and disconnect semantics must be
  documented before implementation (TDD §9).

## Known Context
- Go module (`mcp-host`), Go 1.26, `mcp-go v0.57.0`, `google.golang.org/genai v1.66.0`.
- Composition root is `cmd/server/main.go:run` → per-prompt discovery + one agent loop,
  print final text. Agent (`internal/agent`) depends only on `llm.LLM` + `ToolCaller`
  (ADR-0001; AC5 preserved).
- `internal/agent/agent.go` exposes `ToolCaller` (CallTool) and `Options{LLM, Tools,
  Schemas, MaxIterations, MaxResultBytes}` — a clean seam to run the loop per HTTP request.
- `internal/llm` is a single `Generate(ctx, *Request)` provider-neutral surface.
- Config is env-driven (`internal/config`), fail-fast, secrets never printed.
- Existing features FEAT-00001..00007 done within `internal/` and `cmd/server`; tmp/
  integration tests live in `internal/mcpclient`.
- No HTTP/SSE, no Next.js, no auth code exists. No `package.json` at repo root except
  `.opencode/` tooling.

## Risks
- **Remote auth** is the highest-risk area: exposing a filesystem-backed agent over HTTP
  without robust auth is a remote code-exec/full file-access risk (Rhizome). Auth design
  must resolve before Go + Next.js code (security.md).
- **SSE semantics:** one request ↔ one stream; connection lifecycle, reconnection, and
  disconnect behavior must be specified or UX/backpressure bugs and orphaned MCP child
  processes appear (safety.md).
- **Concurrency:** MVP runs one MCP server per invocation; the HTTP gateway must bound
  concurrent prompts / per-request MCP process lifetime and enforce timeouts.
- **Result size:** unbounded SSE frames risk DoS; must retain `MCP_MAX_RESULT_BYTES` cap.
- **Next.js scope creep:** a full chat UI is large; keep UI thin (a single prompt →
  streamed response view) to stay shippable; enrich later.
- **MCP server spawn per request** cost/latency (SSE may be long-lived): scope/lifetime
  policy to be decided by `/architect`.
- Cross-origin / CORS, secrets in UI config, and streaming-cancel propagation.

## Options
- **A — Read-only MVP (recommended first increment):** HTTP POST JSON endpoint
  `/v1/chat` returning SSE events for streaming final text + tool progress; no interactive
  tool-result surfacing; thin static Next.js page posting a prompt and rendering the
  stream server-side. Smallest safe path; auth = bearer API key.
- **B — Full interactive gateway:** bidirectional per-request session, tool-call/result
  streamed to UI, manual approval for tool calls, full Next.js chat app with SSR. Richer
  UX but far larger surface, more SSE contracts and more auth surface.
- **C — Single unified MCP remote transport:** use mcp-go's HTTP transport as the gateway
  (expose host itself to an MCP client) + Next.js as a thin MCP client. Least new code but
  couples harder to mcp-go semantics and complicates auth/stream shaping.
- **D — Do nothing / defer:** keep DEFERRED. Avoids risk; no remote capability.

## Recommendation
**A — Gateway + SSE streaming MVP**, with bearer-token auth, JSON API contract, streaming
SSE events (start / stream / tool / final / error), one MCP server per request bounded by
`AGENT_MAX_ITERATIONS` + `MCP_MAX_RESULT_BYTES`, reuse of `cmd/server` engine via a new
`internal/server` HTTP adapter + `cmd/serverapi`, and a minimal Next.js page. This respects
the existing seams (AC5), keeps the UI thin, and defers interactive chat-auth/back-pressure
until an explicit follow-up feature. **Auth precedes code** — API-key bearer in an
authorize-by token check in the adapter, no LLM-supplied credentials.

## Acceptance Criteria
- **AC1:** Authenticated HTTP endpoint accepts a JSON prompt and returns a valid SSE stream
  (`application/event-stream`) with final text; unauthorized request → 401 envelope.
- **AC2:** Streaming events preserve the bounded loop: emits at least a final-result event
  and, on failure / iteration-limit, a typed error event with a non-2xx or safe payload
  (no secrets, no stack traces).
- **AC3:** Per-request agent works with the existing `internal/agent` and does not need to
  modify engine seams; fake LLM + fake ToolCaller drive a full end-to-end HTTP test without
  subsidiary SDKs (AC5 preserved).
- **AC4:** Trailing shutdown — a client disconnect closes the MCP process; concurrent
  configurable caller limit enforced; result payload capped.
- **AC5:** Thin Next.js UI posts one prompt and renders the SSE-composed final answer only.

## Exit condition (Clarify)
Goal, constraints, risks, recommendation, and acceptance criteria recorded above.
**Architecture design IS required** (≥2 layers + new transport/auth surface).

## Next commands
1. `/architect design FEAT-00008` — TDD + ADR for HTTP/SSE transport, auth, SSE contract,
   MCP-per-request lifetime, and the gateway/UI boundary.
2. `/ba spec` + `/ba ac FEAT-00008` — API contract (api-convention.md) + SSE event schema.
3. `/pm registry` + `/pm dor FEAT-00008` — confirm/assign dependencies, priorities.
4. `/plan FEAT-00008` → wait for **APPROVED** before any code.
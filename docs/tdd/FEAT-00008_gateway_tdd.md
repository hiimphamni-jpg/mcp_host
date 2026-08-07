# TDD: FEAT-00008 — HTTP/SSE Gateway (`POST /v1/chat`)

**Author:** Architect  |  **Date:** 2026-08-07  |  **Status:** Draft (Flow 1 → Design)

- Flow entry: `.gemini/PROJECT_FLOWS.md` **Flow 1 (New Feature) — Design phase**
- Registry: `docs/REGISTRY.md` `FEAT-00008` (P2 / S2)
- Source of truth for seams: `docs/tdd/MCP_HOST_TDD.md`, `ADR-00001`

---

## 1. Problem Statement

Today the host is only reachable through a headless CLI (`cmd/server`, FEAT-00007):
it reads one prompt, runs one bounded agent loop, prints the final text, and exits.
There is no way for a browser UI or an external program to drive the agent and
observe its progress while it runs.

We want an authenticated HTTP surface that:

- accepts a chat prompt as JSON,
- streams lifecycle + progress events back over SSE, and
- returns the final answer.

Constraint: **the CLI entrypoint and `internal/agent` behavior must remain
unchanged** (the project already has green tests tied to AC5). This feature
introduces the gateway as an additive HTTP layer and an additive (optional)
observation seam in the agent, not a rewrite.

Success metric: a programmatic/browser client can `POST /v1/chat` and render a
fully-typed, ordered SSE sequence (`start → stream* → final|error`) that
reproduces the same final answer the CLI would produce for the same prompt.

---

## 2. Proposed Design

### 2.1 System Architecture

```mermaid
graph TD
    subgraph Clients
        Curl[cURL / programmatic]
        Next[Next.js UI (future feature)]
    end
    Curl --> GW[cmd/gateway - net/http]
    Next --> GW
    GW --> Auth[Bearer auth middleware]
    Auth --> Chat[POST /v1/chat handler]
    Chat --> Runner[internal/server runner]
    Chat --> SSE[SSE writer + Flush]
    Runner --> Agent[internal/agent loop]
    Runner --> LLM[internal/llm Gemini]
    Runner --> MCP[internal/mcpclient stdio]
    Runner --> Map[internal/mapping]
    Runner --> Pol[internal/policy]
    Agent -->|OnEvent observe| SSE
    MCP -->|JSON-RPC stdio| Fs[Filesystem MCP Server]

    subgraph unchanged
        Pol
        MCP
        Map
        LLM
        Agent
    end
```

### 2.2 Module Boundaries & New Packages

| Package | Responsibility | Depends on | New? |
| --- | --- | --- | --- |
| `cmd/server` | Unchanged CLI entrypoint | — | unchanged |
| `cmd/gateway` | **New** entrypoint: load gateway config, build server, listen | `internal/server` | new |
| `internal/server` | HTTP layer: router, auth middleware, SSE writer, `Handler` seam, `GET /healthz` | stdlib + `internal/agent` event types only (for mapping in `server.go`) | **new** |
| `internal/server` (runner) | Real `Handler`: per-request MCP lifecycle + agent loop with observer → `agent.Event` | `mcpclient`, `mapping`, `policy`, `llm`, `agent`, `mcp-go`, `genai` | **new** |
| `internal/agent` | Adds optional `OnEvent` observer (ADR-00006); behavior unchanged | `llm` | **changed (additive)** |
| `internal/llm`, `mcpclient`, `mapping`, `policy` | Unchanged | — | unchanged |

Principle (ADR-00001 D1/D2): `internal/server/server.go` (HTTP) depends only on
an SDK-free `Handler` seam, so the SSE/auth/validation layer is testable without
`mcp-go` or `genai`. The concrete runner (composition of MCP + LLM) mirrors what
`cmd/server/main.go` already does, but stays inside `internal/server` so the CLI
file is untouched. A small duplication of the ~30-line composition block with
`cmd/server` is accepted for MVP (see §3); it can later be extracted into an
`internal/composition` helper as a `REFAC-xxxxx`.

### 2.3 Agent Observation Seam (internal/agent, additive)

Add to `internal/agent` an SDK-free observer so the HTTP layer can stream
progress without owning the loop (ADR-00006):

```go
type EventType string
const (
    EventStart EventType = "start"
    EventText  EventType = "text"   // non-terminal model text fragment
    EventTool  EventType = "tool"   // before/after each sequential tool call
    EventFinal EventType = "final"
)
type Event struct {
    Type     EventType
    Text     string
    ToolName string
    ToolArgs map[string]any
    Bytes    int
}
// in Options:
OnEvent func(Event)   // optional; nil = no-op; synchronous call with recover()
```

Emission points: `EventStart` once in `Run`; per iteration `EventText` (if
`resp.Text != ""`), `EventTool{start}` before each `execute` and
`EventTool{result}` after (with byte count). `EventFinal` on the success return.
The existing `Run(ctx, prompt) (string, error)` signature and `cmd/server` behavior
are unchanged. **cmd/server passes no `OnEvent` → zero behavior change; all
existing tests stay green.**

### 2.4 API Contract

#### `POST /v1/chat`

- **Auth:** `Authorization: Bearer <token>` (ADR-0004)
- **Body:** `{ "prompt": "<string>" }`
- **Accept:** `text/event-stream`
- **Response:** SSE stream; Requests outside the concurrency guard → `429 body`
  before any SSE.

**Pre-SSE (JSON error envelope, HTTP status):**

| Case | Status | Body |
| --- | --- | --- |
| Missing/invalid token | `401` | `{"error":{...}}` |
| Malformed JSON / missing `prompt` | `400` | `{"error":{"code":"BAD_REQUEST",...}}` |
| Empty/whitespace prompt | `422` | `{"error":{"code":"INVALID_CONTENT",...}}` |
| Over concurrency cap | `429` | `{"error":{"code":"TOO_MANY_REQUESTS",...}}` |

**In-stream (after `start`, ADR-00003):** ordered `event` frames the decide
`final` or `error`, then close.

```
event: data
...
```

| Event | `data:` payload |
| --- | --- |
| `start` | `{"request_id":"..","model":".."}` |
| `stream` | `{"type":"text","text":".."}`  ∥  `{"type":"tool","name":"..","status":"start"/"result","bytes":N}` |
| `final` | `{"text":".."}` |
| `error` | `{"code":"..","message":".."}` |
| `id`  | `{"note":"keep-alive"}` (heartbeat) |

`GET /healthz` → `200` (no auth) — liveness.

### 2.5 Configuration Contract (new env vars, additive)

| Variable | Required (gateway) | Purpose |
| --- | --- | --- |
| `GATEWAY_ADDR` | No (default `:8080`) | listen address |
| `GATEWAY_API_TOKEN` | Yes when gateway runs | primary bearer token (ADR-0004) |
| `GATEWAY_API_TOKENS` | No | extra accepted tokens for rotation |
| `GATEWAY_MAX_CONCURRENT` | No (default `8`) | in-flight admission cap → `429` |

All existing host vars (`GEMINI_*`, `MCP_*`, `LLM_TIMEOUT`, …) are reused as-is.

### 2.6 Per-Request Orchestration (Sequence)

One MCP server per request (ADR-00005); context propagates cancellation so a
client disconnect closes the child process (AC4). `agent.Event` maps to SSE on
the fly.

```mermaid
sequenceDiagram
    participant C as Client (fetch/curl)
    participant H as internal/server (SSE handler)
    participant R as Runner
    participant A as internal/agent
    participant L as internal/llm (Gemini adapter)
    participant M as internal/mcpclient
    C->>H: POST /v1/chat (Bearer)
    H->>H: auth (subtle) → 401 or OK
    H->>H: validate body → 400/422 or OK
    opt admitted
        H-->>C: 200, text/event-stream
        H-->>C: event:start
        H->>R: Run(ctx, prompt)
        R->>M: NewStdio(policy) → Initialize → ListTools (MCP_TIMEOUT)
        R->>R: mapping.MapTools → schemas; build Gemini + mcpToolCaller
        R-->>A: agent.New(Options{LLM, Tools, OnEvent})
        loop iterations
            A->>L: Generate (LLM_TIMEOUT)
            A-->>H: EventText → event:stream (text)
            A->>M: CallTool (MCP_TIMEOUT)  [A-->>H: EventTool → event:stream]
            M-->>A: llm.ToolResult
        end
        A-->>R: final string (EventFinal)
        R-->>M post-close
        H-->>C: event:final {"text": ...}
        H-->>C: close stream
    end
    opt failure after start
        H-->>C: event:error {code,message}
        H-->>C: close
    end
    opt client disconnect / cancel
        C->>A: req ctx canceled
        A-->>R: context.Canceled → R-->>M Close → H close (no orphan)
    end
```

### 2.7 Error Model & SSE boundary

- Errors detected before `start`: JSON envelope + HTTP status (audit-safe — no
  secrets).
- Errors after `start`: the `error` SSE event with a machine code (`BAD_GATEWAY`,
  `LLM_FAILURE`, `MCP_ERROR`, `ITERATION_LIMIT`, `CANCELLED`), `message` is
  safe/non-secret (ADR-0001 D3). Streams always end in exactly one terminal
  event.
- Cancellation: stop, `defer client.Close()`, emit `error CANCELLED` if the
  socket is still writable (best-effort), then return.

---

## 3. Alternatives Considered

| Option | Pros | Cons | Why Not → Selection |
| --- | --- | --- | --- |
| **Stdlib `net/http` + hand-rolled SSE** | zero deps, testable, HTTP/2 | manual flush | Selected (ADR-00002) |
| WebSocket | stream-friendly, browser-native | bidirectional overkill, non-success status | Rejected (ADR-00002) |
| Third-party SSE lib | saves ~20 lines | new dep for trivial framing | Rejected |
| Static bearer token (env) | minimal, constant-time, no DB | shared-secret rotation | Selected (ADR-00004) |
| OAuth2 / per-user JWT | per-user control | no user model exists | Rejected (ADR-00004) |
| Per-request MCP lifecycle | isolation (AC4), Fresh schema | per-request spawn cost | Selected (ADR-00005) |
| Pooled/shared MCP process | lower latency | coupled roots/state, global crash | Rejected for MVP |
| Optional `agent.Options.OnEvent` | backward compatible, AC5-safe | sync callback surface | Selected (ADR-00006) |
| Gateway re-implements the loop | no agent touch | duplicates orchestration (D1) | Rejected (ADR-00006) |

---

## 4. Risk & Mitigation

| Risk | Impact | Prob. | Mitigation |
| --- | --- | --- | --- |
| Child-process (OS) exhaustion under concurrency | High | Med | `GATEWAY_MAX_CONCURRENT` admission → `429` (ADR-00005) |
| Orphaned MCP on disconnect/crash | Medium | Med | per-request `defer client.Close()` + context cancel (AC4) |
| SSE delivery to proxy buffering (Nginx/etc.) | Medium | Med | `Cache-Control: no-cache`, `X-Accel-Buffering: no`, periodic `id` heartbeat |
| Token leak via logs / browser bundle | High | Med | strip headers in logs; frontend via server-side proxy only (ADR-00004) |
| Observer panic kills loop | Low | Low | best-effort `recover()` in OnEvent (ADR-00006) |
| Per-request discovery latency | Low | High | acceptable for P2; revisit with pooled MCP transport later |
| Division of terminal state ambiguous | Low | Low | ADR-00003: exactly one `final`/`error`, then close |

---

## 5. Rollout Plan

- **Phase 0 — Design (this TDD).** Approve TDD + ADRs 00002–00006.
- **Phase 1 — Agent observer seam.** Add `OnEvent` to `internal/agent` (additive,
  nil-safe, recover). Verify: all existing tests pass unchanged.
- **Phase 2 — `internal/server` HTTP layer.** Router + auth middleware +
  `Handler` seam + SSE encoder + `healthz` + `GATEWAY_*` config. Unit-tested with
  a fake `Handler` (no dep on mcp-go/genai).
- **Phase 3 — `internal/server` runner (real composition).** Per-request MCP
  lifecycle, event mapping, cancellation handling. Integration with fake LLM +
  `internal/mcpclient` fakeserver.
- **Phase 4 — `cmd/gateway`.** Entrypoint wiring, `:8080`, `health()`, docs.
- **Phase 5 — hardening (later feature/backlog):** Next.js UI (separate FEAT),
  token auto-reload, per-client rate limit, pooled MCP transport.

**Rollback:** gateway is additive; `cmd/server` and `internal/agent` CLI behavior
are unaffected. A defect simply stops shipping the new binary — no migration
needed.

---

## 6. Acceptance Criteria

| ID | Criterion | Verification (automated where feasible) |
| --- | --- | --- |
| GW1 | `POST /v1/chat` with a valid token returns `text/event-stream` starting with `start` and ending with `final` containing the CLI-equivalent answer. | HTTP integration test vs a fake Runner + fake LLM |
| GW2 | Missing/invalid token → `401` JSON, no SSE frames emitted. | auth unit test |
| GW3 | Malformed body → `400`; empty prompt → `422`; no SSE. | handler unit test |
| GW4 | Tool progress is interleaved as `stream` events before `final`. | runner + fake MCP test asserting event order |
| GW5 | Client disconnect cancels the agent and closes the MCP child (no orphan). | fake LLM blocking on ctx; assert `Close` called on cancel |
| GW6 | MCP init/discover/call failure after `start` surfaces as `error` event; process stays up for the next request. | fakeserver failure test |
| GW7 | `cmd/server` CLI behavior unchanged (existing suite green). | `go test ./...` after Phase 1 |
| GW8 | HTTP layer tests contain no `mcp-go`/`genai` — SDK-free `Handler` seam. | package import graph in HTTP test files |
| GW9 | Per-request isolation: two concurrent requests run independent MCP lifecycles. | concurrent httptest + two fakeservers |

---

## 7. Test Strategy

- **Unit:** auth middleware (constant-time, missing/multi-token), body parsing
  and validation, SSE frame encoder, event mapping (`agent.Event`→SSE), admission
  guard.
- **Integration:** `internal/server` HTTP layer with a **fake `Handler`** (no
  deps); runner lifecycle with a fake MCP `fakeserver` + fake LLM leaf; recovery
  on init/list/call error; cancel propagation.
- **E2E (opt-in):** real Gemini + sandboxed Filesystem MCP server behind
  credentials — never a developer working directory.
- **Quality gates (per `rules/testing.md`):** `go test ./...`, `go vet ./...`,
  coverage report; new runner/HTTP logic ≥80% statement coverage. Existing
  suites must remain green after Phase 1.

---

## 8. Frontend Scope Decision

**Next.js UI is OUT of scope for FEAT-00008.** The `/think` verdict lists a
"thin Next.js page," but the gateway contract (ADR-00002/03) must stabilize and
be tested first; and the browser must send an `Authorization` header, so the UI
requires a server-side Next.js route handler / proxy (or BFF) to keep the token
out of the browser bundle (ADR-00004). YAGNI applies — shipping the SSE contract
now lets a UI be built later against a frozen contract. Track as a follow-up
**FEAT-XXXXX (Next.js UI)** in S2; this TDD freezes the contract it depends on.

---

## 9. Recommended Build Sequence

1. **Phase 1** — `internal/agent` `OnEvent` observer (TDD: an integration test
   covers event order with a fake LLM; then implement; existing suites stay green).
2. **Phase 2** — `internal/config` GATEWAY_* keys + tests.
3. **Phase 3** — `internal/server` HTTP layer + auth + `POST /v1/chat` SSE +
   `healthz` + `Handler` seam (fake-tested, SDK-free).
4. **Phase 4** — `internal/server` runner: per-request MCP lifecycle + event
   mapping + cancel (integration with fakeserver).
5. **Phase 5** — `cmd/gateway` entrypoint + docs (rollout behind a flag envvar).
6. **(Later, separate feature)** — Next.js UI/proxy; token rotation; rate
   limiting; pooled MCP transport.

**Guard-rail:** keep `cmd/server/main.go`, `internal/agent` Run signature,
`internal/llm`/`mcpclient` interfaces unchanged. Only additive changes to
`internal/agent` (OnEvent) are allowed.

---

## 10. Open Questions

- [ ] Confirm tokens are static vs. allow future identity integration (ADR-00004
      notes exhaustion options).
- [ ] Should `stream` also carry the bounded tool **result text**? Currently it
      carries only status + byte count (to avoid echoing secrets); confirm.
- [ ] Timeout profile for the whole request (Σ iterations × (LLM+MCP_TIMEOUT));
      decide a global `GATEWAY_REQUEST_TIMEOUT` default.
- [ ] `GATEWAY_MAX_CONCURRENT` default value (8) — confirm vs. target QPS.

---

## 11. References

- `docs/tdd/MCP_HOST_TDD.md` (§2.1, §5, §8, §9, AC4/AC5)
- `docs/adr/ADR-0001-internal-agent-architecture.md` … `ADR-0006-agent-observation-seam.md`
- `cmd/server/main.go` (composition root, `mcpToolCaller`, `neutralSchema`)
- `internal/agent/agent.go`, `internal/llm/interface.go`, `internal/mcpclient`,
  `internal/config`, `internal/policy`, `internal/mapping`
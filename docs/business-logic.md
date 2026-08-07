# Business Logic - Custom MCP Host

## 1. Overview

Custom MCP Host la mot orchestrator headless ket noi Gemini voi MCP Server. MVP chi ho tro Filesystem MCP qua `stdio` va nhan prompt tu CLI. Host khong tu thuc thi shell command, SQL, hay file operation theo du lieu do LLM cung cap; moi thao tac phai di qua MCP tool da duoc discovery va kiem tra policy.

## 2. Core Workflow

1. User goi CLI voi `--prompt` hoac gui mot prompt qua stdin.
2. Host kiem tra configuration, khoi tao request context, va spawn Filesystem MCP Server da duoc cau hinh.
3. Host hoan thanh initialize handshake va goi `tools/list`.
4. Host map schema MCP tool sang Gemini function declarations.
5. Host gui prompt, conversation history, va tool declarations toi Gemini.
6. Neu Gemini tra ve tool call, Host validate ten tool va arguments, sau do goi `tools/call` qua MCP.
7. Host dua tool result da gioi han kich thuoc, hoac loi an toan, vao conversation history va lap lai buoc 5.
8. Khi Gemini tra text ma khong co tool call, Host in text do ra stdout va dong MCP process.

## 3. Agent State and Limits

| State | Meaning | Exit or transition |
|---|---|---|
| `ConfigValidation` | Validate secrets, command, roots, and limits. | Invalid config returns a non-zero exit code. |
| `MCPConnecting` | Spawn process and initialize MCP. | Success goes to `ToolDiscovery`; error goes to `Failed`. |
| `ToolDiscovery` | Request and map `tools/list`. | Success goes to `LLMInference`; unsupported schema goes to `Failed`. |
| `LLMInference` | Send history and tools to Gemini. | Text goes to `Completed`; tool call goes to `ToolExecution`. |
| `ToolExecution` | Validate and call a discovered MCP tool. | Append result/error, then return to `LLMInference`. |
| `Completed` | Return final text. | Close MCP process. |
| `Failed` | Return a concise user-safe error. | Close MCP process. |

The host enforces the following per prompt:

- A parent context covers the complete invocation.
- Every Gemini request uses `LLM_TIMEOUT`.
- Initialize, discovery, and every MCP call use `MCP_TIMEOUT`.
- The loop cannot exceed `AGENT_MAX_ITERATIONS`.
- Each tool result copied into history cannot exceed `MCP_MAX_RESULT_BYTES`.

## 4. Business Rules

- Only the configured Filesystem MCP executable may be spawned. The LLM cannot choose the executable, process arguments, or working directory.
- Only tools returned by `tools/list` may be called.
- Tool arguments must conform to the discovered schema before they are sent to MCP.
- Filesystem access is restricted to configured canonical roots. A path outside those roots is denied.
- Tool errors, MCP crashes, invalid protocol messages, and timeouts must be turned into safe bounded context for Gemini. The host must remain running and clean up child processes.
- API keys, raw sensitive prompts, authorization data, stack traces, and unbounded tool output must not be logged or fed back to users.
- The MVP handles one MCP server per invocation. Concurrent multi-server execution is deferred until each server has an explicit policy and conflict-resolution design.

## 5. Error Handling Contract

| Failure | Host behavior | Context sent to Gemini |
|---|---|---|
| Invalid configuration | Stop before spawning a process; return non-zero. | None. |
| MCP startup or initialize failure | Close process and return a safe error. | None for initial discovery. |
| `tools/list` failure or unsupported schema | Close process and return a safe error. | None for initial discovery. |
| Unknown or unauthorized tool call | Do not call MCP. | A tool result stating the action is unavailable. |
| Invalid tool arguments | Do not call MCP. | A tool result stating arguments failed validation. |
| MCP tool error, crash, or timeout | Preserve host process; append bounded error result. | A concise error code and retry-safe message. |
| Gemini timeout or provider error | Close process and return a safe error. | None. |
| Iteration limit reached | Close process and return a safe error. | None after the limit is reached. |
| Caller cancellation | Cancel LLM/MCP work, close process, and return cancellation status. | None. |

## 6. Acceptance Criteria

- **AC1 - Filesystem connection:** Given a valid Filesystem MCP command and sandbox root, the host completes MCP initialize within `MCP_TIMEOUT`.
- **AC2 - Discovery and mapping:** The host returns discovered Filesystem tools and maps supported JSON Schema fields, including `required`, nested properties, arrays, and enums, to Gemini declarations.
- **AC3 - Complete agent loop:** Given a Gemini response requesting a discovered Filesystem tool, the host validates and executes the call, adds its result to history, and returns the subsequent final Gemini text.
- **AC4 - Safe failure:** Given an MCP tool error, timeout, or child-process crash, the host exits the invocation cleanly without panic and does not leave a child process running.
- **AC5 - Extensible architecture:** `internal/agent` uses MCP and LLM interfaces only; fake implementations can exercise the complete loop without `mcp-go` or the Gemini SDK.

## 7. Out of Scope for MVP

- HTTP API, SSE streaming, Next.js UI, and authentication for remote callers. *(Superseded for the gateway surface by §8–§10 of FEAT-00008; the below out-of-scope items remain.)*
- Multiple concurrent MCP servers.
- PostgreSQL, Git/SVN, Docker/CLI, and remote HTTP/SSE MCP transports.
- DeepSeek and Ollama providers.

---

## 8. Gateway User Stories (FEAT-00008)

> Scope: authenticated HTTP surface over the existing bounded agent loop. The CLI
> entrypoint and `internal/agent` behavior are unchanged (`ADR-00001`).

- **As a programmatic client,** I want to `POST /v1/chat` with a Bearer token and
  a prompt, so that I can drive the agent remotely.
- **As a programmatic client,** I want to receive a typed, ordered SSE sequence
  (`start → stream* → final`), so that I can render progress and a final answer.
- **As a browser UI (future Next.js),** I want the streamed contract to be frozen,
  so that a UI built later depends on a stable SSE interface.
- **As an operator,** I want the gateway authenticated and concurrency-capped, so
  that an unauthorized or excessive caller cannot consume LLM/MCP resources.
- **As an operator,** I want `GET /healthz` to be unauthenticated, so that
  load-balancer liveness probes work.
- **As a user,** I want my request cancelled cleanly on disconnect, so that no
  orphaned MCP child process is left behind.

## 9. Gateway Business Rules (FEAT-00008)

### 9.1 Authentication (ADR-00004)

- `R1`: `POST /v1/chat` REQUIRES `Authorization: Bearer <token>`.
- `R2`: Accepted tokens come ONLY from environment (`GATEWAY_API_TOKEN`, plus
  optional `GATEWAY_API_TOKENS` for rotation). Tokens are never in code, logs,
  or responses.
- `R3`: Token comparison is constant-time (`crypto/subtle`) against every
  configured token; any mismatch → `401`.
- `R4`: `GET /healthz` is unauthenticated.
- `R5`: A `request_id` is assigned before auth for correlation; used in `id:`,
  error `trace_id`, and logs.

### 9.2 Concurrency Guard (TDD §2.5, §4)

- `R6`: `POST /v1/chat` admits at most `GATEWAY_MAX_CONCURRENT` in-flight
  requests (default `8`); requests beyond the cap get `429` before any SSE.
- `R7`: The guard limits concurrent child MCP processes to protect the OS from
  exhaustion.

### 9.3 Per-request MCP lifecycle (ADR-00005)

- `R8`: Each `POST /v1/chat` spawns ONE stdio MCP server; no pooling, no
  cross-request reuse, no in-flight sharing.
- `R9`: The full lifecycle (spawn → Initialize within `MCP_TIMEOUT` → ListTools
  once → map schemas → agent loop → Close) runs inside the request context.
- `R10`: The MCP client is ALWAYS closed — on success, failure, OR cancellation
  (`defer client.Close()` on every path).

### 9.4 Cancellation semantics (ADR-00005, TDD AC4)

- `R11`: Request context propagates cancellation; a client disconnect stops the
  agent and closes the MCP child.
- `R12`: On cancel, if the socket is still writable, emit best-effort
  `event:error {"code":"CANCELLED",...}` then close.

### 9.5 Message streaming & event mapping (ADR-00003, ADR-00006)

- `R13`: Agent emits `start` once, then per iteration `text` (if non-empty) and
  `tool` before/after each sequential tool call, then `final` on success.
- `R14`: `internal/server` maps `agent.Event` → SSE frames on the fly via the
  optional `OnEvent` observer; the agent loop itself is NOT re-implemented.
- `R15`: Terminal state is deterministic: exactly one of `final` or `error`
  per stream, after which the stream closes.
- `R16`: `stream` tool events carry `name` + `status` (`start`/`result`) +
  bounded `bytes`; they do NOT echo tool result text (avoids leaking secrets).
- `R17`: `OnEvent` runs with best-effort `recover()`; a panicking observer must
  not abort the agent loop.

### 9.6 Validation (error-handling, api-convention)

- `R18`: Auth + body validation ALWAYS complete before any SSE. Failures return
  a JSON error envelope with `error.code` + `error.message` and the HTTP status
  (`401`/`400`/`422`/`429`).
- `R19`: Empty or whitespace `prompt` → `422`; malformed JSON → `400`.
- `R20`: Post-`start` failures use the in-stream `error` event with a
  machine code (`BAD_GATEWAY`/`LLM_FAILURE`/`MCP_ERROR`/`ITERATION_LIMIT`/
  `CANCELLED`) and a safe, non-secret `message`.

### 9.7 CLI invariance

- `R21`: `cmd/server` passes no `OnEvent`; `internal/agent` `Run` signature and
  CLI behavior are byte-for-byte unchanged — all pre-existing tests stay green.

## 10. Gateway Acceptance Criteria — FEAT-00008 (GW1..GW9)

| ID | Criterion |
|----|-----------|
| GW1 | Valid token → `text/event-stream` starting with `start` and ending with `final` containing the CLI-equivalent answer. |
| GW2 | Missing/invalid token → `401` JSON, no SSE frames emitted. |
| GW3 | Malformed body → `400`; empty `prompt` → `422`; neither emits SSE. |
| GW4 | Tool progress appears as ordered `stream` events before `final`. |
| GW5 | Client disconnect cancels the agent and closes the MCP child (no orphan). |
| GW6 | MCP init/discover/call failure after `start` → `error` event; process stays up for the next request. |
| GW7 | `cmd/server` CLI behavior unchanged (existing test suite green). |
| GW8 | HTTP-layer tests contain no `mcp-go`/`genai` (SDK-free `Handler` seam). |
| GW9 | Two concurrent requests run independent MCP lifecycles (per-request isolation). |

## 11. Frontend Scope & Browser Token Caveat (FEAT-00008)

- **Next.js UI is explicitly OUT of scope for FEAT-00008.** The gateway contract
  (`ADR-00002/03`) must freeze and be tested first.
- **Token-proxy caveat:** SSE `EventSource` cannot set an `Authorization` header,
  and embedding the Bearer token in a browser bundle is a security risk
  (`ADR-00004`). The future UI MUST call the gateway through a server-side
  Next.js route handler / proxy (BFF) that owns the token and uses
  `fetch` + `ReadableStream`, never `EventSource` directly from the browser.
- Tracked as follow-up **FEAT-XXXXX (Next.js UI)** in Sprint 2.

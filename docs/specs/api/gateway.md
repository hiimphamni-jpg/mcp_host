# v1 Chat Gateway API

> Owner: BA | Last Updated: 2026-08-07
> Related Tasks: FEAT-00008, ADR-00002, ADR-00003, ADR-00004, ADR-00005, ADR-00006

## Overview

The gateway exposes the existing bounded agent loop as an authenticated HTTP
surface. A client sends one chat prompt as JSON and receives lifecycle +
progress events streamed over **SSE**, ending in exactly one terminal event
(`final` or `error`). Auth and body validation always happen **before** any SSE
frame is written, so pre-stream failures return a normal JSON error envelope
with an HTTP status.

- **Transport:** stdlib `net/http`, hand-rolled SSE (`ADR-00002`)
- **Event contract:** `ADR-00003` (`start` → `stream`* → `final`|`error`)
- **Auth:** static Bearer token (`ADR-00004`)
- **Run model:** per-request MCP lifecycle (`ADR-00005`), agent observer seam (`ADR-00006`)

---

## Endpoints

### `POST /v1/chat`

- **Description:** Accept a chat prompt, run the bounded agent loop, and stream
  lifecycle + progress events over SSE, ending in a final answer or a streamed
  error.
- **Auth:** `Authorization: Bearer <token>` (required, `ADR-00004`).
- **Accept:** `text/event-stream` (request body is `application/json`).
- **Request Body:**
  ```json
  {
    "prompt": "<string>"
  }
  ```

**Pre-SSE errors (JSON envelope, HTTP status — no SSE frames emitted):**

| Case                        | Status | `error.code`       |
|-----------------------------|:------:|--------------------|
| Missing/invalid bearer token | `401`  | `UNAUTHORIZED`    |
| Malformed JSON body          | `400`  | `INVALID_FORMAT`  |
| Missing `prompt` field       | `400`  | `VALIDATION_FAILED` |
| Empty / whitespace prompt    | `422`  | `INVALID_CONTENT` |
| Over concurrency cap         | `429`  | `TOO_MANY_REQUESTS` |

**SSE stream (`200` + `Content-Type: text/event-stream`):**

Frames use the SSE wire format `event:<type>\ndata:<json>\n\n`, with a monotonic
`id:` per event for client-side ordering (no Last-Event-ID replay in MVP).

| Event   | `data:` payload | Meaning |
|---------|-----------------|---------|
| `start` | `{"request_id":"<id>","model":"<name>"}` | Accepted; streaming begins. Emitted once, first. |
| `stream` | `{"type":"text","text":"..."}` OR `{"type":"tool","name":"<tool>","status":"start"\|"result","bytes":<n>}` | Optional progress; zero or more, in order. |
| `final` | `{"text":"<final answer>"}` | Successful terminal event; then close. |
| `error` | `{"code":"<CODE>","message":"<safe>"}` | Terminal failure after streaming began; then close. |
| `id` | `{"note":"keep-alive"}` | Heartbeat when idle; no state change. |

**In-stream `error.code` values (post-`start`):** `BAD_GATEWAY`, `LLM_FAILURE`,
`MCP_ERROR`, `ITERATION_LIMIT`, `CANCELLED`. `message` must be human-readable
and non-secret (no stack traces, no raw tool/config output).

**Terminal-state rule (`ADR-00003`):** a stream always emits **exactly one** of
`final` or `error`, after which the handler returns and the stream closes.

#### Example — successful stream

Request:
```http
POST /v1/chat
Authorization: Bearer <token>
Content-Type: application/json
Accept: text/event-stream

{"prompt":"List the files under /tmp"}
```

Streamed response:
```
event: start
id: 1
data: {"request_id":"1f3e...","model":"gemini-..."}

event: stream
id: 2
data: {"type":"tool","name":"read_dir","status":"start","bytes":0}

event: stream
id: 3
data: {"type":"tool","name":"read_dir","status":"result","bytes":184}

event: stream
id: 4
data: {"type":"text","text":"The /tmp directory contains:"}

event: final
id: 5
data: {"text":"<final answer>"}
```

#### Error cases (detail)

| Case | Where detected | Response |
|------|----------------|----------|
| Missing/invalid token | before SSE | `401` JSON |
| Malformed JSON / missing `prompt` | before SSE | `400` JSON |
| Empty/whitespace prompt | before SSE | `422` JSON |
| Over `GATEWAY_MAX_CONCURRENT` | before SSE | `429` JSON |
| MCP init/discover failure after `start` | in-stream | `event:error` `MCP_ERROR`, then close |
| LLM/provider failure after `start` | in-stream | `event:error` `LLM_FAILURE`, then close |
| Iteration limit reached | in-stream | `event:error` `ITERATION_LIMIT`, then close |
| Client disconnect / cancel | in-stream | cancel work, `defer client.Close()`, best-effort `event:error` `CANCELLED`, then close |

---

### `GET /healthz`

- **Auth:** none.
- **Response `200`:** liveness probe, `{"status":"ok"}`.
- **Purpose:** no-auth liveness for load-balancer / orchestrator probes.

---

## Response Security

- Failures before `start` use the JSON error envelope with `error.code`
  (machine-readable) + `error.message` (human-readable) — no secrets.
- Streamed `error` messages are sanitized (`ADR-0001` D3); raw provider/MCP
  errors are never sent to the client.
- Proxies: `Cache-Control: no-cache`, `X-Accel-Buffering: no`, plus periodic
  `id` heartbeat to defeat server buffering.

---

Spec Writing Checklist:
- [x] Clear versioning (`/v1/...`)
- [x] Consistent error envelope (`error.code` + `error.message`)
- [x] Error codes follow `rules/api-convention.md` / `rules/error-handling.md`
- [x] `Authorization` header defined
- [x] Terminal-state deterministic (`final`/`error`)
- [x] Single module; no pagination needed (one-shot request/response + stream)
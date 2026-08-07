# ADR-00003: SSE Event Contract for `POST /v1/chat`

- **Status:** Proposed
- **Date:** 2026-08-07
- **Decision makers:** Architect (Flow 1 Design — FEAT-00008)
- **Related feature:** FEAT-00008 (HTTP/SSE API Gateway)

## Context

The gateway streams one chat request as SSE. Clients need a stable, unambiguous
boundary between the request lifecycle phases: accepted, in-flight progress, and
final outcome. The agent loop emits terminal and intermediate events that must
map onto a small, versioned event vocabulary.

## Decision

Framing follows the SSE wire format: `event:<type>\ndata:<json>\n\n`, with a
monotonic `id:` per event for client-side ordering (not used for Last-Event-ID
replay in MVP). Four client-visible event types are defined, plus a `ping`
keep-alive:

| Event | Payload (`data:`) | Meaning |
| --- | --- | --- |
| `start` | `{"request_id":"<id>","model":"<name>"}` | Request accepted; streaming begins. Emitted once, first. |
| `stream` | `{"type":"text","text":"..."}` **or** `{"type":"tool","name":"<tool>","status":"start"\|"result","bytes":<n>}` | Optional progress; zero or more, in order. |
| `final` | `{"text":"<final answer>"}` | Successful terminal event; close. |
| `error` | `{"code":"<CODE>","message":"<safe>"}` | Terminal failure after streaming began; close. |
| `id` | `{"note":"keepalive"}` | Heartbeat when idle; no state change. |

Terminal events are exactly one of `final` or `error`; after it, the handler
returns and the stream closes. Before `start` is emitted, failures use the
standard JSON error envelope and a normal HTTP status (401/400/422/500), not SSE.

Event names are lowercase past-tense/imperative by convention from
`SKILL_ARCHITECT.md` §4; per-request granularity is deliberately coarse (`stream`
carries progress, not token deltas) because `internal/agent` calls the
synchronous `llm.LLM.Generate` per iteration.

## Rationale

- Versioned, self-describing payloads (`type:` discriminator inside `stream`)
  let a client render tool progress without parsing the whole stream.
- Splitting terminal `final` vs `error` gives a single, checkable terminal
  condition and an HTTP-visible status for gateways/proxies.
- `start` doubles as the "auth+validation passed" signal, making partial delivery
  unambiguous.

## Consequences

### Positive

- Deterministic terminal state: a client needs only to watch for `final`/`error`.
- `stream` events let the Next.js UI show "running `read_file`…" without any
  per-token contract.
- Errors emitted after streaming carry a machine-readable `code` for `internal`
  diagnosis (mapped from typed agent/LLM/MCP errors, never raw secrets).

### Negative

- Not token-streaming; clients that want token-level deltas must wait for a
  future LLM interface extension.
- Client renderer must tolerate arbitrary progress; payload schema inside
  `stream` is versioned via the `type` subfield and may grow additive fields.

### Neutral

- `request_id` generated with `crypto/rand` (hex) per request; used in `id:` and
  error correlation. Not a dependency addition.

## Alternatives Considered

1. **Single envelope, no event types** — lost lifecycle boundary; rejected.
2. **Token-level streaming event (`delta`)** — requires a streaming LLM seam not
   present today; rejected for MVP (deferred, see ADR-00006).
3. **NDJSON over a plain `200` body** — fine for CLI, less standard for the
   frontend and proxies; `event:` framing chosen for explicit SSE semantics.

## References

- `docs/adr/ADR-0002-http-sse-transport.md`
- `docs/adr/ADR-0004-bearer-token-auth.md`
- `docs/adr/ADR-0005-per-request-mcp-lifecycle.md`
- `docs/adr/ADR-0006-agent-observation-seam.md`
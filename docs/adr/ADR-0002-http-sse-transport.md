# ADR-00002: HTTP/SSE Transport for the v1 Chat Gateway

- **Status:** Proposed
- **Date:** 2026-08-07
- **Decision makers:** Architect (Flow 1 Design — FEAT-00008)
- **Related feature:** FEAT-00008 (HTTP/SSE API Gateway)

## Context

FEAT-00008 exposes the existing bounded agent loop (FEAT-00006) over an HTTP
gateway, with a `POST /v1/chat` endpoint that returns streaming progress and a
final answer. The MVP transport must:

- Deliver a single logical response progressively (final text is only known
  after an arbitrary number of LLM↔MCP iterations).
- Work for a thin browser frontend (Next.js) and for programmatic clients.
- Dramatically undercut the cost of adding a new dependency or service.

The MVP criteria in `docs/tdd/MCP_HOST_TDD.md` §1 deliberately excluded HTTP;
this ADR settles the transport before implementation (§9).

## Decision

Use the Go standard library `net/http` to serve `POST /v1/chat` and hand-roll an
**SSE response** (`Content-Type: text/event-stream`) by writing the
`event:`/`data:` framing directly to the `http.ResponseWriter` and calling
`Flush()` via the optional `http.Flusher` interface. No third-party SSE or
streaming library is added.

The response lifecycle is:

1. The HTTP request handler authenticates and validates the body **before** any
   SSE is begun. Failures here return a normal JSON error envelope.
2. Once validated, the handler commits to SSE: `Content-Type: text/event-stream`,
   then writes `start`, zero or more `stream`, then `final` (or `error`), then
   returns.

A `GET /healthz` returns `200` for liveness without authentication.

## Rationale

- SSE is served by plain `net/http` for both HTTP/1.1 and HTTP/2, so the browser
  `EventSource`-style `fetch` + `ReadableStream` pattern and curl both work with
  no new transport layer.
- The one-shot LLM response model does not need the bidirectional upgrade,
  message-type framing, or connection lifecycle that WebSocket provides.
- The framing is ~20 lines and fully unit-testable with an `httptest.Server`
  and `http.Flusher`.

## Consequences

### Positive

- Zero new runtime dependency for the transport; stdlib is version-guaranteed.
- SSE is trivially readable in tests, curl, and browser devtools.
- HTTP status codes (401/400/422/500) remain available for pre-stream errors.

### Negative

- No token-level streaming yet: the current `llm.LLM.Generate` is
  request/response, so `stream` events carry *progress* (tool activity, model
  text fragments per iteration) rather than raw token deltas. Token streaming is
  a later LLM-interface extension (ADR-00006 §Neutral).
- SSE over HTTP/1.1 needs `Flush` and works best over a single connection; server
  buffering proxies must be aware (`Cache-Control: no-cache`).
- Long-lived connections consume a goroutine per in-flight request.

### Neutral

- EventSource (browser) cannot set `Authorization` headers; the gateway uses a
  plain `fetch` + `ReadableStream`, which can carry the header via request init.
  This makes the Next.js frontend depend on `fetch` streaming, not `EventSource`.

## Alternatives Considered

1. **WebSocket (`gorilla/websocket`)** — already an indirect dependency, but
   bidirectional upgrade is overkill for one-shot LLM responses, complicates
   auth/status handling, and is unnecessary for a `fetch`-based frontend.
   Rejected.
2. **Third-party SSE library** (e.g. `sses`, `eventsource`) — saves ~20 lines but
   adds a maintained dependency for trivial framing. Rejected for KISS.
3. **Long-polling / chunked plain body** — no structured event boundaries, harder
   for clients to delimit progress vs final. Rejected.

## References

- `docs/tdd/MCP_HOST_TDD.md` §8 (HTTP/SSE deferred), §9 (document before implement)
- `docs/adr/ADR-0003-sse-event-contract.md`
- `docs/adr/ADR-0004-bearer-token-auth.md`
- `docs/adr/ADR-0005-per-request-mcp-lifecycle.md`
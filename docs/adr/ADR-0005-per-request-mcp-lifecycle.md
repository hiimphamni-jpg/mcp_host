# ADR-00005: Per-Request MCP Lifecycle in the Gateway

- **Status:** Proposed
- **Date:** 2026-08-07
- **Decision makers:** Architect (Flow 1 Design — FEAT-00008)
- **Related feature:** FEAT-00008 (HTTP/SSE API Gateway)

## Context

The CLI path (FEAT-00007) spawns one stdio MCP server per prompt invocation and
closes it on every exit path (`cmd/server/main.go` §5, `MCP_HOST_TDD.md` §5).
The gateway must decide whether to reuse that lifecycle across concurrent HTTP
requests or own a long-lived process pool. The Filesystem MCP server holds a
canonical allowed-roots policy (`internal/policy`) and mutable working state, so
sharing a process across requests couples their filesystem views and failure
domains.

## Decision

**One stdio MCP server per HTTP request**, for every `POST /v1/chat`. The full
lifecycle (validate policy → spawn → `Initialize` within `MCP_TIMEOUT` →
`ListTools` once → map schemas → agent loop → `Close`/terminate) executes inside
the request context and **always** closes the client, on success, failure, or
caller cancellation — mirroring `cmd/server/main.go` (`AC4`). No process pool,
no cross-request reuse, no in-flight sharing.

Concurrency is bounded by the OS process limit; a `GATEWAY_MAX_CONCURRENT`
admission guard (default e.g. 8) is added as hardening so one user cannot
exhaust process slots (see TDD §4). Requests beyond the cap receive HTTP `429`
before any SSE.

## Rationale

- Isolation: each request gets an independent Filesystem view and independent
  failure domain; a crashed MCP server cannot poison a later request.
- Cleanliness: discovery + mapping stay per-request, so tool schemas can never
  be stale across requests; the composition mirrors the CLI exactly (TDD §2.4).
- Simplicity: no connection-pool lifecycle, liveness, or statefulness to manage
  in the MVP.

## Consequences

### Positive

- Correct-by-construction cleanup: `defer client.Close()` on every path (AC4
  holds for the gateway).
- Cancellation propagation is free: the request context drives the whole
  lifecycle, so a client disconnect closes the MCP child (TDD §2.4).
- Sequential tool execution (ADR-0001 D4) still holds; no shared state.

### Negative

- Spawn + initialize + discover cost is paid per request (tens of ms to low
  seconds) before the first LLM call; acceptable for P2 MVP, revisit if latency
  becomes the bottleneck.
- Each concurrent request consumes one child process; requires the admission
  guard and OS limit awareness.

### Neutral

- A future pooled transport (Streamable HTTP MCP servers) can be added behind
  the `mcpclient.Client` seam without changing `internal/agent` or the gateway
  HTTP layer (AC5).

## Alternatives Considered

1. **Long-lived shared MCP process** — couples allowed-roots/state across
   requests and turns a crash into a global failure; rejected.
2. **Connection pool with affinity** — significant lifecycle complexity for an
   MVP that is single-operator, low-QPS; rejected (YAGNI).
3. **Keep-alive reuse with per-request isolation** — not possible with a
   stateful Filesystem server without protocol changes; rejected.

## References

- `docs/tdd/MCP_HOST_TDD.md` §5 (MCP Client Lifecycle), §8 (Security Policy)
- `docs/adr/ADR-0001-internal-agent-architecture.md`
- `docs/adr/ADR-0003-sse-event-contract.md`
- `docs/adr/ADR-0006-agent-observation-seam.md`
# ADR-00006: Agent Observation Seam for Gateway Streaming

- **Status:** Proposed
- **Date:** 2026-08-07
- **Decision makers:** Architect (Flow 1 Design — FEAT-00008)
- **Related feature:** FEAT-00008 (HTTP/SSE API Gateway)

## Context

The gateway must stream progress (`start` / `stream` / `final` / `error`,
ADR-00003) but `internal/agent.Run(ctx, prompt)` currently returns only the final
`string` and never exposes intermediate states (agent loop boundaries, tool-call
activity, or the non-terminal text a model emits alongside tool calls). To stream
truthfully we must observe loop progress without re-implementing the loop and
without breaking the CLI composition root (`cmd/server`) on which the project
already has green tests (ADRs 0001 D1/D4).

## Decision

Add an **optional, nil-safe observer callback** to `agent.Options`, keeping the
contract provider-SDK-free:

```go
type EventType string
const (
    EventStart  EventType = "start"   // one per Run
    EventText   EventType = "text"    // non-terminal model text fragment
    EventTool   EventType = "tool"    // progress/result of a tool call
    EventFinal  EventType = "final"   // terminal success
)

type Event struct {
    Type     EventType
    Text     string          // model text
    ToolName string          // tool events
    ToolArgs map[string]any  // tool events
    Bytes    int             // tool result bytes after bounding
}

// in Options:
OnEvent func(Event)   // optional; nil is a no-op
```

The agent emits `EventStart` once, then per iteration (`EventText` for any
non-empty `resp.Text`, `EventTool` before/after each sequential tool call —
ADR-0001 D4), then `EventFinal` on success. The observer must be **non-forcing**:
it runs synchronously with a best-effort `recover()` so a panicking consumer
cannot abort the loop.

**The existing `Run` signature and behavior are unchanged.** `cmd/server/main.go`
passes no `OnEvent`, so the CLI path is byte-for-byte unchanged and all existing
tests must remain green. Terminal error mapping stays in `internal/agent`
(ADR-0001 D3); the gateway maps success/final to `EventFinal` and errors to its
`error` SSE event.

## Rationale

- Preserves the single orchestration boundary of ADR-00001 D1: the loop stays in
  `internal/agent`; `internal/server` only *observes*.
- Backward compatible: optional callback, defaults to no-op, so AC5 (providers
  replaceable without touching the agent) and the CLI surface are unaffected.
- Keeps `internal/agent` free of `net/http` and transport concerns
  (`stream` mapping lives in `internal/server`).

## Consequences

### Positive

- Truthful, low-effort progress: SSE tool activity + text fragments with no
  duplicated loop.
- No behavior change for the CLI (test suite remains green).
- The seam is transport-agnostic; a WebSocket or gRPC server could reuse it.

### Negative

- Adds a lightweight callback surface to a core package; must stay stable (any
  new field is additive to `Event`).
- Cancellation mid-iteration may yield a truncated `stream`; clients must treat
  `error`/disconnect as terminal (ADR-00003).
- No token-level streaming remains (ADR-00003 §Neutral); future.

## Alternatives Considered

1. **Gateway re-implements the agent loop** — duplicates ADR-00001's single
   orchestration boundary in two places; rejected (DRY, AC5).
2. **Change `Run` to return a stream/iterator** — a breaking signature change
   that forces the CLI to be reworked; rejected (keep `cmd/server` unchanged).
3. **Poll agent internal state** — racy and leaks data; rejected.

## References

- `docs/tdd/FEAT-00008_gateway_tdd.md` §2.3, §4, §6
- `docs/adr/ADR-0001-internal-agent-architecture.md` (D1, D3, D4)
- `docs/adr/ADR-0003-sse-event-contract.md`
- `docs/adr/ADR-0005-per-request-mcp-lifecycle.md`
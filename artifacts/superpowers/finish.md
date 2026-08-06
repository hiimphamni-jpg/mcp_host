# /finish Summary — FEAT-00006: Bounded agentic loop

## Status
- DEV: done (quality gates green)
- TEST: pending (suggest `/test cases FEAT-00006`)
- QC: pending (suggest `/qc review FEAT-00006`)
- Registry: `docs/REGISTRY.md` FEAT-00006 → DEV ✅, Status IN PROGRESS

## What shipped
- `internal/agent` (new): `Options`, `Agent`, `Run` bounded loop, `ToolCaller` seam, `New` validation, `ErrIterationLimit` + typed `agent.Error`.
- `internal/agent/schema.go`: stdlib-only neutral schema-arg validator (object/required/enum/array/number kinds), secret-safe messages.
- `internal/agent/results.go`: `BoundedResult` (JSON + truncation marker), `MessageResult`, `ErrorResult` (non-secret error mapping).
- `internal/agent/agent_test.go` + helper tests: fake-driven loop coverage incl. cancellation propagation, truncation, no-secret, iteration limit.
- `cmd/server/main.go`: composition-root wiring (policy → MCP → discovery → mapping → llm.New → agent), `--prompt` flag, `mcpToolCaller` adapter converting mcp-go results to neutral `llm.ToolResult` (ADR-0001 D2), deferred `client.Close()` on all paths (AC4).
- `cmd/server/main_test.go`: smoke test (unconfigured invocation ends cleanly, no real Gemini/MCP in CI) + adapter/schema unit tests.
- `docs/REGISTRY.md`: FEAT-00006 DEV → done.

## Verification evidence
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (all packages)
- `gofmt -l .` → empty
- `internal/agent` coverage: 91.5% (target ≥ 80%)
- No `genai`/`mcp-go` import in `internal/agent` (comments only).

## Follow-ups
- `/test cases FEAT-00006` — write acceptance-level test cases (TEST column).
- `/qc audit FEAT-00006` then `/qc sign-off FEAT-00006` — flip Global Status to DONE.
- FEAT-00007 (headless CLI stdout/exit-code contract) is the next dev dependency; until then `--prompt` is the only loop entry.

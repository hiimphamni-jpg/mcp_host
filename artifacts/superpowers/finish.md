# /build Finish Summary — FEAT-00003: Stdio MCP client lifecycle

## Status: IMPLEMENTED (ready for `/finish` QC sign-off)

## What was built
New module `internal/mcpclient` (only; no edits to config/policy/cmd/server):

| File | Purpose |
|------|---------|
| `interface.go` | `Client` interface (Initialize/ListTools/CallTool/Close) — the AC5 seam for future `internal/agent`. |
| `errors.go` | Typed sentinel errors `ErrProcessExit`, `ErrInitFailed`, `ErrDiscoveryFailed`, `ErrCallFailed`, `ErrInvalidResponse`, `ErrTimeout` with user-safe `Error`/`TimeoutError`. |
| `client.go` | `StdioClient`: spawn (argv = `policy.Args()+Roots()`, never LLM-supplied), `Initialize`, `ListTools`, `CallTool` under `MCP_TIMEOUT`, idempotent `Close`, error `classify`. |
| `client_integration_test.go` | 6 integration scenarios against a real fake stdio subprocess. |
| `testdata/fakeserver/main.go` | Tiny stdio JSON-RPC MCP server with normal/crash/timeout/error modes + exit-marker. |

## Verification evidence
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (cmd/server, config, policy, mcpclient)
- `go test ./internal/mcpclient/... -count=1 -cover` → 86.4% coverage (target ≥80%)
- `gofmt -l ./cmd ./internal` → clean
- No orphan `mcp-fake-server` processes (verified via `Get-Process` after runs)
- `cmd/server` compiles unchanged; no diff to config/policy/cmd/server

## Review pass
- 🔴 **Blockers:** none.
- 🟠 **Majors:** none.
- 🟡 **Minors:**
  - `Close` after a crashed child returns the child's exit-status error on the first call (second call returns nil — idempotent). Acceptable and documented; could be suppressed later if callers find it noisy.
  - Integration tests build the fake server with `go build` in `TestMain` (~0.3s amortized); acceptable cost for real subprocess coverage.
- ⚪ **Nits:**
  - `ErrDiscoveryFailed`/`ErrCallFailed` are defined and classified but not directly hit by the current fake modes (crash/timeout/error map to process-exit/timeout/invalid-response); reserved for future server-internal errors.
  - Fake server uses `os.Stdout.Write` directly (test-only helper, no production logging concern).

## Follow-ups
- `/test cases FEAT-00003` for formal TEST-xxxxx cases (tester role).
- `/qc audit FEAT-00003` then `/qc sign-off FEAT-00003` (registry QA-00002 overlaps with this feature's integration tests).
- Register DEV column → ✅ in `docs/REGISTRY.md` after QC sign-off.
- Later features: result-size truncation, multi-server, HTTP/SSE (explicitly out of scope here).

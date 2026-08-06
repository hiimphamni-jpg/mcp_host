# Plan — FEAT-00007: Headless CLI prompt input and final-response output

- **Task ID:** FEAT-00007
- **Phase:** `/plan` (Planning) — Flow 1 "New Feature"
- **Date:** 2026-08-06
- **Status:** APPROVED (via `/build` invocation)

---

## Goal

Make `cmd/server` fully headless per TDD §9:
1. Accept a prompt via `--prompt`, or when omitted read one prompt from stdin **only when stdin is not a TTY** (piped/non-interactive).
2. Run the bounded agent loop and write only the final model text to stdout via `Fprintln` (newline appended).
3. Empty input (no `--prompt`, and empty/absent stdin) → print bootstrap status, exit 0 (existing contract preserved).
4. All failures → concise diagnostic to stderr, non-zero exit, no secret leaks.

## Assumptions (user-confirmed)

- **A1:** stdin is read **only when stdin is not a TTY** (piped/non-interactive), avoiding interactive hangs.
- **A2:** empty stdin + no `--prompt` → **bootstrap, exit 0** (existing `TestRun_WritesSafeBootstrapStatus` contract unchanged).
- **A3:** final response printed with **`Fprintln` (newline appended)** — exact raw match not required.

## Scope

- **Files touched:** `cmd/server/main.go` only (+ tests in `cmd/server/main_test.go`).
- **Out of scope:** agent loop, MCP client, mapper, policy, config, output channel flags (`--stdin`, `--out`), HTTP/SSE.

## Plan

### Step 1: Add stdin reading helper (Red — write failing tests first)
- **Files**: `cmd/server/main_test.go`, `cmd/server/main.go`
- **Agent**: /dev
- **Change**: Write tests that exercise a new `readPromptFromStdin(reader io.Reader) (string, error)` seam via an injected `bytes.Buffer`/`strings.Reader`: (a) non-empty stdin returns the trimmed prompt, (b) empty stdin returns `""`, (c) read error is surfaced. Tests reference a yet-unwritten helper → **fail** (Red).
- **Verify**: `go test ./cmd/server -run 'Stdin|Prompt'` → fails with undefined helper
- **Duration**: ~5 min

### Step 2: Implement stdin fallback in `run`
- **Files**: `cmd/server/main.go`
- **Agent**: /dev
- **Change**: Add `readPromptFromStdin(r io.Reader) (string, error)`. In `run`, change signature to accept an `io.Reader` stdin argument (`run(out io.Writer, stdin io.Reader, args ...string)`) so tests can inject input. When `*prompt == ""` **and stdin is not a TTY** (`os.Stdin.Stat()` mode check or a `*os.File` type assertion — keep the seam injectable), read one prompt from stdin; if non-empty, run the loop; else bootstrap.
- **Verify**: `go test ./cmd/server` → all tests pass, including new stdin tests (Green)
- **Duration**: ~8 min

### Step 3: Refactor + diagnostics/exit-code verification
- **Files**: `cmd/server/main.go`
- **Agent**: /dev
- **Change**: Ensure diagnostics stay on stderr (`fmt.Fprintln(os.Stderr, err)`) and `os.Exit(1)` is preserved in `main()`; confirm stdout receives only the final answer (`Fprintln(out, answer)`), no tool logs. Run `go vet` for hygiene.
- **Verify**: `go test ./cmd/server ./... && go vet ./cmd/server/...`
- **Duration**: ~5 min

### Step 4: Regression + contract check
- **Files**: `cmd/server/main_test.go`
- **Agent**: /test
- **Change**: Confirm existing bootstrap tests still pass unchanged (`TestRun_WritesSafeBootstrapStatus`, `TestRun_SmokeUnconfiguredInvocationEndsCleanly`); add test that piped-nonempty-stdin path triggers the loop path and that empty stdin still bootstraps. Update `TestRun_InvalidConfigDoesNotLeakSecret` if the stdin seam changes error text.
- **Verify**: `go test ./...`
- **Duration**: ~5 min

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Blocking/hanging on stdin in interactive runs | Only read stdin when not a TTY; A1 confirmed |
| Breaking existing bootstrap tests | A2 preserves contract; Step 4 re-runs the full suite |
| Secret/prompt leak to stderr on error | Existing leak test extended to cover stdin path |
| TTY detection is platform-specific on Windows | Keep the seam injectable; unit tests use injected reader, TTY branch is thin and verified by code review |

## Rollback Plan

Single-file change; revert `git checkout cmd/server/main.go cmd/server/main_test.go` (or revert the feature commit) restores bootstrap-only behavior. No migration, no data.

## Parallel Opportunities

Steps 1–3 are strictly sequential (test-then-implement-then-refactor, all in the same file). No parallel batches.

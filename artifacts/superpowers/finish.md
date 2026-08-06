# /finish Summary — FEAT-00007: Headless CLI prompt input and final-response output

## Status
- DEV: ✅ done (quality gates green)
- TEST: ✅ passed (TEST-00065..71, 7 scenarios)
- QC: ✅ approved
- Registry: `docs/REGISTRY.md` FEAT-00007 → **DONE**

## What shipped
- `cmd/server/main.go`:
  - `readPromptFromStdin(r io.Reader) (string, error)` — whole-stream read, whitespace-trimmed; empty stream → `""` (bootstrap path).
  - `isTerminal(r io.Reader)` — character-device check via `*os.File` type assertion; injected readers/pipes are never terminals, keeping the seam injectable (A1, Windows TTY risk mitigated).
  - `run` signature changed to `run(out io.Writer, stdin io.Reader, args ...string)`; `main()` passes `os.Stdin`. Empty `--prompt` + non-TTY stdin → read one prompt; non-empty → agent loop, empty → bootstrap (A1/A2).
  - Final answer printed via `Fprintln(out, answer)` (A3); diagnostics stay on stderr via `main()`'s `fmt.Fprintln(os.Stderr, err)` + `os.Exit(1)`; no tool logs on stdout.
- `cmd/server/main_test.go`:
  - New: `readPromptFromStdin` unit tests (trim, empty, read-error), `TestRun_EmptyStdin_Bootstraps`, `TestRun_NonEmptyStdin_TriggersLoop` (deterministic fail-fast MCP spawn, no real Gemini/MCP).
  - Existing bootstrap/secret/flag tests updated to the new stdin seam — contracts unchanged.

## Verification evidence
- `go test ./...` → PASS (all 7 packages)
- `go vet ./...` → PASS
- `gofmt -l .` → empty
- Manual E2E probe: empty piped stdin → `bootstrap complete`, EXIT=0; non-empty piped stdin + unresolvable MCP → EXIT=1, diagnostic on stderr only, empty stdout (A4: no secret/leak, correct exit code).

## Sign-off trail
- Plan: `artifacts/superpowers/plan.md` (user APPROVED)
- Execution: `artifacts/superpowers/execution.md`
- Review/QC: `artifacts/superpowers/review.md` (QC APPROVED, no blockers)
- Tests: `tests/TEST-00007.md` + `tests/REPORTS.md`

## Review pass (final)
- 🔴 Blockers: none.
- 🟠 Majors: none.
- 🟡 Minors: `isTerminal` treats a `Stat` error as "not a terminal" (reads stdin) — safe default, no hang.
- ⚪ Nits: git branch is `feature/FEAT-00003-stdio-mcp-client-lifecycle` — FEAT-00007 changes uncommitted in working tree.

## Follow-ups
- [SUGGEST] Create `feature/FEAT-00007-headless-cli-prompt-stdin` and commit the working-tree changes before merging (git-workflow §1).
- [Manual] Full end-to-end validation with a live Filesystem MCP server + real Gemini API key (covered separately by opt-in QA-00003 E2E).
- Task is **DONE** in Registry; further sprint items: QA-00001..03, DEVOPS-00001, DOC-00001, FEAT-00008/09 (out of scope).
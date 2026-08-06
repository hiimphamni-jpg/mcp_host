# Brainstorm — FEAT-00007: Headless CLI prompt input and final-response output

- **Task ID:** FEAT-00007
- **Phase:** `/think` (Clarify) — Flow 1 "New Feature"
- **Date:** 2026-08-06
- **Status:** Draft (open clarifying questions)

---

## Goal

Make the composition root (`cmd/server`) a fully headless CLI:

1. Accept a prompt either via `--prompt` or, when `--prompt` is omitted, read one prompt from **stdin** (per TDD §9).
2. Emit **only the final model text** to stdout (no tool logs, progress, or diagnostics on stdout).
3. Route all diagnostics to stderr and exit non-zero on any failure (invalid config, timeout, MCP failure, iteration limit).
4. Preserve the existing bootstrap status behavior for the case with **no prompt input at all**.

## Current state (from code)

- `cmd/server/main.go:run` already supports `--prompt` and prints the final answer to `out` (`main.go:122`).
- Missing: **stdin prompt input** when `--prompt` is omitted — currently it prints bootstrap status instead (`main.go:48-49`).
- Conflicting existing test: `TestRun_WritesSafeBootstrapStatus` and `TestRun_SmokeUnconfiguredInvocationEndsCleanly` expect bootstrap output when no `--prompt` is given.

## Constraints

- **No code until `/plan` is APPROVED** (rules/workflow.md §1).
- Respect `rules/api-convention.md`, `rules/code-style.md`, `rules/error-handling.md`, `rules/testing.md`, `rules/security.md`.
- MVP: single MCP server, Gemini only; HTTP/SSE is out of scope (deferred).
- Do not leak secrets (API keys, raw prompts) to stdout/stderr (business-logic §4).
- Preserve the existing bootstrap/test contract or update tests deliberately as part of the plan.
- Keep `cmd/server | internal/*` dependency direction; the agent loop itself is unchanged (FEAT-00006 done).

## Options

1. **Current only (`--prompt`)** — already implemented; does not meet TDD §9 stdin clause. → Reject.
2. **Flag + stdin fallback** — use `--prompt` if provided; otherwise read exactly one prompt from stdin; empty stdin still prints bootstrap. → Recommended.
3. **Stdin always reads** — changes bootstrap semantics, breaks existing tests. → Only if a new "reset/config" subcommand replaces bootstrap; larger scope. → Deferred.
4. **Add `--stdin`, `--out`, separate output channel flags** — over-engineered for headless MVP. → Reject for this task.

## Recommendation

**Option 2**: keep `--prompt`; add an `io.Reader` (stdin) that supplies the prompt when `--prompt` is empty. Bootstrap status remains the fallback when both are empty (preserves existing tests). Rationale: minimal, testable (inject `bytes.Buffer` as stdin), meets TDD §9, keeps diagnostics/exit-code contract, no new config or layers.

## Risks

- **Behavioral ambiguity** between "no prompt" (bootstrap) and "empty stdin" — must be defined in AC and tests.
- **Secret leak** if a raw prompt or answer is echoed to stderr on error — guard in tests.
- **Blocking stdin** — if stdin is read unconditionally it could hang in interactive runs; mitigate by only reading stdin when `--prompt == ""` and documenting headless usage.
- **Trailing newline/whitespace** in final answer affects exact stdout assertions — tests must tolerate formatting.
- Existing bootstrap tests may need to remain valid; scope of change to `main.go` only.

## Open clarifying questions (essential only)

1. When `--prompt` is omitted, should stdin be read **always**, or **only when stdin is not a TTY** (piped) to avoid interactive hangs?
2. Should an **empty stdin** (EOF with no bytes) return the bootstrap status, or be treated as an **error** (non-zero exit)?
3. Is exact stdout formatting (e.g., strip trailing newline) required, or is `Fprintln(answer)` acceptable?

## Acceptance Criteria (draft)

- **AC1:** `mcp-host --prompt "X"` runs the agent loop and writes only the final text to stdout.
- **AC2:** `mcp-host` (no `--prompt`) reads one prompt from stdin; when non-empty it runs the loop and writes only the final text to stdout.
- **AC3:** when neither `--prompt` nor non-empty stdin is present, the CLI prints the bootstrap status and exits 0 (existing behavior preserved).
- **AC4:** on any failure (invalid config, MCP/LLM timeout, iteration limit) the CLI writes a concise diagnostic to stderr and exits non-zero, never leaking secrets.
- **AC5:** tests in `cmd/server` cover stdin-prompt, `--prompt`, empty-input bootstrap, and error-to-stderr paths.

---

## Next gate

> **Architecture design required?** — **No full `/architect` TDD needed** (not ≥3 tables / ≥2 layers; single-file CLI change). A short `/architect review` may optionally confirm the stdin reading seam. The immediate next step is `/pm registry` (confirm 5-digit ID + DoR) then `/plan FEAT-00007`.
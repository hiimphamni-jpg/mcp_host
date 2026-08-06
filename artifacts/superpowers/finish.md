# /build Finish Summary — FEAT-00005: Gemini provider adapter and conversation mapping

## Status: IMPLEMENTED (ready for `/finish` QC sign-off)

## What was built
New package `internal/llm` (only; no edits to config/policy/mcpclient/mapping/cmd/server):

| File | Purpose |
|------|---------|
| `message.go` | SDK-free neutral types: `Role` (`RoleUser`/`RoleModel`), `Message{Role, Text, ToolCalls, ToolResults}`, `ToolCall{Name, Arguments}`, `ToolResult{Name, Response}`, `Response{Text, ToolCalls}`. No `genai` import. |
| `interface.go` | `LLM` interface with `Generate(ctx, *Request) (*Response, error)`; `Request{Contents, ToolNames}`. Agent (FEAT-00006) depends only here (AC5). |
| `errors.go` | Sentinels `ErrTimeout`, `ErrProvider`; typed user-safe `*Error{Cause, Operation}` with `Unwrap`. Carries only a non-sensitive operation label — never the API key or prompt text. |
| `convert.go` | Pure helpers: `toGemContent`, `toolCallToPart`, `toolResultToPart`, `responseToResponse`. No I/O, credentials, or side effects. |
| `gemini.go` | `modelsClient` seam + `Gemini` adapter. `New(client, model, timeout, decls)` sets `generator: client.Models`. `Generate` applies `LLM_TIMEOUT` via `context.WithTimeout`; maps `DeadlineExceeded`→`ErrTimeout`, passes `Canceled` through, others→`ErrProvider`. Builds `Tools` config for named tools only. |
| `convert_test.go` | 12 conversion tests (text, function-call, tool-result, mixed, empty, nil, text+call). |
| `gemini_test.go` | 8 fake-`modelsClient` tests (text, tool-call, timeout, cancel, provider error, tool config, no-tools, contents). No network or `GEMINI_API_KEY`. |

## Verification evidence
- TDD RED confirmed for Steps 4 and 5 (build failed before implementation), GREEN after.
- `gofmt -l internal/llm` → clean (empty; `gofmt -w` applied)
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./internal/llm/...` → PASS
- `git status` → only untracked `internal/llm/`; plan.md pre-existing; no diff to `config/policy/mcpclient/mapping/cmd/server`

## Review pass
- 🔴 **Blockers:** none.
- 🟠 **Majors:** none.
- 🟡 **Minors:**
  - Plan's Step 4 verify was `... -run Convert`, but test functions don't contain the substring `Convert`; the full package run is used instead. Cosmetic divergence from the approved plan test-selector, no behavior impact.
- ⚪ **Nits:**
  - `providerError` discards the underlying provider detail to keep messages user-safe (intentional per business-logic §5); if agent needs to distinguish provider causes, a future `Cause` expansion would be needed.

## Open decisions (flagged to `/architect` before FEAT-00006)
1. Neutral `Request` carries `ToolNames []string` vs full neutral tool declarations. Implemented: names + adapter-owned `declByTool map[string]*genai.FunctionDeclaration` (single mapping via `internal/mapping`).
2. `ctx` cancellation vs `LLM_TIMEOUT` split — implemented: adapter applies `context.WithTimeout(ctx, timeout)`; caller cancellation passes through as `context.Canceled`. Confirm `agent` owns the per-prompt parent context (business-logic §3).

## Follow-ups
- `/architect review` the two open decisions above before FEAT-0006 starts (tool-declaration hand-off).
- FEAT-0006 wires `internal/agent` to the `LLM` interface; adapter is currently unused (compiles standalone, exercised only by its own tests).
- `/test cases FEAT-00005` for formal TEST-xxxxx cases, then `/qc audit` + `/qc sign-off FEAT-00005`.
- Register DEV column → ✅ in `docs/REGISTRY.md` after sign-off.
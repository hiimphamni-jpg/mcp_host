# Plan — FEAT-00005: Gemini provider adapter and conversation request/response mapping

## Goal

Create the `internal/llm` package containing (1) a provider-neutral `LLM` interface plus neutral conversation types, and (2) a Gemini adapter that wires declared tools and conversation history into `genai.GenerateContent` calls and maps the provider response (final text, or requested function calls) back into neutral types. `internal/agent` (FEAT-00006) will depend only on the neutral `LLM` interface, not on `mcp-go` or the Gemini SDK (TDD §2.1 / business-logic AC5).

## Assumptions

- `internal/mapping` already produces `genai.FunctionDeclaration` (FEAT-00004 Done) and is the fixture that converts MCP `tools/list` output into declarations.
- The Gemini adapter wraps the real GenAI SDK (`google.golang.org/genai v1.66.0`, already a direct dependency in `go.mod`). The `LLM` interface + neutral `Request`/`Response`/`Message` types stay SDK-free.
- `agent` owns tool-declaration preparation in FEAT-00006; the neutral `Request` carries only tool *names* as an indirection so the adapter never re-runs JSON-Schema mapping. This is recorded as an Open Decision below so `/architect` can confirm before FEAT-00006 starts.
- Per-prompt timeout comes from `LLM_TIMEOUT` applied via `context.WithTimeout`; a timeout must surface a typed, user-safe error and must not log the API key (rules/security.md, rules/error-handling.md).
- All columns for FEAT-00005 are empty (SPRINT BACKLOG); DEV ⬜. This plan delivers DEV output only.

## Plan

### Step 1: `internal/llm` — neutral conversation types
- **Files**: `internal/llm/message.go`
- **Agent**: /dev
- **Change**: Define SDK-free `Role` (`RoleUser`, `RoleModel`), `Message{Role, Text, ToolCalls, ToolResults}`, `ToolCall{Name, Arguments}`, `ToolResult{Name, Response}`, and `Response{Text, ToolCalls}`. No `genai` import.
- **Verify**: `go build ./internal/llm/...`
- **Duration**: ~5 min

### Step 2: `internal/llm` — provider-neutral interface
- **Files**: `internal/llm/interface.go`
- **Agent**: /dev
- **Change**: Define `type LLM interface { Generate(ctx context.Context, req *Request) (*Response, error) }` and `Request{Contents []*Message, ToolNames []string}`. Document that agent depends only here (AC5).
- **Verify**: `go vet ./internal/llm/...`
- **Duration**: ~3 min

### Step 3: `internal/llm` — typed errors
- **Files**: `internal/llm/errors.go`
- **Agent**: /dev
- **Change**: Add sentinels `ErrTimeout`, `ErrProvider` (wrapping `errors.Is`) and a typed user-safe error with an `Operation` label; must not carry the API key or raw request text (business-logic §5). Mirror `internal/mcpclient/errors.go` style.
- **Verify**: `go test ./internal/llm/...` (with Step 4 tests)
- **Duration**: ~3 min

### Step 4: `internal/llm` — pure conversion helpers + tests
- **Files**: `internal/llm/convert.go`, `internal/llm/convert_test.go`
- **Agent**: /dev (TDD: Red → Green)
- **Change**: Pure helpers: `toGeminiContent(msg *Message) *genai.Content` (maps text parts and FunctionResponse/ToolResults per role); `toolCallToPart`, `toolResultToPart`; `responseToResponse(r *genai.GenerateContentResponse) *Response` (final `Text()` when no calls, else `FunctionCalls()` → `[]*ToolCall`). Tests cover: user text, model function call → part, tool result → user function part, empty candidates, mixed parts.
- **Verify**: `go test ./internal/llm/... -run Convert`
- **Duration**: ~7 min

### Step 5: `internal/llm` — Gemini adapter + tests
- **Files**: `internal/llm/gemini.go`, `internal/llm/gemini_test.go`
- **Agent**: /dev (TDD)
- **Change**: `type modelsClient interface { GenerateContent(ctx, model string, contents []*Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) }`. Implement `Gemini` holding `model` name, `timeout`, `generator modelsClient`, prepared `declByTool map[string]*genai.FunctionDeclaration`. Constructor `New(client *genai.Client, cfg map, timeout)` sets `generator: client.Models`. `Generate`: build `[]*genai.Content`, build `&genai.GenerateContentConfig{Tools: []*genai.Tool{{FunctionDeclarations: decls}}}` for named tools, call under `context.WithTimeout(LM_TIMEOUT)`, wrap errors (`ErrTimeout` on `errors.Is(ctx, DeadlineExceeded)`), return `Response`. Tests inject a fake `modelsClient` (no network/credentials) covering text-only, tool-call, and timeout paths.
- **Verify**: `go test ./internal/llm/...`
- **Duration**: ~10 min

### Step 6: Wiring & repo-wide verification
- **Files**: none new (unused-adapter check only; FEAT-00006 wires `internal/agent`).
- **Agent**: /dev
- **Change**: No production call site yet (agent is FEAT-00006). Ensure package compiles standalone and is exercised only by its own tests.
- **Verify**: `go build ./...; go vet ./...; go test ./internal/llm/...`
- **Duration**: ~4 min

## Risks & Mitigations

- **genai `Models.GenerateContent` value-receiver signature vs fake**: Resolves because `client.Models` is `*Models` and implements the interface; tests use a fake `generator` — no network needed. Verify at Step 5.
- **Tool-declaration hand-off unresolved**: Mitigate by keeping `Request.ToolNames` (names) + adapter-owned `map[name]*Declaration` prepared by agent; convert only once. Confirm with `/architect` before FEAT-00006.
- **Timeout vs cancellation ambiguity**: use `context.WithTimeout` and map `context.DeadlineExceeded` to `ErrTimeout`; distinguish caller cancellation (context.Canceled) so agent (FEAT-00006) can cancel correctly.
- **No credentials in tests**: unit tests rely purely on fake `generator`; no live API call, no `GEMINI_API_KEY` needed, cannot be a flaky/e2e.

## Rollback Plan

- All changes are additive to the new `internal/llm` package; no existing package is edited.
- Revert by deleting `internal/llm/` (new files only) → `go build ./...` returns to prior green state (go.mod already had `genai`, no dependency change).
- `git checkout` the single plan artifact if needed; no cross-module impact.

## Parallel Opportunities

- ⚡ Steps 1, 2, 3 can run in parallel (distinct files `message.go`, `interface.go`, `errors.go`, same package, no file-level conflict; symbols resolve at final compile).
- Steps 4 → 5 are sequential (Step 5 depends on convert.go helpers + types/errors).
- Step 6 is a verification pass gated on Steps 4–5.

## Open Decisions (flag to /architect before FEAT-00006)

1. Neutral `Request` carries `ToolNames []string` vs full neutral tool declarations. Recommendation: names + adapter-owned declaration map (single mapping via `internal/mapping`).
2. Where `ctx` cancellation vs `LLM_TIMEOUT` should be split; confirm `agent` owns the per-prompt parent context (`business-logic §3`).
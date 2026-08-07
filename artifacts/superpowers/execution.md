# /build Execution Log — FEAT-00003: Stdio MCP client lifecycle

Branch: `feature/FEAT-00003-stdio-mcp-client-lifecycle`

## Step 1: Define the MCP client interface and typed errors
- Files: `internal/mcpclient/interface.go`, `internal/mcpclient/errors.go`, `internal/mcpclient/errors_test.go`
- Agent: /dev (TDD red → green)
- Changes:
  - Defined `Client` interface (`Initialize`, `ListTools`, `CallTool`, `Close`) with `mcp-go` result types (AC5 interface for future `internal/agent`).
  - Defined sentinel typed errors (`ErrProcessExit`, `ErrInitFailed`, `ErrDiscoveryFailed`, `ErrCallFailed`, `ErrInvalidResponse`, `ErrTimeout`) with `Error`/`TimeoutError` types carrying user-safe messages (AC4).
- TDD: RED — `errors_test.go` written first failed (package undefined); GREEN after `errors.go`/`interface.go`.
- Verify: `go build ./... && go vet ./... && go test ./internal/mcpclient/...` → PASS

## Step 2: Spawn + initialize lifecycle wrapper
- Files: `internal/mcpclient/client.go`
- Agent: /dev
- Changes:
  - `NewStdio` builds argv as `append(policy.Args(), policy.Roots()...)` and spawns via `client.NewStdioMCPClientWithOptions` (TDD §5/§8: args are policy-supplied, never LLM-supplied).
  - `Initialize` runs the handshake with `mcp.LATEST_PROTOCOL_VERSION` under a `MCP_TIMEOUT`-bounded context.
  - `classify` maps mcp-go errors: deadline→`TimeoutError`, `transport.ErrTransportClosed`→`ErrProcessExit`, server JSON-RPC error payloads→`ErrInvalidResponse`, else the operation's typed kind.
- Verify: `go build ./... && go vet ./...` → PASS

## Step 3: Discovery — tools/list
- Files: `internal/mcpclient/client.go`
- Agent: /dev
- Changes:
  - `ListTools` calls `raw.ListTools` with `mcp.ListToolsRequest{}` under a `MCP_TIMEOUT`-bounded context; classifies process exit / invalid response / timeout via `classify`.
- Verify: `go build ./... && go vet ./...` → PASS

## Step 4: Call — tools/call with timeout
- Files: `internal/mcpclient/client.go`
- Agent: /dev
- Changes:
  - `CallTool` builds `mcp.CallToolRequest{Params:{Name, Arguments}}` and invokes it under a `MCP_TIMEOUT`-bounded context; deadline/child-exit/invalid response map to typed errors (AC4). Caller context is honoured.
- Verify: `go build ./... && go vet ./...` → PASS

## Step 5: Cleanup / Close (idempotent, no orphan)
- Files: `internal/mcpclient/client.go`
- Agent: /dev
- Changes:
  - `Close` closes the underlying transport (mcp-go already SIGTERM→kill-and-wait), is idempotent via a `closed` flag, and is safe on partially-initialized clients and after a crash.
  - `checkClosed` surfaces a typed `ErrProcessExit` before operations after Close.
- Verify: `go build ./... && go vet ./... && go test ./internal/mcpclient/...` → PASS

## Step 6: Integration test — fake stdio MCP server
- Files: `internal/mcpclient/client_integration_test.go`, `internal/mcpclient/testdata/fakeserver/main.go`
- Agent: /test (TDD red → green)
- Changes:
  - Built a tiny stdio JSON-RPC MCP server (`testdata/fakeserver`) driven by modes: `normal`, `crash` (exit 1), `timeout` (never responds, EOF-aware), `error` (JSON-RPC error for tools/call). It writes an exit-marker in the root dir on clean EOF so tests prove termination.
  - Integration tests (all via real subprocess, no live Filesystem server): happy-path initialize+list+call; crash→`ErrProcessExit` + idempotent Close; timeout→`ErrTimeout`; server protocol error→`ErrInvalidResponse`; Close leaves no orphan (exit-marker within deadline).
  - Policy argv constraint honored: bare fake-server command resolved via PATH prepend; argv = `Args()+Roots()` (TDD §5/§8).
- Verify: `go test ./internal/mcpclient/... -count=1 -v` → 9 tests PASS; coverage 86.4% (target ≥80%)
- Notes: `gofmt -w` applied; no orphan `mcp-fake-server` processes confirmed via `Get-Process`.

## Step 7: Full verification + review pass
- Files: none new
- Agent: /dev
- Change: full quality gates + review findings below.
- Verify: `go build ./... && go vet ./... && go test ./... -count=1` → PASS (cmd/server, config, policy, mcpclient all ok)

---

# /build Execution Log — FEAT-00004: MCP JSON Schema → Gemini function-declaration mapper

Branch: `feature/FEAT-00004-json-schema-to-genai-mapper`

Approach: TDD red → green. Tests written first (RED: package undefined), then implementation (GREEN).

## Step 1 (TDD RED): AC2 table + rejection/duplicate/recursion tests
- Files: `internal/mapping/mapping_test.go`
- Agent: /test (TDD red)
- Changes:
  - `TestMapTools_FullSupportedSubsetLossless` mirrors AC2: string/number/integer/boolean/array, nested object, `enum` (both `[]any` and `[]string` shapes), top-level and nested `required` (nested as JSON-decoded `[]any`), array items. Asserts lossless via `reflect.DeepEqual`.
  - Rejection table: unsupported types (`null`, `any`, empty, unknown → `ErrUnsupportedType`; missing type → `ErrMalformed`), nested unsupported with full `config.inner.x` path + tool name, malformed property/properties/required/array-items, non-string enum member.
  - Duplicate-name rejection, cyclic-schema recursion guard, and 300-level deep-nesting recursion guard.
- TDD: RED — `go test ./internal/mapping/...` → FAIL (package has no non-test files); GREEN after implementation.
- Verify: `go test ./internal/mapping/... -count=1` → PASS (all subtests)

## Step 2: Typed errors + package surface
- Files: `internal/mapping/errors.go`
- Agent: /dev
- Changes: sentinels `ErrUnsupportedType`, `ErrDuplicateName`, `ErrMalformed`; typed `*Error{Cause,Tool,Path,Description}` implementing `error` + `Unwrap` (errors.Is), formatted as `tool "X" field "p": reason` (TDD §6 tool-name + field-path requirement).
- Verify: `go build ./... && go vet ./...` → PASS

## Step 3: Primitive/scalar type mapping
- Files: `internal/mapping/schema.go`
- Agent: /dev
- Changes: `mapType` maps `object/string/number/integer/boolean/array` → genai Type enums, path-bearing rejection of any other string (incl. `null`/`any`/empty). `mapSchemaValue` builds scalar schemas with description, enum, and numeric `minimum`/`maximum` bounds (mcp-go Min/Max helpers).
- Verify: `go build ./... && go vet ./...` → PASS

## Step 4: Nested objects, arrays/items, required — recursive builder + depth guard
- Files: `internal/mapping/schema.go`
- Agent: /dev
- Changes: `mapSchemaValue`/`mapObject`/`mapArray` recurse into object `properties` and array `items`, preserving nested `required`/enum. `maxRecursionDepth = 64` guards cyclic/oversized schemas (rejects instead of stack overflow, TDD §6). During review-tighten: `required`/`properties` wrong types now rejected as malformed (no silent failure); nested `required` as JSON-decoded `[]any` handled losslessly.
- Verify: `go build ./... && go vet ./... && go test ./internal/mapping/... -count=1` → PASS

## Step 5: MapTools entrypoint + duplicate rejection
- Files: `internal/mapping/mapping.go`
- Agent: /dev
- Changes: `MapTools([]mcp.Tool)` iterates tools, tracks seen names, returns `ErrDuplicateName` on repeat, builds root object `Parameters` from `InputSchema.Properties`/`Required` (lossless). Empty-name tool rejected as malformed. Depends only on `mcp-go` + `genai` (AC5).
- Verify: `go build ./... && go vet ./... && go test ./internal/mapping/... -count=1` → PASS

## Step 6: Full verification + gofmt + review pass
- Files: none new (gofmt -w on the 4 new files)
- Agent: /dev
- Changes: applied `gofmt -w` to remove formatting diffs; confirmed only `internal/mapping/*` added (untracked), no edits to `config`/`policy`/`mcpclient`/`cmd/server`.
- Verify: `gofmt -l .` → empty; `go build ./...` → PASS; `go vet ./...` → PASS; `go test ./... -count=1` → PASS (28 subtests green); mapping coverage 83.1% (target ≥80%; QA-00001).

---

# /build Execution Log — FEAT-00005: Gemini provider adapter and conversation mapping

Plan: `artifacts/superpowers/plan.md` (user replied APPROVED).

Approach: TDD red → green for Steps 4 and 5. Steps 1–3 are tiny separate files (message.go/interface.go/errors.go) marked ⚡ parallel in the plan; executed sequentially (default), verified together.

## Step 1: Neutral conversation types
- Files: `internal/llm/message.go`
- Agent: /dev
- Changes:
  - SDK-free `Role` (`RoleUser`/`RoleModel`), `Message{Role, Text, ToolCalls, ToolResults}`, `ToolCall{Name, Arguments}`, `ToolResult{Name, Response}`, `Response{Text, ToolCalls}`. No `genai` import.
- Verify: `go build ./internal/llm/...` → PASS

## Step 2: Provider-neutral interface
- Files: `internal/llm/interface.go`
- Agent: /dev
- Changes:
  - `type LLM interface { Generate(ctx context.Context, req *Request) (*Response, error) }`; `Request{Contents []*Message, ToolNames []string}`. Documented agent (FEAT-00006) depends only here (AC5).
- Verify: `go vet ./internal/llm/...` → PASS (with Step 1 file)

## Step 3: Typed errors
- Files: `internal/llm/errors.go`
- Agent: /dev
- Changes:
  - Sentinels `ErrTimeout`, `ErrProvider` (errors.Is via `Unwrap`); typed user-safe `*Error{Cause, Operation}` carrying only a non-sensitive operation label — never the API key or prompt text (business-logic §5, rules/security.md). Mirrors `internal/mcpclient/errors.go` style.
- Verify: `go build ./internal/llm/...` + `go vet ./internal/llm/...` → PASS

## Step 4: Pure conversion helpers + tests
- Files: `internal/llm/convert.go`, `internal/llm/convert_test.go`
- Agent: /dev (TDD: Red → Green)
- Changes:
  - `toGemContent` (text + FunctionResponse per role), `toolCallToPart`, `toolResultToPart`, `responseToResponse` (final text + `FunctionCalls()` → `[]*ToolCall`).
  - Tests cover: user text, model function call → part, tool result → user function part, mixed parts, text-only response, function-call response, empty candidates, nil response, mixed text+call.
- TDD: RED — build failed (helpers undefined); GREEN after `convert.go`.
- Verify: `go test ./internal/llm/...` → PASS (note: test names don't match the plan's `-run Convert` prefix, so the full package run was used)
- Notes: `gofmt -w` applied at Step 6.

## Step 5: Gemini adapter + tests
- Files: `internal/llm/gemini.go`, `internal/llm/gemini_test.go`
- Agent: /dev (TDD)
- Changes:
  - `modelsClient` seam matching `*genai.Models.GenerateContent` signature; `Gemini{model, timeout, generator, declByTool}`; `New(client *genai.Client, model, timeout, decls)` sets `generator: client.Models`.
  - `Generate`: builds `[]*genai.Content`, `&genai.GenerateContentConfig{Tools: [...]}` for named tools only, runs under `context.WithTimeout(ctx, timeout)`; maps `DeadlineExceeded` → `ErrTimeout`, passes `Canceled` through, wraps other provider failures → `ErrProvider`.
  - Tests inject a fake `modelsClient` (no network/credentials): text-only, tool-call, timeout→ErrTimeout, cancel pass-through, provider error, tool-config assembly for named tools, no-tools when none named, contents built.
- TDD: RED — build failed (`Gemini` undefined); GREEN after `gemini.go`.
- Verify: `go test ./internal/llm/...` → PASS
- Notes: initial impl returned all declarations when `ToolNames` was empty; test exposed it, fixed to "tools for named tools" (plan Step 5) → GREEN.

## Step 6: Wiring & repo-wide verification
- Files: none new (unused-adapter check only; FEAT-00006 wires internal/agent)
- Agent: /dev
- Changes:
  - `gofmt -w internal/llm` → `gofmt -l` clean; only `internal/llm/*` added, no edits to config/policy/mcpclient/cmd/server.
- Verify:
  - `gofmt -l internal/llm` → PASS (empty)
  - `go build ./...` → PASS
  - `go vet ./...` → PASS
  - `go test ./internal/llm/...` → PASS
---

# /build Execution Log ? FEAT-00006: Bounded agentic loop, tool-result context, cancellation, and error handling

Plan: artifacts/superpowers/plan.md (user replied APPROVED for the full flow).
Design authority: docs/adr/ADR-0001-internal-agent-architecture.md (D1-D4). ADR is authoritative; plan pseudo-steps reconciled to the ADR's ToolCaller seam (D2).

Plan summary: Step 1 skeleton+errors; Step 2 neutral schema validator; Step 3 result bounding + safe error mapping; Step 4 Run loop; Step 5 fake-driven tests; Step 6 cmd/server wiring; Step 7 quality gates + REGISTRY DEV.

## Step 1: Agent package skeleton + types + sentinel errors
- Files: internal/agent/errors.go, internal/agent/agent.go, internal/agent/agent_test.go (construction tests)
- Agent: /dev (TDD red -> green)
- Changes:
  - Neutral Options{LLM llm.LLM; Tools ToolCaller; Schemas map[string]any; MaxIterations; MaxResultBytes}; Agent{ToolCaller seam per ADR-0001 D2} - agent defines its own SDK-free ToolCaller, no mcp-go/genai import.
  - Sentinel ErrIterationLimit + typed agent.Error{Cause, Operation} mirroring internal/llm/errors.go style (user-safe message).
  - New() rejects nil LLM, nil Tools, non-positive limits; Run() placeholder returns ErrIterationLimit (loop body in Step 4).
- Verify: go build ./... -> PASS; go test ./internal/agent/... -> 6 tests PASS (incl. placeholder iteration-limit).

## Step 2: Neutral schema-argument validator
- Files: internal/agent/schema.go, internal/agent/schema_test.go
- Agent: /dev (TDD red -> green)
- Changes:
  - Stdlib-only validateArgs(schema map[string]any, args map[string]any) error for the MVP subset: object props, required, string/number/integer/boolean/array+items, enum; mirrors internal/mapping subset without importing it (no genai). Unknown keys lenient; empty/untyped schema unconstrained.
  - Secret-safe messages (field names + Go type names only, never embedded values); numeric looseEqual so integer enum members match float64 args.
- Verify: go test ./internal/agent/... -> 14 tests PASS (one red test fixed: nil value for object prop correctly rejected).

## Step 3: Tool-result bounding + safe error mapping
- Files: internal/agent/results.go, internal/agent/results_test.go
- Agent: /dev (TDD red -> green)
- Changes:
  - BoundedResult(tr *llm.ToolResult, maxBytes) serializes Response to JSON, truncates to MaxResultBytes with explicit '...[truncated]' marker (works on the already-neutral llm.ToolResult from the ToolCaller seam, per ADR-0001 D2).
  - MessageResult / ErrorResult map failures to bounded, non-secret context; ErrorResult never embeds err.Error() (no secret/path leak); deadline/cancel labels only.
- Verify: go test ./internal/agent/... -run Result -> 7 PASS (truncation marker, no-secret, deadline/cancel mapping, nil).

## Step 4: Agent loop (Run) - main logic
- Files: internal/agent/agent.go (extend)
- Agent: /dev
- Changes:
  - Full bounded loop per plan pseudo-code + ADR-0001 D2/D3/D4: history starts with the user prompt; each iteration calls LLM.Generate with the neutral Request (Contents + sorted ToolNames); final text with no tool calls returns directly; tool-call turns append a model Message with the calls, execute each call sequentially through the ToolCaller seam, and append a user Message with the bounded ToolResults.
  - Terminal errors: caller cancellation (context.Canceled) propagates immediately (checked before each tool call and after a failed CallTool — D3 precedence); LLM timeout/provider failures wrap into typed agent.Error.
  - Non-terminal: unknown tool name -> "action unavailable", schema-invalid args -> "arguments failed validation", MCP call failure -> ErrorResult (bounded, non-secret), success -> BoundedResult capped to MaxResultBytes.
  - Loop exits with ErrIterationLimit after MaxIterations iterations without a final answer.
- Verify: go test ./internal/agent/... -run TestRun -> PASS (all 12 Run subtests, incl. truncation + no-secret).

## Step 5: Agent unit tests with fakes
- Files: internal/agent/agent_test.go (extend), internal/agent/errors_test.go (extend), internal/agent/schema_test.go (extend)
- Agent: /dev (TDD)
- Changes:
  - stubLLM scripts a response sequence and records Requests; stubMCP returns fixed results/errors by tool name, records the contexts, and can fire a cancel() on first call to simulate in-flight cancellation. simpleSchemas declares one tool with a required string property.
  - Tests: final text direct; single tool call -> final text (result fed back); multiple tool calls in one turn run sequentially in request order; iteration limit -> ErrIterationLimit; unknown tool -> safe bounded result (caller never invoked); invalid args -> safe bounded result; MCP failure -> bounded generic result (no raw error/stacktrace leak); pre-cancelled ctx -> context.Canceled with zero tool invocations; mid-loop cancel -> Canceled and no second tool call, CallTool received the cancelled context; cancel overrides a bounded tool error; result truncated to max bytes with marker; secret content not in final output.
  - New() validation tests (nil LLM/nil Tools/non-positive limits rejected); classifyGenerateError passthrough + sanitization; schema-helper edge tests ([]any required, []string enum, int-family numeric kinds) added to lift coverage past the 80% target.
- Verify: go test ./internal/agent/... -count=1 -> PASS; coverage 91.5% (target >= 80%).

## Step 6: Wire the pipeline in cmd/server
- Files: cmd/server/main.go (rewrite run), cmd/server/main_test.go (extend)
- Agent: /dev
- Changes:
  - run(out, args...) parses a --prompt flag (FlagSet, ContinueOnError). Without --prompt it keeps the bootstrap status contract; with --prompt it runs the full composition-root pipeline: policy.New -> mcpclient.NewStdio -> Initialize -> ListTools -> mapping.MapTools -> declsByName -> llm.New -> agent.New with neutral Schemas (built per tool via JSON round-trip of mcp.Tool.InputSchema) + limits -> agent.Run -> print final text.
  - mcpToolCaller adapts mcpclient.Client to the agent's SDK-free ToolCaller seam (ADR-0001 D2): converts mcp.TextContent / mcp.EmbeddedResource(TextResourceContents) into a neutral llm.ToolResult with content text and isError flag; forwards the caller context verbatim so cancellation reaches the in-flight MCP call.
  - defer client.Close() after NewStdio on every path (AC4: no orphan). Full stdout/exit-code contract remains FEAT-00007.
  - Smoke test: unconfigured invocation (no --prompt) ends cleanly with the bootstrap message, touching no real Gemini/MCP in CI. Adapter unit tests cover text/embedded conversion, isError flag, error+context propagation, and neutralSchema round-trip.
- Verify: go build ./... -> PASS; go vet ./... -> PASS; go test ./cmd/server/... -count=1 -v -> 10 tests PASS.

## Step 7: Cross-package quality gates
- Files: none new (docs/REGISTRY.md updated)
- Agent: /dev
- Changes:
  - gofmt -l . -> empty; go build ./... -> PASS; go vet ./... -> PASS; go test ./... -count=1 -> all packages PASS (cmd/server, agent, config, llm, mapping, mcpclient, policy).
  - internal/agent statement coverage 91.5% (target >= 80%); grep for genai/mcp-go in internal/agent matches comments only — no import crosses into the package.
  - docs/REGISTRY.md: FEAT-00006 DEV column -> done, Status -> IN PROGRESS (TEST/QC pending).
- Verify: go test ./... -count=1 -> PASS; go vet ./... -> PASS; gofmt -l . -> empty.

## Review pass (before finish)
- 🔴 Blockers: none.
- 🟠 Majors: none.
- 🟡 Minors: `defer client.Close()` ignores a close error in runAgentLoop — intentional (primary operation error is returned; Close failure after a completed agent loop is not actionable). No orphan risk: Close is still always called.
- ⚪ Nits: none.

---

# /build Execution Log — FEAT-00007: Headless CLI prompt input and final-response output

Plan: `artifacts/superpowers/plan.md` (approved via `/build` invocation). Steps 1–3 strictly sequential (test-then-implement-then-refactor in the same file); no parallel batches per plan.

## Step 1 (TDD RED): stdin reading helper tests
- Files: `cmd/server/main_test.go`
- Agent: /test (TDD red)
- Changes:
  - `TestReadPromptFromStdin_NonEmpty_ReturnsTrimmedPrompt`, `TestReadPromptFromStdin_Empty_ReturnsEmptyString`, `TestReadPromptFromStdin_ReadError_IsSurfaced` exercise a yet-unwritten `readPromptFromStdin(io.Reader) (string, error)` seam with `strings.Reader` / a failing `errReader`.
  - `TestRun_EmptyStdin_Bootstraps` locks the A2 contract: piped empty stdin + no `--prompt` → bootstrap, exit 0.
- TDD: RED — `go test ./cmd/server -run 'Stdin|Prompt'` → build failed: `undefined: readPromptFromStdin` (expected).
- Verify: `go test ./cmd/server -run 'Stdin|Prompt'` → RED (undefined helper) ✅

## Step 2 (GREEN): stdin fallback in `run`
- Files: `cmd/server/main.go`, `cmd/server/main_test.go`
- Agent: /dev
- Changes:
  - Added `readPromptFromStdin(r io.Reader)` (whole-stream read, trimmed) and `isTerminal(r io.Reader)` (character-device check via `*os.File` type assertion; injected readers/pipes are never terminals so the seam stays testable — Windows TTY risk from plan mitigated).
  - `run` signature changed to `run(out io.Writer, stdin io.Reader, args ...string)`; `main()` passes `os.Stdin`. When `--prompt` is empty **and** stdin is not a TTY, one prompt is read from stdin; non-empty → agent loop, empty → bootstrap (A1, A2).
  - Existing bootstrap/error tests updated to the new signature; `TestRun_NonEmptyStdin_TriggersLoop` proves the piped path spawns the MCP loop (deterministic fail-fast via unresolvable command, no real Gemini/MCP touched).
- Verify: `go test ./cmd/server -count=1` → PASS (11 tests) ✅

## Step 3: Refactor + diagnostics/exit-code contract
- Files: `cmd/server/main.go`
- Agent: /dev
- Changes:
  - Confirmed `main()` keeps `fmt.Fprintln(os.Stderr, err)` + `os.Exit(1)`; `runAgentLoop` writes only the final answer via `Fprintln(out, answer)` (A3); tool logs never reach stdout.
  - `gofmt -l cmd/server/` → empty; `go vet ./cmd/server/...` → PASS.
  - Manual E2E probe (built binary): empty piped stdin → `bootstrap complete`, EXIT=0; non-empty piped stdin with unresolvable MCP → EXIT=1, diagnostic on stderr only, empty stdout.
- Verify: `go test ./cmd/server ./... -count=1` → PASS (all 7 packages); `go vet ./cmd/server/...` → PASS; `gofmt -l cmd/server/` → empty ✅

## Step 4: Regression + contract check
- Files: `cmd/server/main_test.go`
- Agent: /test
- Changes:
  - Existing contracts unchanged: `TestRun_WritesSafeBootstrapStatus`, `TestRun_SmokeUnconfiguredInvocationEndsCleanly`, `TestRun_InvalidConfigDoesNotLeakSecret`, `TestRun_InvalidConfigReturnsError`, `TestRun_FlagParseErrorReturnsCleanError` all pass with the new stdin seam.
  - `TestRun_NonEmptyStdin_TriggersLoop` + `TestRun_EmptyStdin_Bootstraps` cover the A1/A2 stdin paths; leak test still guards secrets.
- Verify: `go test ./... -count=1` → PASS (cmd/server 13 tests, all packages green) ✅

## Review pass (before finish)
- 🔴 Blockers: none.
- 🟠 Majors: none.
- 🟡 Minors: `isTerminal` returns false on a `Stat` error (treats an unreadable stdin as piped, reads it) — safe default: worst case bootstrap handles a read failure surfaced to stderr; no hang.
- ⚪ Nits: none.

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
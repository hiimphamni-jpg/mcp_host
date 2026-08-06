# Plan — FEAT-00003: Stdio MCP client lifecycle

## Goal
Create `internal/mcpclient`, the Stdio MCP client lifecycle module. It must:
1. Spawn the configured Filesystem MCP child process (command/args from `policy.FilesystemPolicy`, never LLM-supplied).
2. Complete the MCP `initialize` handshake within `MCP_TIMEOUT`.
3. Run `tools/list` once and retain discovered tools for the invocation.
4. Call `tools/call` for a validated tool within `MCP_TIMEOUT`.
5. `Close` the client and terminate the child process on completion/failure/cancellation, leaving no orphan process.
6. Return typed, non-fatal errors for child-process exit, invalid JSON-RPC, initialization failure, and timeout — never crashing the host.
7. Expose an interface so `internal/agent` (future FEAT-00006) can use fakes (AC5).

Scope is strictly the MCP stdio lifecycle per TDD §5 and AC1/AC4. No renewal/heartbeat, no multi-server, no HTTP/SSE.

## Assumptions
- Uses `github.com/mark3labs/mcp-go@v0.57.0` (`client.NewStdioMCPClientWithOptions` auto-starts transport; explicit `c.Initialize`, `c.ListTools`, `c.CallTool`; `c.Close()`).
- Existing `internal/policy.FilesystemPolicy` supplies `Command()`, `Args()`, `Roots()`; its validation already rejects unsafe commands/empty args.
- New module lives at `internal/mcpclient` (TDD §2.1). It depends on `mcp-go` and `internal/policy` only; it must not depend on `internal/llm`/Gemini or `internal/agent`.
- `Config` values (MCP timeout) come from `internal/config.Config`.
- Testing: Go standard `testing`; a fake MCP subprocess (a tiny stdio server binary driven by `go test`) exercises integration paths; no live Filesystem server in this feature's test path (that is QA-00002).
- Typed errors defined in the package (e.g. `ErrInit`, `ErrDiscovery`, `ErrCall`, `ErrTimeout`, `ErrProcessExit`) implementing `error`, carrying a user-safe message (no credentials, host paths, or stack traces).

## Plan

### Step 1: Define the MCP client interface and typed errors
- **Files**: `internal/mcpclient/interface.go`, `internal/mcpclient/errors.go`
- **Agent**: /dev
- **Change**: Define `type Client interface` with `Initialize(ctx) error`, `ListTools(ctx) (*mcp.ListToolsResult, error)`, `CallTool(ctx, name string, args map[string]any) (*mcp.CallToolResult, error)`, `Close() error`. Define sentinel wrapped typed errors (`ErrProcessExit`, `ErrInitFailed`, `ErrTimeout`, `ErrInvalidResponse`) implementing `error`, with user-safe message helpers.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~5 min

### Step 2: Spawn + initialize lifecycle wrapper
- **Files**: `internal/mcpclient/client.go`
- **Agent**: /dev
- **Change**: Implement `StdioClient` that builds args `append(policy.Args(), policy.Roots()...)`, calls `client.NewStdioMCPClientWithOptions(policy.Command(), nil, args)`, then `Initialize` with a context bounded by `MCP_TIMEOUT` (`mcp.InitializeRequest`, `LATEST_PROTOCOL_VERSION`). Map any failure to typed errors. Cache `*client.Client`.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~7 min

### Step 3. Discovery: tools/list
- **Files**: `internal/mcpclient/client.go`
- **Agent**: /dev
- **Change**: Add `ListTools` calling `mcp.ListToolsRequest{}` against the cached client with a `MCP_TIMEOUT`-bounded context; return the discovered result. Detect process exit / invalid response and map to typed errors.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~5 min

### Step 4. Call: tools/call with timeout
- **Files**: `internal/mcpclient/client.go`
- **Agent**: /dev
- **Change**: Add `CallTool` building `mcp.CallToolRequest{Params: {Name, Arguments}}`, invoking `CallTool` under a `MCP_TIMEOUT`-bounded context. Return the `*mcp.CallToolResult`; on context deadline, error, or child exit, return the corresponding typed error (host remains usable). (Result-size truncation is scope of a later feature.)
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~6 min

### Step 5. Cleanup / Close (idempotent, no orphan)
- **Files**: `internal/mcpclient/client.go`
- **Agent**: /dev
- **Change**: Implement `Close` that terminates the child process and closes the transport; make it safe to call multiple times and on partially-initialized clients (start fail, cancelled context). Guard so a cancelled/failed context still yields a clean child termination.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~4 min

### Step 6. Integration test — fake stdio MCP server
- **Files**: `internal/mcpclient/client_integration_test.go`, `internal/mcpclient/testdata/fakeserver` (small stdio JSON-RPC server binary/testdata)
- **Agent**: /test
- **Change**: Add a tiny fake MCP stdio process that handles `initialize`, `tools/list` (one scripted tool), and `tools/call`, and honors a marker arg to exit/crash/time-out. Add tests verifying: happy-path initialize+discovery+call returns the expected tool result; child-process crash returns typed error and `Close` is safe; timeout returns `ErrTimeout`; `Close` after failure leaves no orphan. Mirrors AC1/AC4.
- **Verify**: `go test ./internal/mcpclient/... -count=1`
- **Duration**: ~10 min

### Step 7. Full verification + review pass
- **Files**: none new
- **Agent**: /dev
- **Change**: Run `go build ./... && go vet ./... && go test ./...`; confirm no orphan processes; list 🔴/🟠/🟡/⚪ findings; ensure host `cmd/server` still compiles unchanged.
- **Verify**: `go build ./... && go vet ./... && go test ./... -count=1`
- **Duration**: ~4 min

## Risks & Mitigations
- **mcp-go behavior drift** (e.g. auto-start transport): consulted v0.57.0 source explicitly; the wrapper isolates the third-party API so future upgrades don't leak into `internal/agent`.
- **Child-process leaks on early failure**: `Close` is written to be safe on partially-initialized states; integration test asserts a process is not left behind.
- **Flaky timeout tests**: use short but reliable timeout values and a fake server that deliberately blocks; keep the marker-based crash scenario deterministic.
- **Result-size / truncation omitted here**: explicitly out of scope (later feature) to keep FEAT-00003 minimal and reviewable.

## Rollback Plan
- `internal/mcpclient` is new; no existing behavior is changed (no edits to `config`/`policy`/`cmd/server`). Rollback = delete `internal/mcpclient/`. No migration or config change.

## Parallel Opportunities
- ❗ Steps 2–5 run sequentially on the same file (`client.go`) with `Initialize`→`List`→`Call`→`Close` hierarchy — no parallelism among them.
- ⚡ Step 1 (interface/errors) can be drafted in parallel with standing up the fake `testdata` server used in Step 6, since both define the contract before real `client.go` logic.
- Step 6 depends on Steps 2–5 being present, so it is not parallel with them.
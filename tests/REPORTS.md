# Test Reports

## FEAT-00001 - Bootstrap Go Module and CLI Composition Root

**Executed:** 2026-08-05 | **By:** Tester | **Environment:** Local Go toolchain

| ID | Scenario | Status | Evidence |
|---|---|---|---|
| TEST-00001 | Approved SDK baseline resolves | PASS | `go list -m google.golang.org/genai` returned `google.golang.org/genai v1.66.0`. |
| TEST-00002 | Bootstrap does not disclose a credential | PASS | With `GEMINI_API_KEY=test-secret`, `go run ./cmd/server` returned only the expected bootstrap status. |
| TEST-00003 | Later-feature configuration remains inert at bootstrap | PASS | With representative Gemini and MCP configuration variables set, `go run ./cmd/server` returned only the expected bootstrap status. |

**Unit-test adequacy:** The composition-root unit test asserts exact output with a representative credential set. There is no input parser, configuration validation, MCP lifecycle, provider integration, API, or UI behavior in `FEAT-00001`; those tests belong to their owning follow-on tasks.

**Result:** TEST-00001 through TEST-00003 passed. No defects found.

---

## FEAT-00002 - Config Validation and Filesystem MCP Process Policy

**Executed:** 2026-08-06 | **By:** Tester | **Environment:** Local Go toolchain (go1.26.5 windows/amd64)

### Unit tests (`go test ./internal/config ./internal/policy ./cmd/server -v`)

| ID | Scenario | Test | Status | Evidence |
|---|---|---|---|---|
| TEST-00004 | Valid full config loads and values parse | `TestConfigLoad_Valid` | PASS | All fields parsed (`MCPTimeout=30s`, `LLMTimeout=45s`, `iterations=10`, `bytes=65536`) |
| TEST-00005 | Missing required field rejected | `TestConfigLoad_MissingRequired` | PASS | Error mentions `GEMINI_API_KEY` |
| TEST-00006 | Malformed args JSON rejected | `TestConfigLoad_MalformedArgsJSON` | PASS | Error mentions `MCP_FILESYSTEM_ARGS_JSON` |
| TEST-00007 | Zero/non-positive limits rejected | `TestConfigLoad_ZeroOrNegativeRejected` | PASS | `AGENT_MAX_ITERATIONS=0` rejected |
| TEST-00008 | Wrong-type duration/int rejected | `TestConfigLoad_BadDuration`, `TestConfigLoad_BadInt` | PASS | Errors name the variable |
| TEST-00009 | Config error never leaks API key | `TestConfigLoad_SecretNotLeaked` | PASS | `supersecret-abc123` absent from error |
| TEST-00010 | Default `HOST_LOG_LEVEL` applied | `TestConfigLoad_Valid` (LogLevel assert) | PASS | `LogLevel == "info"` |
| TEST-00011 | Bare valid executable accepted | `TestPolicy_NewValidCommand` | PASS | `Command()=="npx"`, args preserved |
| TEST-00012 | Unsafe/non-bare command rejected | `TestPolicy_NewRejectsUnsafeCommand` | PASS | 8 injection cases all rejected |
| TEST-00013 | Empty command rejected | `TestPolicy_NewRejectsEmptyCommand` | PASS | Error returned |
| TEST-00014 | File inside root allowed | `TestPolicy_CanonicalizesRoots` | PASS | `Contains==true`, roots absolute |
| TEST-00015 | Path outside root denied | `TestPolicy_ContainsDeniedOutsideRoot` | PASS | `Contains==false` |
| TEST-00016 | Parent traversal denied | `TestPolicy_ContainsDeniedParentTraversal` | PASS | `Contains==false` |
| TEST-00017 | `ParseArgs` valid JSON array | `TestPolicy_ParseArgs_ValidJSONArray` | PASS | 2 elements, `[0]=="-y"` |
| TEST-00018 | `ParseArgs` rejects non-string/invalid | `TestPolicy_ParseArgs_RejectsNonStringArray` | PASS | All error cases fail |
| TEST-00019 | Invalid config fails fast, no MCP spawn, no secret leak | `go run ./cmd/server` (E2E) | PASS | Exit 1; stderr `invalid configuration: ... GEMINI_API_KEY is required...`; sentinel secret absent; no MCP process spawned |

### Command-line verification (E2E — TEST-00019)

- **Invalid config (no `GEMINI_API_KEY`):** `go run ./cmd/server` → stderr `invalid configuration: GEMINI_API_KEY is required GEMINI_MODEL is required MCP_FILESYSTEM_COMMAND is required ...`; **exit=1**. No MCP spawned.
- **No-leak:** with `GEMINI_API_KEY=SENTINEL-leaky-key-xyz`, output does **not** contain the sentinel.
- **Valid config regression:** with full env, exits 0 and prints `MCP Host bootstrap complete. Integrations are not configured or started.` (FEAT-00001 bootstrap preserved).
- **Full suite:** `go test ./...` all PASS.

**Result:** TEST-00004 through TEST-00019 all PASS. No defects found.

---

## FEAT-00003 - Stdio MCP Client Lifecycle

**Executed:** 2026-08-06 | **By:** Tester | **Environment:** Local Go toolchain (go1.26.5 windows/amd64)

### Unit + integration (`go test ./internal/mcpclient -v -count=1`)

| ID | Scenario | Test | Status | Evidence |
|---|---|---|---|---|
| TEST-00020 | Happy path — initialize/list/call | `TestStdioClient_HappyPath_InitializeListCall` | PASS | Initialize ok; 1 tool `echo`; call content `ok`, IsError false |
| TEST-00021 | Constructor rejects nil policy / zero timeout | `TestStdioClient_InvalidPolicyAndTimeoutArg` | PASS | Both return errors |
| TEST-00022 | Child crash → ErrProcessExit + idempotent Close | `TestStdioClient_Crash_ReturnsTypedErrorAndCloseIsSafe` | PASS | `errors.Is(ErrProcessExit)`; second Close nil |
| TEST-00023 | Never-responding child → ErrTimeout (200ms) | `TestStdioClient_Timeout_ReturnsErrTimeout` | PASS | `errors.Is(ErrTimeout)`; Close clean |
| TEST-00024 | Server protocol error → ErrInvalidResponse | `TestStdioServer_InvalidResponse_ReturnsErrInvalidResponse` | PASS | `errors.Is(ErrInvalidResponse)` |
| TEST-00025 | Close leaves no orphan child | `TestStdioServer_Close_LeavesNoOrphan` | PASS | exit-marker written within 3s |
| TEST-00026 | Typed errors unwrap to sentinels | `TestError_UnwrapMatchesSentinel` | PASS | All 5 Kinds; never ErrTimeout |
| TEST-00027 | Error messages user-safe (no path/secret) | `TestError_MessageIsUserSafe`, `TestTimeoutError_IsErrTimeout` | PASS | No `\`, no `secret`/`token`; operation named |

**Full suite:** `go test ./...` all PASS. Coverage `internal/mcpclient` = **86.4%** (≥ 80% per TDD §11).

**Result:** TEST-00020 through TEST-00027 all PASS (8 scenarios). No defects found.

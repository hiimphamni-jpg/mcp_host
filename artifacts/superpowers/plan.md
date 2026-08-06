# Implementation Plan - FEAT-00002

## Goal

Implement configuration validation and the safe Filesystem MCP process policy: an `internal/config` package that parses and validates the documented environment contract (fail fast, never disclose secrets), and an `internal/policy` package that enforces the Filesystem process allowlist, canonical root containment, and safe error semantics defined in the TDD. Wire fail-fast config validation into the `cmd/server` composition root without spawning or initializing MCP (that is `FEAT-00003`).

## Assumptions

- `docs/tdd/MCP_HOST_TDD.md` §4 (Configuration Contract) and §8 (Security Policy) are the authoritative source for the env contract and executable/root rules.
- `docs/business-logic.md` state machine: `ConfigValidation` failure returns a non-zero exit and a concise user-safe message; secrets are never printed.
- `internal/config` must have no application dependency; `internal/policy` is pure and testable without `mcp-go`.
- GO 1.26.5 module already pins `github.com/mark3labs/mcp-go` and `google.golang.org/genai`, but FEAT-00002 must NOT import or invoke them — only stdlib, `os`, `os/exec`, `path/filepath`, `encoding/json`, `time`.
- `godotenv` is an approved baseline dependency (`go.mod`) and may be used for local `.env` loading, per TDD §3.
- MCP spawn, initialize, and `tools/list` belong to `FEAT-00003` and `FEAT-00004`; this task validates and prepares the process policy only.
- Existing `cmd/server/main.go` bootstrap test must keep passing; composition wiring is additive and minimal.

## Plan

### Step 1: Implement `internal/config` load and validation
- **Files**: `internal/config/config.go`, `internal/config/config_test.go`
- **Agent**: `/dev`
- **Change**: Define a `Config` struct with all §4 fields typed for `time.Duration`/`int`. `Load` reads env (optionally via `.env`); `Validate` returns aggregated field-level errors without echoing secret values. Required: `GEMINI_API_KEY`, `GEMINI_MODEL`, `MCP_FILESYSTEM_COMMAND`, `MCP_FILESYSTEM_ARGS_JSON` (JSON array), `MCP_ALLOWED_ROOTS`, `MCP_TIMEOUT`, `LLM_TIMEOUT`, `AGENT_MAX_ITERATIONS`, `MCP_MAX_RESULT_BYTES`. Optional: `HOST_LOG_LEVEL` defaults to `info`. Durations and ints must parse and be positive/bounded.
- **Verify**: `go test ./internal/config -run TestConfig` covers valid, missing-required, malformed-JSON, bad-duration, zero/bound; `go vet ./internal/config`
- **Duration**: ~8 min

### Step 2: Implement `internal/policy` process allowlist and root containment
- **Files**: `internal/policy/policy.go`, `internal/policy/policy_test.go`
- **Agent**: `/dev`
- **Change**: Implement `FilesystemPolicy` that (a) allows only the exact configured executable (no separators/path components, reject `|`, `;`, `&&`, whitespace-delimited injected args — the child argv is the fixed decoded JSON array, never LLM-supplied), (b) canonicalizes `MCP_ALLOWED_ROOTS` to absolute clean paths via `filepath.Abs`+`EvalSymlinks`, (c) provides a containment check that denies any requested path outside the canonical roots, and (d) validates args decode to a JSON array of strings.
- **Verify**: `go test ./internal/policy -run TestPolicy`; `go vet ./internal/policy`
- **Duration**: ~20 min

### Step 3: Wire fail-fast validation into `cmd/server` composition
- **Files**: `cmd/server/main.go`, `cmd/server/main_test.go`
- **Agent**: `/dev`
- **Change**: Update `run` to load and validate config, and return a user-safe diagnostic error + non-zero exit on invalid config, still avoiding MCP spawn and without printing secrets. Keep the existing bootstrap test green; add cases proving invalid config exits non-zero and no secret is echoed.
- **Verify**: `go test ./...`; `go run ./cmd/server` (expect user-safe failure if `GEMINI_API_KEY` unset) 
- **Duration**: ~15 min

### Step 4: Format and run repository quality gates
- **Files**: `internal/config/*.go`, `internal/policy/*.go`, `cmd/server/*.go`
- **Agent**: `/dev` and `/test`
- **Change**: `gofmt` all touched Go files, run `go mod tidy` if this task added imports, then run repository gates.
- **Verify**: `gofmt -w <files>`; `go test ./...`; `go vet ./...`
- **Duration**: ~5 min

## Risks & Mitigations

- **Allowlist bypass via argv injection**: Child command is fixed to the configured executable with a fixed JSON argv; reject any config using shell metacharacters or bare command strings instead of an explicit argv array. Cover with policy tests.
- **Root-escape / IDOR via symlinks**: canonicalization uses `filepath.Abs` + `EvalSymlinks` so a symlink outside a root is denied. Mitigated and unit-tested.
- **Secret leakage in errors/logs**: ensure `config.Validate` and the `cmd/server` diagnostic never format `GEMINI_API_KEY` or argv contents. Add an explicit test asserting the error string excludes the secret.
- **Scope creep into FEAT-00003**: any process-spawn / initialize code is prohibited here; policy only validates and returns configuration. If wiring tempts spawning, stop for `/rethink superspread`.
- **Parallel edits to `cmd/server`**: Step 3 edits composition root; Steps 1–2 touch distinct packages, so they can run first, then Step 3 sequentially.

## Rollback Plan

Remove `internal/config` and `internal/policy`, and revert `cmd/server/main.go`/`main_test.go` to the FEAT-00001 revisions. No data or external state is changed. Re-run `go test ./...` and `go vet ./...` after rollback.

## Parallel Opportunities

⚡ Step 1 (`internal/config`) and Step 2 (`internal/policy`) are independent package additions and can run concurrently. Step 3 depends on both and must run after. Step 4 must run last as the final gate.
# Build Execution Log — FEAT-00002

Mode: SEQUENTIAL

## Step 1: Implement `internal/config` load and validation
- Files: `internal/config/config.go`, `internal/config/config_test.go`
- Changes:
  - Added `Config` struct with all §4 fields; `Load` reads env (best-effort `.env` via godotenv) and parses JSON args, durations, and ints with aggregated field errors.
  - `Validate` returns aggregated errors for missing required fields and non-positive/bound values, never echoing the API secret.
  - Tests cover valid, missing-required, malformed-JSON, bad-duration, zero/negative, bad-int, and secret-not-leaked.
- Verify: `go test ./internal/config -run TestConfig` → PASS; `go vet ./internal/config` → PASS
- Notes: (red) helper name `validConfig`→`validEnv`; tuning `SecretNotLeaked` test to force a validation error while keeping the secret assertion.

## Step 2: Implement `internal/policy` process allowlist and root containment
- Files: `internal/policy/policy.go`, `internal/policy/policy_test.go`
- Changes:
  - `FilesystemPolicy` validates the configured executable (bare name, no path separators/shell metacharacters), canonicalizes allowed roots via `filepath.Abs`+`EvalSymlinks`, and provides a `Contains` containment check denying paths outside canonical roots.
  - `ParseArgs` enforces the JSON array-of-strings contract; child argv stays fixed and never LLM-supplied.
  - Tests cover valid/unsafe/empty command, canonical roots, inside/outside/parent-traversal containment, and JSON arg parsing.
- Verify: `go test ./internal/policy -run TestPolicy` → PASS; `go vet ./internal/policy` → PASS
- Notes: (red) test helper cleanup (removed unused `strings`, direct `os.WriteFile`).

## Step 3: Wire fail-fast validation into `cmd/server` composition
- Files: `cmd/server/main.go`, `cmd/server/main_test.go`
- Changes:
  - `run` now calls `config.Load()` first and returns a user-safe diagnostic on invalid config; `main` maps any error to a non-zero exit.
  - No MCP spawn/initialize code was added (scope stays in FEAT-00003).
  - Bootstrap test updated to a full valid env; new tests assert invalid config errors, no bootstrap output on failure, and no secret leakage in error/output.
- Verify: `go test ./cmd/server` → PASS; `go vet ./cmd/server` → PASS; `go run ./cmd/server` → `invalid configuration: ...` + `exit status 1` (PASS)
- Notes: none.

## Step 4: Format and run repository quality gates
- Files: `internal/config/*.go`, `internal/policy/*.go`, `cmd/server/*.go`
- Changes:
  - `gofmt -w` on all six touched Go files.
  - `go mod tidy` removed the not-yet-imported `mcp-go`/`genai` pins reserved for FEAT-00003/00004; reverted `go.mod`/`go.sum` since this task added no new imports (only the already-pinned `godotenv` is used).
- Verify: `gofmt -w <files>` → OK; `go test ./...` → PASS; `go vet ./...` → PASS
- Notes: deviation — skipped `go mod tidy` retention to keep future-feature dependency pins intact.
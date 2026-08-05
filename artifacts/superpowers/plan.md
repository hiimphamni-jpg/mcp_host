# Implementation Plan - FEAT-00001

## Goal

Establish a compiling, testable Go bootstrap for the MCP Host: approved baseline dependencies and a CLI composition root with deterministic, user-safe behavior. Configuration, MCP lifecycle, Gemini calls, agent behavior, and prompt handling remain assigned to `FEAT-00002` through `FEAT-00007`.

## Assumptions

- `docs/tdd/MCP_HOST_TDD.md` is the approved technical design; no additional architecture phase is required.
- The Google Gen AI SDK is an approved direct dependency baseline, but it must not be invoked in this task.
- The existing `execution.md` is not sufficient verification evidence because its claimed behavior differs from the present source. This task will establish fresh verification during `/build`.
- Empty future package directories are not created because Go does not retain them; their documented boundaries are preserved by keeping the bootstrap free of speculative implementations.

## Plan

### Step 1: Align the dependency baseline
- **Files**: `go.mod`, `go.sum`
- **Agent**: `/dev`
- **Change**: Add the official Google Gen AI Go SDK as a direct module dependency alongside the existing MCP and dotenv baselines, resolving checksums without adding provider behavior. Do not run `go mod tidy`: all three baseline dependencies are intentionally retained ahead of their owning feature tasks.
- **Verify**: `go get google.golang.org/genai@latest` followed by `go list -m google.golang.org/genai`
- **Duration**: ~3 min

### Step 2: Specify safe bootstrap behavior with a failing test
- **Files**: `cmd/server/main_test.go`
- **Agent**: `/dev`
- **Change**: Add a focused unit test for the composition root's deterministic bootstrap output. The test must prove it does not inspect or disclose environment-derived credentials and does not claim MCP or Gemini is initialized.
- **Verify**: `go test ./cmd/server -run TestRun`
- **Duration**: ~4 min

### Step 3: Replace the placeholder composition root
- **Files**: `cmd/server/main.go`
- **Agent**: `/dev`
- **Change**: Remove dotenv loading, MCP client placeholders, credential detection, and readiness claims. Implement a minimal, testable `run` boundary called by `main` that emits a concise bootstrap-only message; leave runtime configuration and integrations to their assigned features.
- **Verify**: `go test ./cmd/server -run TestRun` and `go run ./cmd/server`
- **Duration**: ~5 min

### Step 4: Format and verify the repository baseline
- **Files**: `go.mod`, `go.sum`, `cmd/server/main.go`, `cmd/server/main_test.go`
- **Agent**: `/dev` and `/test`
- **Change**: Format touched Go source and run the required repository quality checks. Record the actual results in the `/build` execution artifact.
- **Verify**: `gofmt -w cmd/server/main.go cmd/server/main_test.go`, `go test ./...`, and `go vet ./...`
- **Duration**: ~4 min

## Risks & Mitigations

- Google SDK version compatibility may conflict with the declared Go version. Resolve it through Go modules and stop for `/fix` if the quality checks fail.
- `go mod tidy` will prune intentionally unused baseline dependencies before their adapters are implemented. Defer tidy until the appropriate feature task imports each dependency.
- A bootstrap message could be mistaken for service readiness. State explicitly that integrations are not configured or started.
- Adding package stubs would create misleading APIs before their task-level contracts exist. Preserve only the documented boundary in the TDD until each owning feature begins.

## Rollback Plan

Revert the four task files to their pre-build revisions and run `go mod tidy` to restore the previous module graph. No data, external processes, or configuration state is changed by this task.

## Parallel Opportunities

No implementation steps can run in parallel: the test in Step 2 defines the behavior that Step 3 implements, and Step 4 verifies the resulting module and source changes.

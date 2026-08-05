# Brainstorm - FEAT-00001

## Goal

Establish a clean, compiling Go foundation for the MCP Host: the module and approved baseline dependencies, a `cmd/server` composition root, and repository structure that lets later tasks add configuration, MCP, LLM, policy, and agent implementations behind the documented boundaries.

## Constraints

- Do not implement configuration validation, MCP lifecycle, Gemini integration, agent behavior, or final prompt handling; those belong to `FEAT-00002` through `FEAT-00007`.
- `cmd/server` may depend on `internal/*`; future `internal/agent` must not depend directly on MCP or Gemini SDKs.
- The Go version and dependencies must remain compatible with the declared technical design.
- The composition root must avoid printing secret-derived state or pretending that unimplemented integrations are operational.
- Implementation is prohibited until a detailed `/plan` is explicitly approved.

## Known Context

- `docs/REGISTRY.md` defines `FEAT-00001` as P0/S1: bootstrap Go module, dependency baseline, and CLI composition root.
- The technical design in `docs/tdd/MCP_HOST_TDD.md` already defines the module boundaries and composition-root responsibility.
- The current repository contains `go.mod`, `go.sum`, `.env.example`, and a placeholder `cmd/server/main.go`; no `internal/*` packages or tests exist yet.
- The registry marks DEV complete, while `artifacts/superpowers/execution.md` records a broad "Transport Layer" setup. This Flow 1 run must establish concrete verification evidence before later test and QC phases can close the task.

## Risks

- Adding provider or transport behavior during bootstrap would overlap later feature tasks and make interfaces harder to test.
- Importing concrete MCP or Gemini libraries in domain packages would violate the dependency boundary.
- A placeholder executable can falsely report readiness while no configuration or protocol behavior exists.
- Dependency version drift may make the documented Go version or planned adapters unavailable.

## Options

1. Retain the current placeholder executable and only verify that dependencies compile.
2. Make `cmd/server` a minimal composition root that delegates to a small application boundary, leaving all runtime behavior for later features.
3. Build configuration, MCP client, and Gemini wiring now to prove an end-to-end startup path.

## Recommendation

Use option 2. Keep the foundation deliberately thin: establish the intended package structure and a deterministic composition-root boundary, with tests or compile checks that demonstrate the baseline. This respects the task boundary and leaves configuration and adapters to their assigned follow-on tasks. No new architecture design is required because `docs/tdd/MCP_HOST_TDD.md` already defines this durable structure.

## Acceptance Criteria

- The repository remains a valid Go module using the Go version and baseline dependencies approved in the TDD.
- `cmd/server` is the CLI composition root and does not embed MCP protocol, Gemini, configuration-validation, or agent-loop behavior assigned to later tasks.
- Package structure is ready for the documented `internal/config`, `internal/mcpclient`, `internal/llm`, `internal/agent`, and `internal/policy` boundaries without creating speculative implementations.
- The executable has deterministic, user-safe bootstrap behavior and does not report unavailable integrations as initialized.
- `go test ./...` and `go vet ./...` pass after implementation.

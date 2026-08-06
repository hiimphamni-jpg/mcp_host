# Plan — FEAT-00004: MCP JSON Schema → Gemini function-declaration mapper

## Goal
Create `internal/mapping`, which converts discovered MCP `mcp.Tool` definitions into Gemini
`*genai.FunctionDeclaration` objects. Per TDD §6 and AC2 / business-logic §6, the mapping must:
1. Support the MVP JSON Schema subset: object properties + nested objects; `string`, `number`,
   `integer`, `boolean`, `array`; `enum`, `required`, and array `items`.
2. Preserve required fields, enums, and nested properties losslessly.
3. Reject unsupported or malformed schemas with an error that includes the **tool name and field
   path** (TDD §6).
4. Reject duplicate tool names rather than silently overwriting them (TDD §6, multi-server safety).
5. Depend only on `mcp-go` types (input) and `genai` types (output). It must **not** depend on
   `internal/agent`, `internal/llm`, or `internal/mcpclient`, keeping `agent` SDK-free (AC5).

Scope is strictly the mapper (TDD §6 + AC2). Schema-based **argument validation** of a received
tool call (the "same schema used to validate tool-call arguments") is deferred to the agent loop
(FEAT-00006), which will reuse the built `*genai.Schema`.

## Assumptions
- Input type is `mcp.Tool` with `Name`, `Description`, and `InputSchema` (`ToolInputSchema` exposing
  `Type`, `Properties map[string]any`, `Required []string`, `AdditionalProperties`).
- Property schemas are `map[string]any` with a `"type"` string plus optional `"description"`,
  `"enum"` (JSON array), `"properties"` (nested), `"items"` (array item), `"required"` (nested
  object), and scalar constraints. Concrete helper keys come from `mcp-go` `With*` property options.
- Output is `*genai.FunctionDeclaration{Name, Description, Parameters *genai.Schema}`. The
  `genai.Schema` uses `genai.Type*` enums (`TypeString/NUMBER/INTEGER/BOOLEAN/ARRAY/OBJECT`),
  `Properties map[string]*genai.Schema`, `Required []string`, `Enum []string`, and `Items *genai.Schema`.
- New package lives at `internal/mapping` (TDD §2.1 doesn't list a mapper module; a dedicated
  cross-adapter package is cleanest so both `agent` and `llm` stay SDK-free and the conversion is
  unit-testable in isolation). Dependency direction: `mapping` → `mcp-go` + `genai` only.
- MVP runs a single MCP server (business-logic §7), but duplicate tool-name detection is still
  implemented so a future multi-server host fails safe.
- Typed errors in `internal/mapping` (e.g. `ErrUnsupportedType`, `ErrDuplicateName`) wrap context
  via helper `fmt.Errorf("tool %q field %q: %w", ...)`.

## Plan

### Step 1: Define package surface, public entrypoint, and typed errors
- **Files**: `internal/mapping/mapping.go`, `internal/mapping/errors.go`
- **Agent**: /dev
- **Change**: Define `func MapTools(tools []mcp.Tool) ([]*genai.FunctionDeclaration, error)` as the
  public entrypoint. Define package errors `ErrUnsupportedType`, `ErrDuplicateName`, `ErrMalformed`
  (unexported sentinel + exported `Error` type wrapping name/path) implementing `error`. Declare the
  internal helpers this package will use (`mapSchema`, `mapType`) as signatures so tests are stable.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~5 min

### Step 2. Primitive type + scalar field mapping
- **Files**: `internal/mapping/schema.go`
- **Agent**: /dev
- **Change**: Implement `mapType(t string, path string) (genai.Type, error)` mapping the supported
  strings (`object`,`string`,`number`,`integer`,`boolean`,`array`) to `genai.Type` enums and
  rejecting anything else (incl. `null`, empty, `any`) with a path-bearing `ErrUnsupportedType`.
  Implement conversion of a scalar property `map[string]any` → `*genai.Schema` (Type, Description,
  Enum from JSON array of strings, per-type bound constraints if present).
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~6 min

### Step 3. Nested objects, arrays/items, and required — recursive schema builder
- **Files**: `internal/mapping/schema.go`
- **Agent**: /dev
- **Change**: Implement `mapSchema(props map[string]any, required []string, path string)`
  (`*genai.Schema, error`) that builds `Properties`/`Required`, recursing into object `properties`
  and array `items`. Every rejection is annotated with tool field path (e.g. `tool.foo.bar`). Guard
  against cyclic/oversized recursion depth to avoid stack overflow on malformed schemas.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~7 min

### Step 4. Tool entrypoint: name/description/parameters + duplicate rejection
- **Files**: `internal/mapping/mapping.go`
- **Agent**: /dev
- **Change**: Implement `MapTools` that iterates `mcp.Tool` list, tracks seen names, returns
  `ErrDuplicateName` on a repeat, and builds each `*genai.FunctionDeclaration` using the symlink
  `Tool.Name`, `Description`, and `mapSchema` for the `InputSchema` (root `Parameters`). Preserve
  `required`, `enum`, nested properties.
- **Verify**: `go build ./... && go vet ./...`
- **Duration**: ~5 min

### Step 5. Unit table tests for mapping + rejection paths (AC2)
- **Files**: `internal/mapping/mapping_test.go`
- **Agent**: /test
- **Change**: Table tests covering: full supported subset preserving required/enum/nested
  properties/arrays (mirrors AC2); empty properties (no-arg tool); each unsupported type/malformed
  schema returning a typed error whose message includes the tool name and field path; duplicate
  tool names returning `ErrDuplicateName`; deep-nesting recursion guard.
- **Verify**: `go test ./internal/mapping/... -count=1`
- **Duration**: ~10 min

### Step 6. Full verification + gofmt + review pass
- **Files**: none new
- **Agent**: /dev
- **Change**: Run `gofmt -l .` (expect empty), `go build ./...`, `go vet ./...`, `go test ./...`;
  confirm no existing packages (`config`/`policy`/`mcpclient`/`cmd/server`) changed or broken;
  list 🔴/🟠/🟡/⚪ findings.
- **Verify**: `gofmt -l .; go build ./...; go vet ./...; go test ./... -count=1`
- **Duration**: ~4 min

## Risks & Mitigations
- **Property key variance from MCP servers**: Filesystem server may emit `$defs`/`definitions`
  (mcp-go `ToolArgumentsSchema.Defs`). The mapper assumes inline schemas (MVP subset per TDD §6);
  a schema using `$ref`/`$defs` is treated as unsupported with a clear tool/path error rather than
  silently mis-mapped. Mitigated: `/ba` confirm the Filesystem server stays within the MVP subset
  during AC2 integration review (QA-00001/QA-00002).
- **`enum` values as `[]any` vs `[]string`:** handle both via a type-asserting helper, rejecting
  non-stringable enum members with path context.
- **Recursion on malformed cyclic schema:** guard recursion depth; reject exceeding it.
- **gemini/genai `Type` drift:** isolated to `mapType`; a table maps the string subset to enums and
  fails loudly on drift.
- **Mapper placement not in TDD §2.1 module list:** documented assumption; keeps `agent` clean and
  is a small, isolated, easily-relocated package.

## Rollback Plan
`internal/mapping/` is brand new; no existing file is modified (no edits to `config`/`policy`/
`mcpclient`/`cmd/server`). Rollback = delete `internal/mapping/`. No migration or config change.

## Parallel Opportunities
- ⚡ Step 1 (package API + error contract) can be authored in parallel with Step 2's standalone
  `mapType`/types, since the scalar mapping is independent of the entrypoint.
- ⚡ Step 5 test fixtures can be drafted against the Step 1 signatures while Steps 2–4 implement the
  logic.
- ❗ Steps 2–4 share `schema.go`/`mapping.go` and build dependencies (entrypoint → schema builder →
  type mapping), so those run sequentially.
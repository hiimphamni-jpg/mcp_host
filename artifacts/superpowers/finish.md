# /build Finish Summary — FEAT-00004: MCP JSON Schema → Gemini function-declaration mapper

## Status: IMPLEMENTED (ready for `/finish` QC sign-off)

## What was built
New module `internal/mapping` (only; no edits to config/policy/mcpclient/cmd/server):

| File | Purpose |
|------|---------|
| `mapping.go` | `MapTools([]mcp.Tool) ([]*genai.FunctionDeclaration, error)` — entrypoint that tracks seen names (rejects duplicates with `ErrDuplicateName`), rejects empty names, and builds root object `Parameters` losslessly. Depends only on `mcp-go` + `genai` (AC5: agent stays SDK-free). |
| `errors.go` | Sentinels `ErrUnsupportedType`, `ErrDuplicateName`, `ErrMalformed`; typed `*Error{Cause, Tool, Path, Description}` with `Unwrap` (errors.Is) formatted `tool "X" field "p": reason` (TDD §6 tool-name + field-path requirement). |
| `schema.go` | `mapType` (MVP subset → genai.Type enums, rejects `null`/`any`/empty/unknown), `mapSchemaValue`/`mapObject`/`mapArray` recursive builder preserving enum/nested `required`/items; `maxRecursionDepth = 64` cyclic/deep-nesting guard; JSON-decoded `[]any` shapes for nested `required` and `enum` handled losslessly. |
| `mapping_test.go` | AC2 full-subset lossless test (reflect.DeepEqual), no-arg tool, unsupported-type table, nested-path rejection, malformed property/required/properties/items, non-string enum member, duplicate-name, cyclic-schema and deep-nesting recursion guards. |

## Verification evidence
- TDD RED confirmed: tests failed before implementation (package undefined), GREEN after.
- `gofmt -l .` → clean (empty)
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (cmd/server, config, policy, mcpclient, mapping; 28 mapping subtests green)
- `go test ./internal/mapping/... -count=1 -cover` → 83.1% coverage (target ≥80%, QA-00001)
- `git status` → only untracked `internal/mapping/`; `plan.md` pre-existing uncommitted FEAT-00004 plan (not touched); no diff to config/policy/mcpclient/cmd/server

## Review pass
- 🔴 **Blockers:** none.
- 🟠 **Majors:** none.
- 🟡 **Minors:**
  - `mapSchemaValue` silently ignores `additionalProperties`, `minLength`/`maxLength`, `pattern`, and format keys (out of MVP subset per TDD §6); if a later server relies on them they must be mapped or rejected explicitly.
  - Numeric bounds (`minimum`/`maximum`) are carried through but the `minimum` from mcp-go `Min()` arrives as `int`/`int64`/`float64`; `asFloat64` covers those plus JSON-decoded `float64` — a `json.Number` (rare custom decoder) would be rejected as malformed.
- ⚪ **Nits:**
  - `ErrDuplicateName` error message omits the field path (no field context for a duplicate name) — intentional, name is the path.

## Follow-ups
- `/test cases FEAT-00004` for formal TEST-xxxxx cases (tester role).
- `/qc audit FEAT-00004` then `/qc sign-off FEAT-00004`.
- FEAT-00006 will reuse the built `*genai.Schema` for tool-call argument validation (same schema used to validate received arguments, TDD §6); the deferred schema-based argument validation is out of this task's scope.
- Register DEV column → ✅ in `docs/REGISTRY.md` after QC sign-off.
- Risk watch (plan): confirm Filesystem server stays within MVP subset during AC2 integration review (QA-00001/QA-00002); `$defs`/`$ref` schemas are rejected with a clear tool/path error, not silently mis-mapped.

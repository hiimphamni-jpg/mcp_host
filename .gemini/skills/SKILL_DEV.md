---
name: Dev Skill
description: >
  Technical execution standards for software development tasks. Use this skill whenever the user is:
  writing or implementing code (features, components, hooks, API routes), setting up or running tests (TDD, unit tests, integration tests),
  preparing a PR or commit (branch naming, checklist, description), refactoring or reviewing code,
  handling errors or defining error patterns, working on Next.js / TypeScript / React / Go projects,
  or following naming conventions for files, functions, variables, or database migrations.
  Trigger even if the user just says "implement this", "write a test for", "create a PR", "add an API route", or "refactor".
---
 
# 💻 Dev Skill — Technical Execution Standards
 
---
 
## 1. Task Execution Workflow
 
```
1. Receive Task ID from PM
2. Read Spec + AC from BA
3. Read TDD from Architect (if exists)
4. Create branch: feature/{TASK-ID}-{kebab-description}
5. Write tests first (TDD where mandatory)
6. Implement code
7. Self-review (checklist below)
8. Create PR
9. Update Registry: DEV → ✅
```
 
### Branch Naming Convention
```
feature/{TASK-ID}-{short-kebab-description}   # new feature
fix/{TASK-ID}-{short-kebab-description}        # bug fix
chore/{TASK-ID}-{short-kebab-description}      # refactor, deps, config
migration/{TASK-ID}-{short-kebab-description}  # DB migration only
```
 
### Branch Announcement
Mỗi phản hồi khi bắt đầu task phải thông báo:
```
🔀 Branch: feature/{TASK-ID}-{short-description}
📋 Task: {TASK-ID} — {description}
```
 
---
 
## 2. TDD Workflow
 
```
┌───────────┐     ┌───────────┐     ┌───────────┐
│  🔴 RED   │────►│ 🟢 GREEN  │────►│ 🔵 REFAC  │──► repeat
│ Write     │     │ Minimal   │     │ Clean up  │
│ failing   │     │ code to   │     │ (tests    │
│ test      │     │ pass      │     │ still ✅) │
└───────────┘     └───────────┘     └───────────┘
```
 
### When TDD is Mandatory
- All calculation/formula functions
- All state machine transitions (e.g. session status, payment lock)
- All validation logic
- All data transformation functions
- All custom hooks with non-trivial logic
 
### When Test-After is OK
- Thin HTTP handlers / API routes (integration test)
- UI components (visual/snapshot testing)
- Config/setup code
 
### TypeScript/Next.js Test Pattern (Vitest / Jest)
```ts
// ✅ Good: describe → arrange → act → assert
describe('calculateOrderTotal', () => {
  it('returns 0 when items is empty', () => {
    expect(calculateOrderTotal([])).toBe(0)
  })
 
  it('sums item prices correctly', () => {
    const items = [{ price: 10000, qty: 2 }, { price: 5000, qty: 1 }]
    expect(calculateOrderTotal(items)).toBe(25000)
  })
})
```
 
### Go Test Pattern
```go
func TestCalculateOrderTotal(t *testing.T) {
    t.Run("returns 0 when items empty", func(t *testing.T) {
        result := CalculateOrderTotal([]Item{})
        assert.Equal(t, 0, result)
    })
}
```
 
---
 
## 3. Backend Architecture (Go)
 
### Layer Separation
```
internal/
├── domain/        # ❤️ Heart — Structs, Interfaces, Constants, Errors
│                  #    No imports from other layers
├── service/       # 🧠 Brain — Business logic
│                  #    Depends on: domain interfaces
│                  #    No DB knowledge
├── repository/    # 🤲 Hands — Data access (SQL/ORM)
│                  #    Implements: domain interfaces
└── handler/       # 🎭 Face — HTTP/WS transport
                   #    Depends on: service interfaces
                   #    Only request/response mapping
```
 
### File Organization
- 1 file = 1 concern (e.g., `session_service.go`, `session_repository.go`)
- Test file adjacent: `session_service_test.go`
- Constants file per package: `constants.go`
 
### Go Naming Conventions
| Item | Convention | Example |
|------|-----------|---------|
| Package | lowercase, single word | `service`, `repository` |
| Exported type | PascalCase | `SessionService` |
| Unexported | camelCase | `parsePayload` |
| Constants | PascalCase or ALL_CAPS | `MaxRetries`, `STATUS_LOCKED` |
| Error vars | `Err` prefix | `ErrSessionNotFound` |
| Interface | noun or `-er` suffix | `SessionRepository`, `Notifier` |
 
### Go Error Handling
```go
// ✅ Wrap errors with context
if err != nil {
    return fmt.Errorf("sessionService.Lock: %w", err)
}
 
// ✅ Sentinel errors in domain layer
var ErrSessionLocked = errors.New("session already locked")
 
// ✅ Check sentinel at handler layer
if errors.Is(err, domain.ErrSessionLocked) {
    http.Error(w, "session is locked", http.StatusConflict)
    return
}
```
 
---
 
## 4. Frontend Architecture (Next.js / React / TypeScript)
 
### Component Strategy
```
ui/            → Atoms (Button, Input, Badge) — reusable, no business logic
components/    → Organisms (OrderForm, ParticipantList) — compose atoms
app/           → Pages + Layouts only — thin, delegates to components
lib/           → Utilities, API clients, constants
hooks/         → Custom hooks — encapsulate logic away from JSX
types/         → Shared TypeScript interfaces and types
```
 
### Server vs Client Components
- **Default: Server Component** — no directive needed
- Add `"use client"` ONLY when needed: `useState`, `useEffect`, `onClick`, browser APIs
- Heavy logic → extract to hook or `lib/`, keep component "dumb"
 
### TypeScript Naming Conventions
| Item | Convention | Example |
|------|-----------|---------|
| Component file | PascalCase | `QRScannerSection.tsx` |
| Hook file | camelCase, `use` prefix | `useOrderSession.ts` |
| Utility/lib file | camelCase | `formatCurrency.ts` |
| Type/Interface | PascalCase | `OrderSession`, `PaymentStatus` |
| Enum | PascalCase | `SessionStatus` |
| Constants | SCREAMING_SNAKE_CASE | `MAX_RETRY_COUNT` |
| Props type | `{ComponentName}Props` | `QRScannerSectionProps` |
 
### API Route Conventions (Next.js App Router)
```
app/
└── api/
    └── {resource}/
        ├── route.ts          # GET (list), POST (create)
        └── [id]/
            └── route.ts      # GET (single), PATCH, DELETE
```
 
```ts
// app/api/sessions/route.ts
export async function GET(request: Request) { ... }
export async function POST(request: Request) { ... }
 
// app/api/sessions/[id]/route.ts
export async function PATCH(
  request: Request,
  { params }: { params: { id: string } }
) { ... }
```
 
**API Response shape — always consistent (ref: `rules/api-convention.md`):**
```ts
// Success
{
  data: T,
  meta: { request_id: string, timestamp: string }
}

// Collection (paginated)
{
  data: T[],
  meta: {
    request_id: string,
    timestamp: string,
    pagination: { page: number, page_size: number, total_items: number, total_pages: number }
  }
}

// Error
{
  error: { code: string, message: string, details?: FieldError[], trace_id: string }
}
```
 
### TypeScript Error Handling
 
```ts
// ✅ Define typed error codes
export type AppErrorCode =
  | 'SESSION_NOT_FOUND'
  | 'SESSION_LOCKED'
  | 'UNAUTHORIZED'
 
export class AppError extends Error {
  constructor(public code: AppErrorCode, message: string) {
    super(message)
    this.name = 'AppError'
  }
}
 
// ✅ In API route — always return structured error
try {
  const session = await getSession(id)
  return Response.json({ data: session, error: null })
} catch (err) {
  if (err instanceof AppError) {
    return Response.json(
      { data: null, error: { code: err.code, message: err.message } },
      { status: mapErrorToStatus(err.code) }
    )
  }
  return Response.json(
    { data: null, error: { code: 'INTERNAL_ERROR', message: 'Unexpected error' } },
    { status: 500 }
  )
}
 
// ✅ In hook/client — narrow error type before using
const { data, error } = await fetchSession(id)
if (error) {
  if (error.code === 'SESSION_LOCKED') { ... }
  toast.error(error.message)
  return
}
```
 
---
 
## 5. Database Migration Workflow
 
```
1. Design schema change (consult Architect if complex)
2. Write UP migration (follow rules/database.md)
3. Write DOWN migration (rollback)
4. Test on local DB
5. Include in PR with [MIGRATION] tag
```
 
### Migration Naming
```
{timestamp}_{description}.up.sql
{timestamp}_{description}.down.sql
```
 
---
 
## 6. PR Preparation Checklist
 
Trước khi tạo PR, Dev phải tự kiểm tra:
 
### Code Quality
- [ ] No magic numbers — all constants extracted
- [ ] No hardcoded URLs or secrets
- [ ] No `fmt.Println` / `console.log` in production code
- [ ] No unused variables or imports
- [ ] Functions ≤ 50 lines, files ≤ 300 lines
- [ ] TypeScript: no `any` type unless explicitly justified with comment
- [ ] TypeScript: all async functions have proper return type annotation
 
### Testing
- [ ] TDD: All unit tests pass (100%)
- [ ] Coverage maintains or improves baseline
- [ ] Edge cases tested (boundary values, null/undefined inputs)
- [ ] New API routes have at least one integration test
 
### Standards
- [ ] Logic matches spec from BA (100%)
- [ ] API response follows consistent `{ data, error }` shape
- [ ] Error handling follows patterns in Section 3 (Go) / Section 4 (TS)
- [ ] `go fmt` / `npm run lint` / `tsc --noEmit` clean
 
### Registry
- [ ] Updated DEV column → ✅ in `docs/REGISTRY.md`
- [ ] PR linked to Task ID
- [ ] Branch named correctly: `feature/{ID}-{kebab-desc}`
- [ ] PR description includes: **What**, **Why**, **How to test**
 
---
 
## 7. Code Review Response
 
When receiving review feedback:
- `[MUST]` → Fix before merge, no exceptions
- `[SUGGEST]` → Consider and respond, may decline with reason
- `[QUESTION]` → Answer in PR comments
- `[NITPICK]` → Fix if easy, otherwise note for future
 
---
 
## 8. Conflict Resolution
 
Nếu User yêu cầu viết code trái với spec của BA:
1. **Cảnh báo ngay lập tức**: "⚠️ Yêu cầu này mâu thuẫn với spec XYZ"
2. **Trích dẫn** phần spec bị ảnh hưởng
3. **Đề xuất** phương án thay thế
4. **Chỉ thực hiện** nếu User xác nhận override sau khi đã biết hậu quả
 
---
name: Code Style
description: Go and TypeScript/React coding standards
---

# 🎨 Code Style Standards

> **Principle:** Code is read 10x more than it is written. Optimize for the reader, not the writer.

---

## PART A — Go Standards

### 1. Formatting & Tooling
- **Always run** `go fmt` and `go vet` before commit
- **Linter**: `golangci-lint` with default config + `errcheck`, `gosimple`, `govet`, `staticcheck`
- **Import order**: stdlib → third-party → internal (separated by blank line)

### 2. File & Package Structure
```
internal/
├── domain/        # Structs, Interfaces, Constants (pure, no dependencies)
├── service/       # Business Logic (no DB knowledge, depends on domain)
├── repository/    # Data Access (SQL/ORM, implements domain interfaces)
└── handler/       # Transport Layer (HTTP/WS, request/response only)
```

### 3. Size Limits
| Element | Max Lines | Action if exceeded |
|---------|:---------:|-------------------|
| Function | 50 | Split into sub-functions |
| File | 300 | Split by responsibility |
| Struct methods | 10 | Consider splitting struct |

### 4. Naming
- **Packages**: short, lowercase, single word — `session`, `order`, `auth` (not `sessionService`)
- **Interfaces**: action-based — `Reader`, `SessionStore`, `OrderCalculator` (not `ISession`)
- **Exported funcs**: verb-first — `CalculateTotal()`, `FindByID()`
- **Unexported helpers**: descriptive — `parseAmount()`, `validateInput()`
- **Constants**: `PascalCase` for exported, `camelCase` for unexported
- **No stuttering**: `session.New()` ✅ — `session.NewSession()` ❌

### 5. Error Handling
- **Always wrap errors** with context: `fmt.Errorf("create order: %w", err)`
- **Guard clauses first** — return early on error, keep happy path unindented
- **No naked returns** in functions with named return values
- **No `_ = func()`** — every error must be checked or explicitly documented why ignored

```go
// ✅ Good — guard clause
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
    session, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get session %s: %w", id, err)
    }
    if session.IsExpired() {
        return nil, ErrSessionExpired
    }
    return session, nil
}

// ❌ Bad — deep nesting
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
    session, err := s.repo.FindByID(ctx, id)
    if err == nil {
        if !session.IsExpired() {
            return session, nil
        } else {
            return nil, ErrSessionExpired
        }
    } else {
        return nil, err
    }
}
```

### 6. Context Usage
- **First parameter** of every function that does I/O: `func DoSomething(ctx context.Context, ...)`
- **Never store** `context.Context` in a struct
- **Pass through** — don't create new context unless adding timeout/values

### 7. Interface-First Design
- Define interfaces in the **consumer package** (domain), not the provider
- Keep interfaces small (1–3 methods ideal)
- Every service/repository must have an interface for testability (mocking)

### 8. Logging
- Use **structured logging** (`slog` in Go 1.21+)
- Log levels: `Debug` (development), `Info` (business events), `Warn` (recoverable), `Error` (needs attention)
- **Never** use `fmt.Println` or `log.Println` in production code

---

## PART B — TypeScript / React Standards

### 1. Formatting & Tooling
- **ESLint** + **Prettier** — run before commit
- **Strict TypeScript**: `"strict": true` in `tsconfig.json`
- No `any` type — use `unknown` + type guards if needed

### 2. Component Architecture (Atomic Design)
```
ui/            # Atoms: Button, Input, Badge, Skeleton
components/    # Molecules/Organisms: OrderForm, ParticipantList
app/           # Pages + Layouts only (Next.js App Router)
lib/           # Utilities, hooks, constants
hooks/         # Custom React hooks
```

### 3. Component Rules
- **Server Components by default** — only add `"use client"` when needed (state, effects, browser APIs)
- **Dumb components**: UI components receive props, no internal API calls
- **Logic separation**: Heavy logic → custom hooks or `lib/`, not inside JSX
- **One component per file** — exception: small tightly-coupled sub-components

### 4. Naming
| Element | Convention | Example |
|---------|-----------|---------|
| Component file | `PascalCase.tsx` | `OrderCard.tsx` |
| Hook file | `camelCase.ts` | `useSession.ts` |
| Utility file | `camelCase.ts` | `formatCurrency.ts` |
| Constant file | `camelCase.ts` | `constants.ts` |
| Type/Interface | `PascalCase` | `SessionResponse` |
| CSS Module | `ComponentName.module.css` | `OrderCard.module.css` |

### 5. Imports
- **Barrel exports**: Use `index.ts` in component directories
- **Absolute imports**: Configure `@/` alias for `src/` or app root
- **Import order**: React → third-party → internal → types → styles

### 6. State Management
- **Local state**: `useState`, `useReducer`
- **Server state**: React Query / SWR / Server Actions
- **No prop drilling** beyond 2 levels — use Context or composition
- **Avoid global state** unless truly global (auth, theme)

### 7. Styling
- Use project's CSS framework consistently (Tailwind / CSS Modules)
- **Conditional classes**: `cn()` utility (clsx + tailwind-merge)
- **No inline styles** except truly dynamic values (e.g., `style={{ width: percentage }}`)

### 8. No Magic Values
```typescript
// ❌ Bad
if (status === 3) { ... }
setTimeout(fn, 86400000);

// ✅ Good
const SESSION_STATUS = { OPEN: 1, LOCKED: 2, ORDERED: 3 } as const;
const ONE_DAY_MS = 24 * 60 * 60 * 1000;

if (status === SESSION_STATUS.ORDERED) { ... }
setTimeout(fn, ONE_DAY_MS);
```

---

## PART C — Universal Rules

### 1. DRY (Don't Repeat Yourself)
- If code appears **twice**, consider extracting into a function
- If code appears **three times**, **must** extract

### 2. Comments
- **Don't comment WHAT** — the code should be self-explanatory
- **Comment WHY** — explain non-obvious decisions, workarounds, business rules
- **TODO format**: `// TODO(owner): description — TICKET-ID`

### 3. Dead Code
- **Delete it**. Don't comment it out. Git has history.
- No unused variables, imports, or functions

### 4. Constants
- **No hardcoded values** in business logic — extract to constants file
- Environment-specific values → environment variables
- Feature-specific constants → co-located with feature

---
name: Error Handling
description: Structured error handling, tracing, and zero-silence policy
---

# 🚨 Error Handling Standards

> **Principle:** Zero-Silence. Every error must be caught, classified, logged, and surfaced appropriately. A silent error is a ticking bomb.

---

## 1. Error Response Format

Every API error response must use this envelope:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Session with ID abc-123 was not found",
    "details": [],
    "trace_id": "trace_20250315_abc123"
  }
}
```

| Field | Type | Required | Purpose |
|-------|------|:--------:|---------|
| `code` | `string` | ✅ | Machine-readable error code (`UPPER_SNAKE_CASE`) |
| `message` | `string` | ✅ | Human-readable explanation |
| `details` | `array` | ❌ | Field-level errors for validation |
| `trace_id` | `string` | ✅ | Unique ID for log correlation |

---

## 2. Error Code Taxonomy

### Standard Codes (use across all projects)

| Code | HTTP | When |
|------|:----:|------|
| `VALIDATION_FAILED` | 400 | Request body/params invalid |
| `INVALID_FORMAT` | 400 | Malformed JSON, wrong content type |
| `UNAUTHORIZED` | 401 | Missing or invalid credentials |
| `FORBIDDEN` | 403 | Authenticated but not authorized |
| `RESOURCE_NOT_FOUND` | 404 | Entity doesn't exist |
| `RESOURCE_CONFLICT` | 409 | Duplicate, state violation |
| `BUSINESS_RULE_VIOLATION` | 422 | Valid data but business logic rejected |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | Dependency down, maintenance |

### Validation Details Format
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Request validation failed",
    "details": [
      { "field": "email", "reason": "required", "message": "Email is required" },
      { "field": "amount", "reason": "min", "message": "Amount must be > 0", "meta": { "min": 1 } }
    ],
    "trace_id": "trace_xyz"
  }
}
```

---

## 3. Backend Error Handling (Go)

### Error Wrapping — Always Add Context
```go
// ✅ Every layer adds context
func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*Session, error) {
    var s Session
    err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Name)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("repo find session %s: %w", id, err)
    }
    return &s, nil
}

func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
    session, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get session: %w", err)
    }
    return session, nil
}
```

### Sentinel Errors — Define Domain Errors
```go
package domain

import "errors"

var (
    ErrNotFound          = errors.New("resource not found")
    ErrConflict          = errors.New("resource conflict")
    ErrForbidden         = errors.New("action forbidden")
    ErrBusinessViolation = errors.New("business rule violation")
)
```

### Handler — Map Domain Errors to HTTP
```go
func mapError(err error) (int, string) {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return 404, "RESOURCE_NOT_FOUND"
    case errors.Is(err, domain.ErrConflict):
        return 409, "RESOURCE_CONFLICT"
    case errors.Is(err, domain.ErrForbidden):
        return 403, "FORBIDDEN"
    case errors.Is(err, domain.ErrBusinessViolation):
        return 422, "BUSINESS_RULE_VIOLATION"
    default:
        return 500, "INTERNAL_ERROR"
    }
}
```

### Zero-Silence Rules
- ❌ `_ = doSomething()` — NEVER ignore errors
- ❌ `if err != nil { return }` — always return the error
- ❌ Empty catch blocks
- ✅ If truly ignorable, document why: `_ = conn.Close() // best-effort cleanup`

### Logging Errors
```go
// Log at the handler level (edge), not deep inside service/repo
slog.Error("failed to create order",
    "error", err,
    "trace_id", traceID,
    "session_id", sessionID,
)
```

---

## 4. Frontend Error Handling (React/TypeScript)

### Error Boundary — Catch Rendering Errors
```tsx
// Wrap sensitive components
<ErrorBoundary fallback={<ErrorFallback />}>
  <OrderForm />
</ErrorBoundary>
```

- Every major page/feature must have an ErrorBoundary
- Fallback shows user-friendly message + retry button
- Log error to monitoring service

### API Error Handling
```typescript
// Typed error response
interface ApiError {
  code: string;
  message: string;
  details?: { field: string; reason: string; message: string }[];
  trace_id: string;
}

// Handle in data layer, surface in UI
try {
  const data = await api.createOrder(payload);
} catch (error) {
  if (isApiError(error)) {
    // Show user-friendly message based on error.code
    toast.error(getErrorMessage(error.code));
  } else {
    // Unknown error — log and show generic message
    captureException(error);
    toast.error("Something went wrong. Please try again.");
  }
}
```

### No Silent Failures
- ❌ Empty `.catch(() => {})` blocks
- ❌ `console.log(error)` as the only handling
- ✅ Every catch must either recover or notify the user
- ✅ Use `toast` / `alert` / inline error for user feedback

---

## 5. Input Validation

### Dual Validation — Both Sides, Always

| Layer | Tool | Purpose |
|-------|------|---------|
| Frontend | **Zod** | Fast UX feedback, type inference |
| Backend | **Go validator** / custom | Security gate, source of truth |

### Rules
- Frontend validation = UX convenience (can be bypassed)
- Backend validation = security enforcement (source of truth)
- **Never trust frontend** — always re-validate on backend
- Validation errors → `400 VALIDATION_FAILED` with field details

---

## 6. Tracing

- Every request gets a `trace_id` (generated at API gateway/first handler)
- `trace_id` included in all log entries for that request
- `trace_id` returned in error responses for user support
- Format: `trace_{timestamp}_{random}` or UUID

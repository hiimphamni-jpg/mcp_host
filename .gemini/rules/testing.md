---
name: Testing Standards
description: TDD methodology, test pyramid, coverage targets, and naming conventions
---

# 🧪 Testing Standards

> **Principle:** If it's not tested, it's broken — you just don't know it yet. Test pyramid, not test ice cream cone.

---

## 1. Test Pyramid

```
        ╱ E2E (10%) ╲           Slow, expensive, catches integration gaps
       ╱─────────────╲
      ╱ Integration    ╲        API contracts, DB queries, service boundaries
     ╱   (20%)          ╲
    ╱───────────────────╲
   ╱   Unit Tests (70%)   ╲     Fast, isolated, catches logic bugs
  ╱─────────────────────────╲
```

| Level | What | Speed | Isolation | Coverage Target |
|-------|------|:-----:|:---------:|:---------------:|
| Unit | Functions, methods, hooks | ⚡ <1s | Full (mocked) | ≥ 80% |
| Integration | API endpoints, DB queries | 🔵 <10s | Partial (test DB) | Critical paths |
| E2E | Full user flows | 🔴 <60s | None (real browser) | Core flows only |

### Anti-Pattern: Ice Cream Cone 🍦
❌ Many E2E → Slow CI, flaky tests, hard to maintain
✅ Many Unit → Fast CI, reliable, easy to debug

---

## 2. TDD Workflow (Red-Green-Refactor)

```
┌─────────┐    ┌─────────┐    ┌──────────┐
│  RED     │───►│  GREEN  │───►│ REFACTOR │───► repeat
│ Write    │    │ Make it │    │ Clean up │
│ failing  │    │ pass    │    │ (tests   │
│ test     │    │ (minimal│    │  still   │
│          │    │  code)  │    │  pass)   │
└─────────┘    └─────────┘    └──────────┘
```

### When TDD is Mandatory
- All business logic (calculations, state machines, rules)
- All data transformations
- All validation logic

### When TDD is Optional
- Thin handler/controller layers (integration test preferred)
- UI components (visual testing preferred)
- Third-party integrations (mock-based testing preferred)

---

## 3. Test File & Naming

### Go
```
session_service.go      → session_service_test.go
order_calculator.go     → order_calculator_test.go
```

- Test functions: `TestFunctionName_Scenario_Expected`
- Examples:
  - `TestCalculateTotal_WithDiscount_ReturnsReducedAmount`
  - `TestCreateSession_DuplicateName_ReturnsConflictError`
  - `TestParseAmount_NegativeValue_ReturnsValidationError`

### TypeScript / React
```
useSession.ts           → useSession.test.ts
OrderCard.tsx           → OrderCard.test.tsx
formatCurrency.ts       → formatCurrency.test.ts
```

- Test blocks: `describe('FunctionName', () => { it('should...') })`

---

## 4. Test Structure (AAA Pattern)

```go
func TestCalculateShipping_WithVoucher_ReducesTotal(t *testing.T) {
    // Arrange — setup test data
    order := NewOrder(items, shippingFee: 30000)
    voucher := Voucher{Type: "shipping", Amount: 15000}

    // Act — execute the function
    result := CalculateShipping(order, voucher)

    // Assert — verify the result
    assert.Equal(t, 15000, result.ShippingFee)
}
```

### BDD Format (for acceptance tests)
```gherkin
Given a session with 2 participants
  And participant A ordered 50,000 VND
  And participant B ordered 100,000 VND
When shipping fee is 30,000 VND
Then participant A pays 10,000 VND shipping
  And participant B pays 20,000 VND shipping
```

---

## 5. Mocking Strategy

### What to Mock
- ✅ External APIs (payment gateways, third-party services)
- ✅ Database layer (for unit tests)
- ✅ Time/Clock (for time-dependent logic)
- ✅ File system (for file operations)

### What NOT to Mock
- ❌ The function under test itself
- ❌ Simple data structures
- ❌ Standard library functions
- ❌ Database in integration tests (use test DB)

### Go — Interface Mocking
```go
// Define interface in domain
type SessionRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*Session, error)
}

// Mock in tests
type MockSessionRepo struct {
    FindByIDFunc func(ctx context.Context, id uuid.UUID) (*Session, error)
}
```

---

## 6. Coverage Targets

| Category | Minimum | Notes |
|----------|:-------:|-------|
| Business logic (calculations, rules) | 90% | Critical — money involved |
| Service layer | 80% | Core application logic |
| Repository layer | 70% | Integration tests cover gaps |
| Handler layer | 60% | Thin layer, integration test preferred |
| Utility functions | 90% | Widely reused, high impact |
| UI Components | — | Visual testing, storybook |

### Coverage Rules
- Coverage must **not decrease** on any PR
- New code must maintain or improve overall coverage
- Coverage reports generated on every CI run
- **Don't chase 100%** — diminishing returns after 90%

---

## 7. Test Data

### Principles
- **Self-contained**: Each test creates its own data, no shared state
- **Deterministic**: Same input → same output, every time
- **Minimal**: Only create what the test needs
- **Cleanup**: Tests clean up after themselves (or use transactions that rollback)

### Test Database
- Separate test database instance
- Migrations run before test suite
- Each test wrapped in transaction (rollback after test)
- **Never** test against production data

---

## 8. CI Test Execution

### Performance Targets
| Metric | Target |
|--------|--------|
| Full test suite | < 15 minutes |
| Unit tests only | < 3 minutes |
| Flaky test rate | < 2% |

### Flaky Test Policy
- Flaky test detected → immediately quarantine
- Fix within **2 business days** or delete
- Never `@skip` or `t.Skip()` without a ticket and deadline
- Track flaky rate weekly

---

## 9. What to Test vs What Not to

### Always Test
- ✅ Business calculations
- ✅ State transitions
- ✅ Input validation
- ✅ Error handling paths
- ✅ Edge cases and boundary values
- ✅ Authorization logic

### Don't Over-Test
- ❌ Framework internals (React rendering, Go HTTP routing)
- ❌ Third-party library behavior
- ❌ One-to-one code-to-test mapping (test behavior, not implementation)
- ❌ Trivial getters/setters

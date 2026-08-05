---
name: QC Skill
description: >
  Quality audit methodology, checklist, reject/approve workflow, and sign-off criteria.
  Use this skill whenever QC audits a completed PR or task — including code review,
  security audit, spec compliance check, test adequacy review, and final sign-off.
  Trigger on phrases like "QC review", "audit this PR", "check compliance", "approve task",
  "sign off", or when the DEV column is ✅ and TEST column is ✅ in the Registry.
---

# 🛡️ QC Skill — Quality Audit & Sign-off

---

## Quick Reference

| Section | When to use |
|---------|-------------|
| [1. Audit Methodology](#1-audit-methodology) | Starting any QC audit |
| [2. Quality Gate Checklist](#2-quality-gate-checklist) | Systematic audit execution |
| [3. Security Audit](#3-security-audit) | Every PR with new endpoints or auth changes |
| [4. Performance Review](#4-performance-review) | Pre-release or heavy DB/API changes |
| [5. Spec & Architecture Compliance](#5-spec--architecture-compliance) | Cross-reference BA spec + Architect TDD |
| [6. Reject Workflow](#6-reject-workflow) | When violations found |
| [7. Approve & Sign-off](#7-approve--sign-off) | When all gates pass |

---

## 1. Audit Methodology

### Audit Layers (in order)
```
1. Spec Compliance   → Code matches BA spec + Architect TDD exactly
2. Code Quality      → Clean code, architecture layer compliance
3. Testing Audit     → TDD compliance, coverage adequacy, test quality
4. Security Audit    → IDOR, injection, data exposure, STRIDE threats
5. Performance       → DB queries, N+1, bundle size, SLO targets
6. Cross-Reference   → Code = Spec = Test = Registry consistent
```

### Audit Against Rules
Mỗi audit phải đối chiếu với:
- `rules/code-style.md` → Code quality & naming
- `rules/testing.md` → Test quality & TDD compliance
- `rules/security.md` → Security compliance
- `rules/error-handling.md` → Error handling patterns
- `rules/database.md` → DB schema & queries
- `rules/api-convention.md` → API design & response shape
- `rules/git-workflow.md` → PR & commit standards
- `docs/tdd/` → Architect TDD (if exists for this feature)
- `docs/adr/` → Relevant ADRs

### Audit Triggers
- DEV column in Registry → ✅
- TEST column in Registry → ✅
- All `[MUST]` items from Dev's self-review checklist → resolved

---

## 2. Quality Gate Checklist

### A. Code Quality
- [ ] Clean Architecture layers respected — no cross-layer imports
- [ ] Functions ≤ 50 lines, files ≤ 300 lines
- [ ] No magic numbers or hardcoded values — all constants extracted
- [ ] Meaningful variable/function names per naming conventions (SKILL_DEV §3, §4)
- [ ] No `fmt.Println` / `console.log` / dead code in production
- [ ] Guard clauses used (no deep nesting > 3 levels)
- [ ] Error wrapping with context at every layer (Go: `fmt.Errorf("context: %w", err)`)
- [ ] TypeScript: no `any` type unless justified with comment
- [ ] TypeScript: all async functions have explicit return type annotation
- [ ] `go fmt` / `npm run lint` / `tsc --noEmit` all clean

### B. Testing Compliance
- [ ] Test files exist for all business logic (`*_test.go` / `*.test.ts`)
- [ ] Coverage ≥ 80% for service/business logic layer
- [ ] Coverage ≥ 90% for money/calculation/formula functions
- [ ] Coverage ≥ 60% for HTTP handlers (integration tests count)
- [ ] Tests cover positive, negative, and boundary cases (≥ 3 per function)
- [ ] No trivial/fake tests that always pass regardless of implementation
- [ ] TDD unit tests: describe → arrange → act → assert pattern followed
- [ ] New API routes have ≥1 integration test

### C. Database
- [ ] Primary keys use appropriate ID format (UUID v7 recommended, or as per `rules/database.md`)
- [ ] Migrations have both UP and DOWN scripts
- [ ] No `SELECT *` in queries — explicit column lists
- [ ] Foreign keys have indexes
- [ ] Parameterized queries only — no string concatenation (SQL injection risk)
- [ ] Complex queries have EXPLAIN plan reviewed
- [ ] Migrations idempotent or safe to re-run

### D. API & Error Handling
- [ ] Response envelope follows standard `{ data, error }` shape
- [ ] Error responses include `error_code` and `message` (and `trace_id` if applicable)
- [ ] Correct HTTP status codes used (ref: SKILL_ARCHITECT §4)
- [ ] Validation errors return field-level details
- [ ] All list endpoints are paginated `{ data: [...], meta: { total, page, per_page } }`
- [ ] API versioned (`/v1/` prefix) for public-facing endpoints

### E. Registry & PR Standards
- [ ] Registry DEV column → ✅ with PR link
- [ ] Registry TEST column → ✅ with test evidence link
- [ ] Branch named correctly: `feature/{ID}-{kebab-desc}` / `fix/...` / `chore/...`
- [ ] PR description includes: **What**, **Why**, **How to test**
- [ ] No unrelated changes bundled into this PR

---

## 3. Security Audit

Audit mỗi PR có endpoint mới, auth thay đổi, hoặc xử lý dữ liệu nhạy cảm.

### STRIDE Security Checklist (per feature)
- [ ] **Spoofing** — Identity header validated on every mutating endpoint
- [ ] **Tampering** — IDOR prevention: ownership verified before mutation (not just route-level auth)
- [ ] **Repudiation** — Sensitive operations logged in audit trail (payment, permission changes)
- [ ] **Info Disclosure** — No secrets/passwords/PII in API responses or logs
- [ ] **DoS** — Rate limiting configured for public & auth endpoints
- [ ] **Elevation** — Authorization checked at resource level (RBAC enforced)

### Additional Security Checks
- [ ] Input validated and sanitized (SQL injection, XSS, path traversal)
- [ ] CORS whitelist configured — no wildcard `*` in production
- [ ] HTTPS enforced; no mixed content
- [ ] Secrets in environment variables, not source code
- [ ] Dependencies checked for known CVEs (`npm audit` / `go mod verify`)

---

## 4. Performance Review

### Backend Performance
- [ ] No N+1 query patterns — use JOINs or batch loads
- [ ] Complex queries have proper indexes (check with EXPLAIN ANALYZE)
- [ ] `context.Context` passed to all DB/network calls
- [ ] Connection pooling configured
- [ ] No goroutine leaks in WebSocket/long-polling handlers
- [ ] p50 API response < 100ms, p99 < 500ms (ref: SKILL_ARCHITECT §8 SLOs)

### Frontend Performance
- [ ] Server Components used where possible (reduce bundle size)
- [ ] Images optimized via Next.js `<Image>` component
- [ ] No unnecessary re-renders (useEffect/useMemo dependency arrays correct)
- [ ] Lazy loading for heavy components (`dynamic(() => import(...))`)
- [ ] LCP < 2.5s on mobile (ref: SKILL_ARCHITECT §8 SLOs)
- [ ] Bundle size not increased beyond 10% without justification

---

## 5. Spec & Architecture Compliance

### BA Spec Compliance
- [ ] Implementation matches project spec contract exactly (request/response shape, field names — ref: `docs/api_spec.md` or equivalent)
- [ ] Business logic matches documented formulas (billing, calculation, state machine)
- [ ] UI flow matches design specifications and AC (Acceptance Criteria)
- [ ] All edge cases from spec are handled in code

### Architect TDD Compliance
- [ ] If TDD exists in `docs/tdd/`: implementation follows approved design
- [ ] Layer separation matches TDD system architecture diagram
- [ ] API contracts match TDD §2.2
- [ ] DB schema changes match TDD §2.3
- [ ] Any deviation from TDD requires new ADR or TDD revision — not silent override

### Cross-Reference Final Check
```
Code logic  ↔  BA Spec AC        (100% match)
API shape   ↔  Project Spec doc  (100% match)
Test cases  ↔  Spec scenarios    (all AC covered)
Registry    ↔  Actual PR/branch  (IDs consistent)
```

---

## 6. Reject Workflow

When QC finds violations:

```
1. Document: Which rule is violated + exact file:line + evidence
2. Classify: [MUST] blocking vs [SUGGEST] improvement vs [QUESTION] clarification needed
3. Action:
   - [MUST] → Set DEV column back to 🏗️, QC column → ❌
   - [MUST] → Notify PM to re-open task
   - [SUGGEST] → Comment on PR, non-blocking for merge
   - [QUESTION] → Tag Dev or BA for clarification before proceeding
4. Report using Reject Message Format below
```

### Reject Message Format
```markdown
## ❌ QC REJECT — {TASK-ID}

### Violations
1. **[MUST] rules/security.md §2.1**: IDOR — `DeleteOrder` doesn't verify device ownership
   - File: `internal/handler/order_handler.go:45`
   - Fix required: Add `deviceID` ownership check before delete

2. **[MUST] rules/testing.md §3**: Coverage 62% on `SessionService` (required: ≥80%)
   - Missing: negative cases for `LockSession` and boundary test for empty participant list

3. **[SUGGEST] rules/code-style.md §5**: Function `ProcessOrder` is 67 lines (limit: 50)
   - File: `internal/service/order_service.go:120`
   - Suggestion: Extract validation block into `validateOrderItems()`

### Status Update
- DEV: 🏗️ (was ✅) — re-open for fixes
- TEST: ⬜ (pending DEV fix)
- QC: ❌
- Registry: Updated
```

---

## 7. Approve & Sign-off

### Prerequisites
```
DEV ✅  +  TEST ✅  +  All [MUST] violations resolved  =  QC ✅
```

No exceptions. QC ✅ is the final gate before Registry status → `DONE`.

### Sign-off Message Format
```markdown
## ✅ QC APPROVED — {TASK-ID}

### Audit Summary
- Spec Match:    ✅ 100% aligned with BA spec + TDD
- Code Quality:  ✅ Clean, follows architecture layers, meets style rules
- Testing:       ✅ Coverage 87% (service), 94% (billing logic), all scenarios adequate
- Security:      ✅ STRIDE checked, no IDOR or injection risks found
- Performance:   ✅ No N+1, indexes verified, p99 estimated < 300ms
- Cross-Ref:     ✅ Code = Spec = Test = Registry consistent

### Registry Update
- QC: ✅
- Status: → DONE
```

### When to Escalate (не approve, не reject)
- Spec is ambiguous → escalate to BA before auditing
- TDD has conflicting design from implementation → escalate to Architect
- Security risk unclear → raise with Architect for threat assessment

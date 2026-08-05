---
name: Tester Skill
description: >
  Test execution methodology, test case structure, bug reporting, and evidence policy.
  Use this skill whenever the user is: writing or designing test cases, executing tests
  against a feature, reporting bugs, verifying bug fixes, or signing off the TEST column
  in the Registry. Trigger on phrases like "write test cases for", "test this API",
  "verify this feature", "report a bug", "retest", or when DEV column is ✅ in Registry.
---

# 🧪 Tester Skill — Testing Methodology & Reporting

---

## Quick Reference

| Section | When to use |
|---------|-------------|
| [1. Test Case Structure](#1-test-case-structure) | Creating any test case (TEST-xxxxx) |
| [2. Testing Scope & Strategy](#2-testing-scope--strategy) | Planning tests for a task |
| [3. TDD Alignment](#3-tdd-alignment) | Verifying Dev's unit tests are adequate |
| [4. API & Integration Testing](#4-api--integration-testing) | Testing every endpoint |
| [5. E2E Testing Methodology](#5-e2e-testing-methodology) | Core user flow testing |
| [6. Performance Testing](#6-performance-testing) | Pre-release or scaling validation |
| [7. Bug Reporting](#7-bug-reporting) | When a test fails |
| [8. Evidence & Reporting Policy](#8-evidence--reporting-policy) | After test execution |
| [9. Verify (Retest) Workflow](#9-verify-retest-workflow) | After Dev marks bug Fixed |
| [10. Definition of Done](#10-definition-of-done-test-sign-off) | Before marking TEST ✅ |

---

## 1. Test Case Structure

### Format bắt buộc cho mỗi `TEST-xxxxx`

| Field | Description | Example |
|-------|-------------|---------|
| **ID** | `TEST-xxxxx` | `TEST-00001` |
| **Target** | Task/API/Feature being tested | `FEAT-00005 — Lock Session API` |
| **Type** | API / UI / E2E / WebSocket / Performance | `API` |
| **Objective** | What is being validated | Verify session lock blocks further orders |
| **Precondition** | System state required before test | Session OPEN, ≥2 participants, ≥1 order |
| **Steps** | Numbered, reproducible steps | 1. Call `POST /sessions/{id}/lock`... |
| **Expected** | From spec/AC — what must happen | `200 OK`, session status → `LOCKED` |
| **Actual** | Filled after execution | (Updated after test run) |
| **Status** | ✅ Pass / ❌ Fail / ⏭️ Skip | ✅ |
| **Evidence** | Link or inline JSON/screenshot | `tests/REPORTS.md#TEST-00001` |

---

## 2. Testing Scope & Strategy

### Risk-Based Test Priority
```
Priority 1 (Always test):
  - Money/billing/calculation logic
  - State machine transitions (session status, payment lock)
  - Auth/authorization flows
  - Data mutation endpoints (POST, PATCH, DELETE)

Priority 2 (Test on every change):
  - Core user flows (create session, join, order, checkout)
  - Validation rules
  - Error handling and edge cases

Priority 3 (Test before release):
  - Performance under load
  - E2E cross-browser
  - WebSocket reconnection behavior
```

### Coverage Requirements (aligned with SKILL_DEV TDD)
Every task must have **≥3 scenarios minimum**:
1. ✅ **Positive** — happy path, all inputs valid
2. ❌ **Negative** — invalid input, unauthorized access, wrong state
3. 🔲 **Boundary** — min/max values, empty collections, null inputs, exact limits

For money/formula functions: ≥5 scenarios including zero-value, overflow, rounding cases.

### Test Types by Priority
| Type | When Required | Tool |
|------|--------------|------|
| **Unit Test** | All business logic (via TDD by Dev) | Vitest / Jest / Go test |
| **API Testing** | Every endpoint — all HTTP methods | cURL / Postman / httpie |
| **Integration** | New API routes, DB interactions | Supertest / Go httptest |
| **UI Testing** | Every UI change visible to user | Browser (Chrome DevTools) |
| **E2E Testing** | Core user flows before release | Playwright / Cypress |
| **WebSocket** | Realtime features | Custom scripts / wscat |
| **Performance** | Before major release or load-critical feature | k6 / Artillery |

---

## 3. TDD Alignment

Tester verifies Dev's unit tests (from TDD workflow) are adequate — not just present.

### Unit Test Adequacy Checklist
- [ ] Tests exist for all functions with business logic
- [ ] Format: `describe → arrange → act → assert` (TypeScript) / subtests (Go)
- [ ] Positive case implemented and passes
- [ ] Negative cases: invalid input, null/undefined, wrong type
- [ ] Boundary cases: zero, max, exact limits
- [ ] No test mocked so heavily it tests nothing real
- [ ] Tests fail if the implementation is removed (non-trivial)

### When TDD Was Mandatory (from SKILL_DEV §2)
If any of these exist in the task, verify unit tests are present:
- Calculation/formula functions
- State machine transitions (session status, payment lock)
- Validation logic
- Data transformation functions
- Custom hooks with non-trivial logic

---

## 4. API & Integration Testing

### API Test Checklist (per endpoint)
- [ ] **401** — unauthenticated request rejected
- [ ] **403** — authenticated but wrong ownership (IDOR test)
- [ ] **400/422** — missing required fields, invalid types
- [ ] **200/201/204** — success path with correct response shape
- [ ] **409** — conflict state (e.g., session already locked)
- [ ] Response shape matches `{ data: T, error: null }` / `{ data: null, error: {...} }`
- [ ] Paginated list endpoints: verify `meta.total`, `page`, `per_page` present

### API Test Execution Format
```bash
# Document every test call with:
# [TEST-xxxxx] Scenario description
curl -X POST https://staging.api/v1/resources/{id}/action \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json"

# Expected: 200 OK + { data: { status: "UPDATED" }, error: null }
# Actual: [paste response]
```

### Data Consistency Verification
After every successful mutation:
```
UI display  ↔  API response  ↔  Database row
(100% match at all 3 layers)
```

---

## 5. E2E Testing Methodology

### Core User Journey (always test before release)
```
[Define core user flow of your project here]
Example: Open App → Auth → Create Resource → Update → Verify → Complete
```

### E2E Scenario Design
Mock real user behavior — don't shortcut the UI:
1. Start from app open (no pre-seeded state unless explicitly needed)
2. Use clean test data per run (setup/teardown or unique identifiers)
3. Verify system state at each step (not just at end)

### Environment Rules
- Execute on **Staging/Preview** environment only — never production
- Clean data per test run (setup/teardown scripts or unique session IDs)
- Test on at minimum: **Mobile Chrome** + **Desktop Chrome**
- Document browser + environment version in report

### E2E Playwright Pattern
```ts
test('complete session flow — happy path', async ({ page }) => {
  // Arrange
  await page.goto('/sessions/new')

  // Act — simulate user steps
  await page.getByRole('button', { name: 'Create Session' }).click()
  await page.getByLabel('Table Number').fill('A1')
  await page.getByRole('button', { name: 'Confirm' }).click()

  // Assert
  await expect(page.getByText('Session Active')).toBeVisible()
  await expect(page.getByTestId('session-status')).toHaveText('OPEN')
})
```

---

## 6. Performance Testing

### When Required
- Feature handles >100 concurrent users
- New heavy DB query added
- Before major public release

### SLO Targets (from SKILL_ARCHITECT §8)
| Metric | Target | Fail threshold |
|--------|--------|----------------|
| API response (p50) | < 100ms | > 200ms |
| API response (p99) | < 500ms | > 1000ms |
| Page LCP (mobile) | < 2.5s | > 4s |

### k6 Smoke Test Pattern
```js
import http from 'k6/http'
import { check } from 'k6'

export const options = {
  vus: 10,         // 10 concurrent users
  duration: '30s',
}

export default function () {
  const res = http.get('https://staging.api/v1/sessions')
  check(res, {
    'status 200': (r) => r.status === 200,
    'response < 500ms': (r) => r.timings.duration < 500,
  })
}
```

---

## 7. Bug Reporting

### Registry Bug Tracker (one-line entry)
```markdown
| Bug ID | Related | Description | Severity | Status |
|--------|---------|-------------|----------|--------|
| BUG-00012 | FEAT-00005 | Negative qty accepted in order | High | Open |
```

### Bug Report Details
```markdown
## BUG-xxxxx: [Short description]
- **Severity**: Critical / High / Medium / Low
- **Related Task**: FEAT-xxxxx / API-xxxxx
- **Environment**: Staging / Preview / Production
- **Browser/Platform**: Chrome 122 / iOS 17 / etc.

### Steps to Reproduce
1. [Step 1 — be specific, include exact values]
2. [Step 2]
3. [Step 3]

### Expected
[Reference the spec or AC — "According to FEAT-00005 AC-2, the system should..."]

### Actual
[What actually happened — include exact error message or behavior]

### Evidence
[Screenshot / Video / Response JSON — mandatory for Critical & High]
```

### Severity Classification
| Level | When | SLA |
|-------|------|-----|
| **Critical** | Data loss, security breach, system down | Fix same day |
| **High** | Core feature broken, wrong calculation, blocking other tasks | Fix within sprint |
| **Medium** | Non-core feature issue, UX degraded but workaround exists | Next sprint |
| **Low** | Cosmetic, minor UX, typo | Backlog |

---

## 8. Evidence & Reporting Policy

| Scenario | What to Report |
|----------|---------------|
| **All Pass** | ✅ Summary: "TEST-00001 to 00005 all passed — 15 scenarios" |
| **Critical Path** | Save 1 JSON response as evidence in `tests/REPORTS.md` per task |
| **Failure** | Full detail: Expected vs Actual + error code + screenshot + repro steps |
| **Retest** | Reference `BUG-xxxxx` + confirm fix + note if regression check passed |

### Report Structure in `tests/REPORTS.md`
```markdown
## FEAT-00005 — Lock Session API (TEST-00003)
**Executed**: 2025-03-27 | **By**: Tester | **Build**: preview-abc123

| # | Scenario | Status | Notes |
|---|----------|--------|-------|
| 1 | Happy path — valid lock request | ✅ | Response 87ms |
| 2 | Unauthorized — wrong device ID | ✅ | 403 returned |
| 3 | Conflict — already locked | ✅ | 409 returned |
| 4 | Boundary — 0 participants | ✅ | 400 returned |

**Evidence**: [response JSON](./evidence/FEAT-00005-lock.json)
```

### Log Retention
- Keep only **latest** test results per task (overwrite previous)
- Archive evidence for Critical/High bugs for **30 days**
- Delete old runs to prevent `tests/` file bloat

---

## 9. Verify (Retest) Workflow

When Dev marks bug as "Fixed":
1. Pull latest code / check preview deploy (confirm build hash)
2. Re-execute **exact same steps** from bug report
3. Verify fix doesn't introduce regression in related areas
4. If fixed: ✅ Update bug status → `Verified`, add note with build hash
5. If not fixed: ❌ Reopen with additional details and new evidence

### Regression Scope
After a bug fix, also re-test:
- Any feature that shares the same service or component
- Any test cases previously marked `⏭️ Skip` related to this area

---

## 10. Definition of Done (Test Sign-off)

Tester marks TEST ✅ only when **all** of the following are met:

- [ ] All positive scenarios pass ✅
- [ ] ≥3 negative/boundary scenarios tested and pass ✅
- [ ] Unit test adequacy verified (SKILL_TESTER §3 checklist) ✅
- [ ] All related bugs are `Verified` (status `Fixed`) ✅
- [ ] UI data matches API response (100% — verified at all 3 layers) ✅
- [ ] Tested on mobile viewport (≥375px) ✅
- [ ] Evidence saved in `tests/REPORTS.md` ✅
- [ ] Registry TEST column updated → ✅ with report link ✅

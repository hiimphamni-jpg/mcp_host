---
name: PM Skill
description: Sprint management methodology, registry standards, and task lifecycle. Use this skill whenever the user mentions sprints, backlogs, task IDs, story points, sprint planning, grooming, DoR, DoD, bug triage, velocity, or any project management workflow. Trigger even for partial mentions like "add a task", "update the registry", "what's the status of", or "plan the next sprint" — don't wait for explicit PM terminology.
---
 
# 📑 PM Skill — Sprint Management & Registry
 
---
 
## 1. Registry Management (`docs/REGISTRY.md`)
 
### Task ID Format
 
| Type | Format | Example |
|------|--------|---------|
| Feature | `FEAT-xxxxx` | `FEAT-00005` |
| API endpoint | `API-xxxxx` | `API-00012` |
| Bug fix | `BUG-xxxxx` | `BUG-00003` |
| Refactor | `REFAC-xxxxx` | `REFAC-00001` |
| DevOps | `OPS-xxxxx` | `OPS-00002` |
| UI/UX | `UX-xxxxx` | `UX-00004` |
| Documentation | `DOC-xxxxx` | `DOC-00001` |
| Testing/QA | `QA-xxxxx` | `QA-00007` |
 
All IDs use 5-digit zero-padded numbers. IDs are unique across the entire registry (no two tasks share an ID regardless of type).
 
### Registry Table Format
 
```markdown
| ID | Description | Priority | Sprint | DEV | TEST | QC | Status |
|----|-------------|----------|--------|-----|------|-----|--------|
| FEAT-00005 | Calculate shipping | P0 | S3 | ✅ | ✅ | ✅ | DONE |
| API-00012 | Settlement endpoint | P1 | S3 | 🏗️ | ⬜ | ⬜ | IN_PROGRESS |
| UX-00004 | Redesign checkout flow | P1 | S4 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| DOC-00001 | API reference guide | P2 | S4 | ⬜ | ⬜ | ⬜ | BACKLOG |
| QA-00007 | Regression suite setup | P1 | S3 | ⬜ | 🏗️ | ⬜ | IN_TESTING |
```
 
### Status Flow
 
```
BACKLOG → SPRINT BACKLOG → IN_PROGRESS → IN_TESTING → IN_REVIEW → DONE
                                ↑                          ↓
                                └────── RE-OPEN ←──────────┘
```
 
---
 
## 2. Sprint Lifecycle
 
### Sprint Duration
- Default: 2 weeks (10 working days)
 
### Ceremony Schedule
 
| Day | Ceremony | Duration |
|-----|----------|----------|
| Day 1 | Sprint Planning | 2–4 hrs |
| Day 1–10 | Daily Standup | 15 min/day |
| Day 6 | Backlog Grooming (prep next sprint) | 1–2 hrs |
| Day 10 | Sprint Review + Retrospective | 1–2 hrs |
 
### Sprint Planning Process
1. PM selects tasks from Backlog based on priority (P0 > P1 > P2)
2. Dev confirms capacity (subtract PTO, meetings)
3. Total story points ≤ 80% of team capacity
4. Sprint Goal must be measurable
5. All tasks must pass DoR before entry
 
### Backlog Grooming (Day 6)
- Held on Day 6 to allow more time to clarify requirements before next Sprint Planning
- Tasks > 8 points must be broken down into smaller tasks
- Clarify Acceptance Criteria with BA if missing
- Estimate using story points (Fibonacci: 1, 2, 3, 5, 8, 13)
 
---
 
## 3. Definition of Ready (DoR)
 
A task may only enter a Sprint when **all** of the following are met:
 
- [ ] Has an ID in the Registry
- [ ] Linked to a REQ/FEAT reference
- [ ] BA has completed Spec + Acceptance Criteria
- [ ] Priority confirmed (P0/P1/P2)
- [ ] Estimate ≤ 8 story points
- [ ] No unresolved dependencies
- [ ] **Design approval obtained** (required for UX tasks and any FEAT/API task with a UI component)
 
---
 
## 4. Definition of Done (DoD)
 
A task is only **DONE** when all three columns are ✅:
 
1. **DEV ✅**
   - Code passes lint and formatting checks
   - Unit tests pass 100%
   - **Pull Request reviewed and approved** by at least one other developer
   - PR merged to main/develop branch
 
2. **TEST ✅**
   - Test cases executed
   - ≥ 3 scenarios pass
   - No open bugs
 
3. **QC ✅**
   - Code audit passed
   - Security check OK
   - **Deployment checklist completed:**
     - [ ] Environment variables confirmed
     - [ ] DB migrations reviewed
     - [ ] Rollback plan documented
     - [ ] Staging deployment verified
 
→ PM updates Status to **DONE** only after all three columns are ✅
 
---
 
## 5. Priority Classification
 
| Level | Label | SLA | When |
|:-----:|-------|-----|------|
| P0 | Critical | Within current sprint | Core feature, data integrity, security |
| P1 | High | Next 1–2 sprints | Important feature, major UX issue |
| P2 | Medium | Backlog (prioritized) | Nice-to-have, optimization |
| P3 | Low | Backlog (unprioritized) | Cosmetic, minor improvement |
 
---
 
## 6. Bug Triage in Sprint
 
### Bug Tracker Format
 
```markdown
| Bug ID | Related Task | Description | Severity | Status |
|--------|-------------|-------------|----------|--------|
| BUG-00001 | API-00005 | Negative qty accepted | High | Open |
```
 
### Triage Rules
- **P0 bug** found in Sprint → Fix immediately, same Sprint
- **P0 bug** too large → Remove related task from Sprint, move to Backlog
- **P1/P2 bugs** → Add to next Sprint Backlog
- Bug fixer reports back to Tester for verification
 
### Bug ID Format
- 5-digit zero-padded: `BUG-00001`, `BUG-00002`, ...
 
---
 
## 7. Velocity & Reporting
 
### Metrics to Track
- **Velocity**: Story points completed per Sprint
- **Commitment accuracy**: Points committed vs. completed (target ≥ 80%)
- **Bug escape rate**: Bugs found in production vs. total bugs (target < 5%)
- **Sprint burndown**: Daily progress tracking
 
### Capacity Planning
- Use average velocity from last 3 sprints
- Deduct 20% for tech debt, bugs, and unexpected issues
- Never commit more than actual capacity
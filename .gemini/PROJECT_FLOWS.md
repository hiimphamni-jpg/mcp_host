# Project Task Flows

Use these flows to coordinate the existing roles. A phase may be skipped only when its stated condition applies. Each role must read its own agent, skill, command, and relevant rule files before working.

## Flow 1: New Feature

Use for a new user-facing capability, API, data model, or non-trivial behavior.

| Phase | Command | Exit condition |
|---|---|---|
| Clarify | `/think {idea}` | Goal, constraints, risks, recommendation, and acceptance criteria are recorded. |
| Design | `/architect design {feature}` | Technical design exists; ADR is added when the decision has durable impact. |
| Specify | `/ba spec` and `/ba ac {ID}` | Business rules, API contract, and acceptance criteria are unambiguous. |
| Register | `/pm registry` then `/pm dor {ID}` | A five-digit task ID exists and DoR is passed. |
| Plan | `/plan {ID}` | Detailed plan is saved and the user replies `APPROVED`. |
| Implement | `/build` or `/dev code {ID}` | TDD or an explicit verification alternative passes. |
| Test | `/test cases`, then `/test api`, `/test ui`, or `/test e2e` as applicable | Test results cover the acceptance criteria. |
| Audit | `/qc audit {ID}` then `/qc sign-off {ID}` | No Blocker remains and QC signs off. |
| Release | `/devops deploy staging`, then production when approved | Deployment and monitoring checks pass. |
| Close | `/finish {ID}` | Artifacts are complete and Registry status is `DONE`. |

## Flow 2: Bug Fix

Use for a defect in existing behavior. The regression test is mandatory whenever technically feasible.

| Phase | Command | Exit condition |
|---|---|---|
| Diagnose | `/fix {symptom}` | Reproduction, root cause hypothesis, and minimal fix scope are written. |
| Triage | `/pm triage` | A `BUG-xxxxx` ID, priority, owner, and target release are defined. |
| Clarify | `/ba logic` only if expected behavior is unclear | Expected behavior and acceptance criteria are agreed. |
| Plan | `/plan BUG-xxxxx` | User replies `APPROVED`. |
| Fix | `/dev test BUG-xxxxx`, then `/dev code BUG-xxxxx` | Regression test proves the defect, then passes with the smallest fix. |
| Verify | `/test verify` | Original reproduction and affected test suite pass. |
| Audit | `/qc review BUG-xxxxx` | No Blocker or unaddressed Major remains. |
| Release and close | `/devops deploy {env}` when needed, then `/finish BUG-xxxxx` | Fix is released if required and Registry is updated. |

## Flow 3: Change Request

Use when an approved specification, design, or in-progress task changes scope.

| Phase | Command | Exit condition |
|---|---|---|
| Impact analysis | `/ba cr {request}` | Impact on requirements, API, data, security, and acceptance criteria is documented. |
| Technical assessment | `/architect review {change}` | Design impact, migration strategy, and compatibility risks are resolved. |
| Reprioritize | `/pm plan` and `/pm registry` | Scope, task IDs, dependencies, and DoR are updated. |
| Re-plan | `/plan {task IDs}` | User replies `APPROVED` to the revised plan. |
| Deliver | `/build`, `/test verify`, `/qc audit {ID}` | Implementation, verification, and QC follow the New Feature flow. |
| Close | `/finish {ID}` | Registry and workflow artifacts capture the final outcome. |

## Command Shortcuts

- `/feature {idea}` starts Flow 1 and identifies the next required command.
- `/bugfix {symptom}` starts Flow 2 and preserves the debug report.
- `/change {request}` starts Flow 3 and prevents implementation before impact analysis and approval.

Do not bypass the `/plan` approval gate. Use `/fix` immediately when a required verification fails.

# OpenCode Project Commands

This directory adapts the existing `.gemini` role system for OpenCode. Do not edit `.gemini` source instructions to add an OpenCode command; add a Markdown command file in `commands/` instead.

## Task Flows

| Command | Use |
|---|---|
| `/feature {idea}` | New-feature lifecycle from discovery through QC and release. |
| `/bugfix {symptom}` | Reproduce, triage, regression-test, fix, verify, and close a defect. |
| `/change {request}` | Assess and deliver a change to approved or in-progress work. |

The full checkpoint definitions are in `.gemini/PROJECT_FLOWS.md`. The `/plan` approval gate remains mandatory before implementation.

## Daily Workflow

| Command | Use |
|---|---|
| `/think {idea}` | Brainstorm requirements and risks without writing code. |
| `/plan {task or request}` | Write a detailed implementation plan and request `APPROVED`. |
| `/build` | Execute the approved plan step by step with verification. |
| `/fix {symptom}` | Debug systematically and record a root-cause report. |
| `/review` | Review changes with Blocker, Major, Minor, and Nit severities. |
| `/finish {ID}` | Record completion artifacts and update sign-off status. |

## Utility Commands

| Command | Use |
|---|---|
| `/status {ID}` | Show the task state, completed gates, blockers, and next action. |
| `/handoff {ID}` | Prepare a concise, evidence-based handoff to the next role. |
| `/verify {scope}` | Select and run appropriate tests, linting, and build checks without editing files. |
| `/release-check {ID}` | Assess whether a task is ready for a staging or production deployment. |
| `/docs {topic}` | Update project documentation after an approved change. |

## Role Commands

| Role | Command |
|---|---|
| Business Analyst | `/ba spec`, `/ba ac {ID}`, `/ba logic` |
| Architect | `/architect design {feature}`, `/architect adr` |
| Project Manager | `/pm registry`, `/pm dor {ID}`, `/pm triage` |
| Developer | `/dev code {ID}`, `/dev test {ID}`, `/dev pr {ID}` |
| Tester | `/test cases`, `/test api`, `/test e2e`, `/test verify` |
| Quality Control | `/qc review {ID}`, `/qc audit {ID}`, `/qc sign-off {ID}` |
| DevOps | `/devops deploy {env}`, `/devops rollback`, `/devops pipeline` |

## Examples

```text
/feature Add an endpoint to revoke an API token
/bugfix Login returns HTTP 500 when email is absent
/status FEAT-00005
/handoff FEAT-00005 to qc
/verify ./...
/release-check FEAT-00005 staging
/docs Document API token revocation endpoint
```

Restart OpenCode after changes under `.opencode/` so it reloads commands and agents.

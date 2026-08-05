# 🎮 Command Reference

Quick reference for all available slash commands. Dùng **Workflow Commands** trước, gọi **Agent Commands** khi cần chuyên sâu.

---

## ⚡ Workflow Commands (Hàng ngày — ngắn gọn)

| Command | Mục đích | File |
|---------|-----------|------|
| `/think [mô tả]` | Brainstorm, phân tích yêu cầu | `commands/think.md` |
| `/plan [yêu cầu]` | Lập kế hoạch → gate APPROVED | `commands/plan.md` |
| `/build` | Thực thi plan từng step có verify | `commands/build.md` |
| `/fix [lỗi]` | Debug có hệ thống | `commands/fix.md` |
| `/review` | Review code (Blocker/Major/Minor/Nit) | `commands/review.md` |
| `/finish` | Tổng kết + QC sign-off | `commands/finish.md` |

### Luồng cơ bản:
```
/think → /plan → APPROVED → /build → /review → /finish
```

---

## 👥 Agent Commands (Chuyên sâu)

| # | Command | Role | Skill File |
|---|---------|------|-----------|
| 1 | `/ba` | Business Analyst | `skills/SKILL_BA.md` |
| 2 | `/pm` | Project Manager | `skills/SKILL_PM.md` |
| 3 | `/dev` | Senior Developer | `skills/SKILL_DEV.md` |
| 4 | `/test` | Tester | `skills/SKILL_TESTER.md` |
| 5 | `/qc` | Quality Control | `skills/SKILL_QC.md` |
| 6 | `/devops` | DevOps / SRE | `skills/SKILL_DEVOPS.md` |
| 7 | `/architect` | Tech Lead / Architect | `skills/SKILL_ARCHITECT.md` |

---

## All Commands

### `/ba` — Business Analyst
| Command | Purpose |
|---------|---------|
| `/ba spec` | Tạo/cập nhật API specification |
| `/ba logic` | Thiết kế business logic & formulas (INPUT/RULE/OUTPUT) |
| `/ba ac {ID}` | Viết Acceptance Criteria (Gherkin format) |
| `/ba workflow` | Vẽ user flow diagram (Mermaid) |
| `/ba ticket` | Viết sprint ticket (Story/Task/Bug/Spike) |
| `/ba cr` | Change Request + full impact analysis |
| `/ba audit` | Gap analysis & edge case review |

### `/pm` — Project Manager
| Command | Purpose |
|---------|---------|
| `/pm plan` | Sprint planning & task selection |
| `/pm registry` | Quản lý Registry — tạo/cập nhật Task ID (5-digit) |
| `/pm dor {ID}` | Check Definition of Ready trước khi vào Sprint |
| `/pm sprint` | Sprint lifecycle (start / close / retro) |
| `/pm status` | Report sprint progress & velocity |
| `/pm triage` | Bug triage & priority assignment |

### `/dev` — Senior Developer
| Command | Purpose |
|---------|---------|
| `/dev code {ID}` | Implement a specific task |
| `/dev test {ID}` | Write tests (TDD) for a task |
| `/dev pr {ID}` | Prepare PR — self-review checklist + PR description |
| `/dev refactor` | Refactor existing code |
| `/dev migrate` | Create database migration (UP + DOWN) |
| `/dev review` | Pre-PR self-review against DoD checklist |

### `/test` — Tester
| Command | Purpose |
|---------|---------|
| `/test cases` | Viết test cases (TEST-xxxxx) từ AC |
| `/test api` | Test API endpoints — status codes, response shape |
| `/test ui` | Test UI/UX — responsive, mobile |
| `/test e2e` | Run E2E automation (full user flow) |
| `/test ws` | Test WebSocket realtime sync |
| `/test bug` | Report bug → BUG-xxxxx in Registry |
| `/test verify` | Retest fixed bugs |

### `/qc` — Quality Control
| Command | Purpose |
|---------|---------|
| `/qc review {ID}` | Code review — architecture, naming, clean code |
| `/qc security` | Security audit (STRIDE — IDOR, injection, exposure) |
| `/qc spec {ID}` | Spec & TDD compliance — Code ↔ BA spec ↔ Architect TDD |
| `/qc audit {ID}` | Full cross-reference: Code = Spec = Test = Registry |
| `/qc perf` | Performance review — N+1, SLO compliance |
| `/qc sign-off {ID}` | Final approval → QC ✅ or REJECT |

### `/devops` — DevOps / SRE
| Command | Purpose |
|---------|---------|
| `/devops deploy {env}` | Deploy to staging or production |
| `/devops rollback` | Rollback (feature flag → container → migration → snapshot) |
| `/devops pipeline` | Design/update CI/CD pipeline |
| `/devops infra` | Infrastructure as Code — provision/update resources |
| `/devops incident {SEV}` | Activate incident response (SEV-1/2/3) |
| `/devops postmortem {ID}` | Write post-mortem after incident |
| `/devops monitor` | Setup/review monitoring & SLO-based alerting |
| `/devops secrets` | Rotate, audit, or add secrets |

### `/architect` — Tech Lead
| Command | Purpose |
|---------|---------|
| `/architect design {feature}` | Write Tech Design Document (TDD) |
| `/architect adr` | Create Architecture Decision Record |
| `/architect review` | Review system design or breaking changes |
| `/architect debt` | Tech debt assessment & planning |
| `/architect capacity` | Capacity & scalability review |
| `/architect diagram` | Generate system/sequence/ER diagrams (Mermaid) |

---

## Task ID Format Reference

All IDs use **5-digit zero-padded** numbers, unique across the entire Registry.

| Type | Format | Example |
|------|--------|---------|
| Feature | `FEAT-xxxxx` | `FEAT-00005` |
| API spec | `API-xxxxx` | `API-00012` |
| Bug fix | `BUG-xxxxx` | `BUG-00003` |
| Refactor | `REFAC-xxxxx` | `REFAC-00001` |
| DevOps | `OPS-xxxxx` | `OPS-00002` |
| UI/UX | `UX-xxxxx` | `UX-00004` |
| Documentation | `DOC-xxxxx` | `DOC-00001` |
| Testing/QA | `QA-xxxxx` | `QA-00007` |
| Test case | `TEST-xxxxx` | `TEST-00001` |

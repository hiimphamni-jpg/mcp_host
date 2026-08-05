# 🤖 GEMINI - MASTER CONTROL

Đây là tài liệu hướng dẫn tối cao quy định cách thức hoạt động của toàn bộ hệ thống AI Engineering trong dự án này. Mọi Agent và Developer đều phải tuân thủ tuyệt đối các quy tắc dưới đây.

---

## 👥 1. PHÂN VAI TRÁCH NHIỆM (Role Definitions)
Mọi yêu cầu bắt đầu bằng mã lệnh sẽ kích hoạt bộ kỹ năng tương ứng:

### 🔄 Workflow Commands (Dùng hàng ngày — ưu tiên dùng trước)

| Lệnh | Mục đích | Skill |
| :--- | :--- | :--- |
| **/think** | Brainstorm ý tưởng, phân tích yêu cầu | `SKILL_WORKFLOW.md §Brainstorm` |
| **/plan** | Lập kế hoạch chi tiết → Gate APPROVED | `SKILL_WORKFLOW.md §Planning` |
| **/build** | Thực thi plan từng step có verification | `SKILL_WORKFLOW.md §Execution` |
| **/fix** | Debug có hệ thống | `SKILL_WORKFLOW.md §Debug` |
| **/review** | Review chất lượng code (Blocker/Major/Minor/Nit) | `SKILL_WORKFLOW.md §Review` |
| **/finish** | Tổng kết, sign-off, cập nhật Registry | `SKILL_WORKFLOW.md §Finish` |

### 👥 Agent Commands (Dùng khi cần chuyên sâu)

| # | Mã lệnh | Vai trò | Agent | Skill | Sản phẩm đầu ra |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | **/ba** | Business Analyst | `.gemini/agents/AGENT_BA.md` | `.gemini/skills/SKILL_BA.md` | Spec & Acceptance Criteria |
| 2 | **/pm** | Project Manager | `.gemini/agents/AGENT_PM.md` | `.gemini/skills/SKILL_PM.md` | `REGISTRY.md` |
| 3 | **/dev** | Senior Developer | `.gemini/agents/AGENT_DEV.md` | `.gemini/skills/SKILL_DEV.md` | Source Code & Migrations |
| 4 | **/test** | Tester | `.gemini/agents/AGENT_TESTER.md` | `.gemini/skills/SKILL_TESTER.md` | `TEST-xxxxx` & Bug Report |
| 5 | **/qc** | Quality Control | `.gemini/agents/AGENT_QC.md` | `.gemini/skills/SKILL_QC.md` | Approve/Reject Sign-off |
| 6 | **/devops** | DevOps / SRE | `.gemini/agents/AGENT_DEVOPS.md` | `.gemini/skills/SKILL_DEVOPS.md` | CI/CD & Deploy |
| 7 | **/architect** | Tech Lead | `.gemini/agents/AGENT_ARCHITECT.md` | `.gemini/skills/SKILL_ARCHITECT.md` | TDD & ADR |

**Quick Reference:** Xem `.gemini/commands/README.md` để biết tất cả slash commands.

---

## ⚙️ 2. QUY TRÌNH PHỐI HỢP (Workflow Pipeline)

### 2.1 Giai đoạn Thiết kế (Architect & BA)
1. **Architect (`/architect`)**: Thiết kế hệ thống, viết TDD (Tech Design), tạo ADR cho features lớn. 
2. **BA (`/ba`)**: Tiếp nhận yêu cầu, viết đặc tả nghiệp vụ chi tiết (API spec, business logic) vào `docs/`.

### 2.2 Giai đoạn Chuẩn bị (PM)
1. **PM (`/pm`)**: Phân rã Spec thành các Task nhỏ (`FEAT-xxxxx`, `API-xxxxx`) với ID 5 chữ số trong `REGISTRY.md`. Đảm bảo DoR (Definition of Ready).

### 2.3 Giai đoạn Thực hiện (Dev)
1. **Dev (`/dev`)**: Nhận ID Task, đọc logic từ BA, TDD từ Architect, viết code theo TDD workflow (Test → Code → Refactor).

### 2.4 Giai đoạn Kiểm soát (Test & QC)
1. **Test (`/test`)**: Chạy các kịch bản `TEST-xxxxx` để xác nhận logic. Report BUG nếu có.
2. **QC (`/qc`)**: Kiểm tra chất lượng code, bảo mật (STRIDE), tuân thủ rules & spec. Sign-off cuối cùng.

### 2.5 Giai đoạn Phát hành (DevOps)
1. **DevOps (`/devops`)**: CI/CD, deploy staging/production, monitoring, incident response.

---

## 📂 3. CẤU TRÚC TÀI LIỆU (Documentation Structure)
```plaintext
├── docs/                         ← Tên thư mục có thể thay đổi theo dự án
│   ├── api_spec.md               # [BA] - API contract & endpoints (hoặc chia nhỏ theo module)
│   ├── business-logic.md         # [BA] - Business logic, formulas, rules
│   ├── api_overview.md           # [BA] - API overview & common patterns (optional)
│   ├── REGISTRY.md               # [PM] - Task registry & status (Single Source of Truth)
│   ├── tdd/                      # [Architect] - Tech Design Documents
│   └── adr/                      # [Architect] - Architecture Decision Records
│
├── .gemini/
│   ├── agents/               # WHO — Role personas (7 files)
│   ├── rules/                # THE LAW — Engineering standards (8 files, +workflow.md)
│   ├── skills/               # HOW — Role methodologies (8 files, +SKILL_WORKFLOW.md)
│   ├── commands/             # QUICK REF — Command cheat sheets (14 files, +6 workflow cmds)
│   └── GEMINI.md             # This file — Master control
│
├── artifacts/                # [Workflow] - Generated outputs (git-ignored)
│   └── superpowers/
│       ├── brainstorm.md     # Output của /think
│       ├── plan.md           # Output của /plan (approved plan)
│       ├── execution.md      # Step-by-step log của /build
│       ├── debug.md          # Output của /fix
│       ├── review.md         # Output của /review
│       └── finish.md         # Output của /finish (final summary)
│
└── tests/                    # [Tester] - Test scripts & reports
```

---

## 📏 4. ENGINEERING RULES (Bộ quy chuẩn bắt buộc)
Tất cả roles phải tuân thủ các rules trong `.gemini/rules/`:

| Rule File | Nội dung | Áp dụng cho |
| :--- | :--- | :--- |
| `rules/workflow.md` | **Plan Gate, TDD, Verification, Artifact Persistence** | **TẤT CẢ** |
| `rules/api-convention.md` | RESTful API design, JSON envelope, status codes | BA, Dev, QC |
| `rules/code-style.md` | Go + TypeScript coding standards | Dev, QC |
| `rules/database.md` | UUID v7, migrations, query optimization | Dev, Architect, QC |
| `rules/error-handling.md` | Structured errors, tracing, zero-silence | Dev, QC |
| `rules/git-workflow.md` | Branching, commits, PR process, CI/CD | Dev, PM, DevOps, QC |
| `rules/testing.md` | TDD, coverage targets, test pyramid | Dev, Tester, QC |
| `rules/security.md` | Auth, IDOR, validation, CORS, secrets | Dev, DevOps, QC |

---

## 🏗️ 5. MA TRẬN PHỤ THUỘC (Interdependency Matrix)

| Vai trò | Phụ thuộc vào (Input) | Sản phẩm bàn giao (Output) | Ai đợi kết quả này? |
| :--- | :--- | :--- | :--- |
| **Architect** | Requirements (User/BA) | TDD, ADR, System Design | BA & Dev |
| **BA** | Yêu cầu từ User, TDD | Spec, Logic, AC | PM & Dev |
| **PM** | Spec của BA | ID Task & Priority (Registry) | Dev & Test |
| **Dev** | ID Task (PM) & Spec (BA) | Code, PR & Unit Test Pass | QC & Test |
| **QC** | Code của Dev & Test results | Kết quả Audit (Sign-off ✅) | PM & Dev (nếu reject) |
| **Test** | PR (Dev) & AC (BA) | Kết quả E2E (Pass/Fail) | PM & Dev (nếu bug) |
| **DevOps** | QC Sign-off | Deployment & Monitoring | PM & Stakeholder |

---

## 📝 6. QUY TẮC KÝ NHẬN (Task Sign-off Rules)
Một Task chỉ được coi là hoàn tất (**DONE**) khi cả 3 cột trong `REGISTRY.md` đạt trạng thái ✅:

1. **[/dev] Hoàn thành code:** Tuân thủ `rules/` + `skills/SKILL_DEV.md`, đã có Migration (nếu cần).
2. **[/test] Hoàn thành kiểm thử:** Tuân thủ `skills/SKILL_TESTER.md`. Nếu lỗi, tạo `BUG-xxxxx`.
3. **[/qc] Duyệt cuối cùng:** Tuân thủ `skills/SKILL_QC.md`. Đổi Global Status thành **DONE**.

---

## 📜 7. KỶ LUẬT THÉP (Core Commandments)
1. **No Spec, No Code**: `/dev` không tự ý viết code nếu chưa có plan được approve và Spec từ `/ba`.
2. **Rules First**: Mọi code phải tuân thủ `.gemini/rules/`. Không có ngoại lệ.
3. **Targeted Patch**: Khi sửa lỗi, chỉ tác động đúng vùng được chỉ định.
4. **Conflict Warning**: Nếu User yêu cầu `/dev` làm trái logic của `/ba`, AI phải cảnh báo ngay lập tức.
5. **Dependency First**: Tuyệt đối tuân thủ Ma trận phụ thuộc. Không làm tắt, không bỏ bước.
6. **Design Before Build**: Features lớn (≥3 tables hoặc ≥2 layers) phải có TDD từ `/architect` trước khi code.
7. **5-Digit IDs**: 100% Task/Test/Bug IDs phải dùng format 5 chữ số zero-padded (ví dụ: FEAT-00005).

> **Note:** Các quy tắc kỹ thuật cụ thể (auth headers, DB primary key format...) được định nghĩa trong `rules/` của từng dự án — không hard-code ở đây.

---

## 🔄 8. ACTIVATION PROTOCOL

### Khi Workflow Command được gọi (`/think`, `/plan`, `/build`, `/fix`, `/review`, `/finish`):
1. Đọc `commands/{command}.md` → nắm quy trình cụ thể.
2. Đọc `skills/SKILL_WORKFLOW.md` → áp dụng methodology (Brainstorm/Planning/TDD/Debug/Review).
3. Đọc `rules/workflow.md` → enforce Plan Gate + Verification + Artifact Persistence.
4. Tự động kích hoạt Agent mindset phù hợp dựa trên loại công việc (xem bảng trong command file).

### Khi Agent Command được gọi (`/ba`, `/dev`, `/architect`...):
1. Đọc `agents/AGENT_*.md` tương ứng → xác định persona + scope.
2. Kiểm tra Prerequisite Status trong `REGISTRY.md` (ví dụ: QC Sign-off check Dev/Test ✅).
3. Đọc `skills/SKILL_*.md` → nắm methodology cụ thể.
4. Đọc `rules/` liên quan → tuân thủ engineering standards.
5. Tham khảo `commands/*.md` → format output chuẩn.

---

## 🚀 9. WORKFLOW THỐNG NHẤT (Superpowers Integration)

### Luồng làm việc hàng ngày (đơn giản nhất)

```
/think [ý tưởng]       → Brainstorm, phân tích
         ↓
/plan [yêu cầu]        → Lập kế hoạch → hỏi APPROVED
         ↓
    APPROVED
         ↓
/build                 → Thực thi từng step có verification
         ↓
/review                → Review trước khi merge
         ↓
/finish                → Tổng kết + sign-off
```

### Khi cần chuyên sâu (kết hợp linh hoạt)

```
/think [ý tưởng]
  → Phát hiện cần spec nghiệp vụ → /ba spec
  → Phát hiện cần system design  → /architect design {feature}
         ↓
/plan (kết hợp output từ /ba và /architect)
         ↓
/build
  → Step code       → kích hoạt /dev skill
  → Step viết test  → kích hoạt /test skill
  → Step migrate DB → kích hoạt /dev skill + rules/database.md
         ↓
/review → /qc review {ID} (nếu cần formal QC)
         ↓
/finish → cập nhật REGISTRY.md → /qc sign-off {ID}
```

### Quick Reference

| Tình huống | Dùng lệnh |
| :--- | :--- |
| Bắt đầu tìm hiểu yêu cầu | `/think [mô tả]` |
| Cần spec nghiệp vụ chi tiết | `/ba spec` |
| Cần system design | `/architect design {feature}` |
| Lập kế hoạch bất kỳ task | `/plan [yêu cầu hoặc TASK-ID]` |
| Thực thi plan | `/build` |
| Bug xuất hiện | `/fix [lỗi]` |
| Trước khi merge PR | `/review` |
| Hoàn thành task | `/finish` hoặc `/finish {TASK-ID}` |
| CI/CD, deploy | `/devops deploy {env}` |
| Task phức tạp cần PM | `/pm registry` hoặc `/pm plan` |

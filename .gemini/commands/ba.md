# /ba Commands

## /ba spec
- **Purpose**: Tạo hoặc cập nhật API specification trong `docs/api_spec.md`
- **Input**: Feature name hoặc REQ-xxxxx reference
- **Output**: Updated `docs/api_spec.md` với endpoints, request/response, error codes
- **Follows**: `rules/api-convention.md` + SKILL_BA §1
- **Split trigger**: Auto-split nếu > 300 lines thành `docs/specs/api/{module}.md`
- **Example**: `/ba spec` → "Viết spec cho feature settlement"

## /ba logic
- **Purpose**: Thiết kế và document business logic, công thức tính toán
- **Input**: Business requirement description
- **Output**: Formulas (INPUT/RULE/OUTPUT format) với ví dụ số cụ thể, rounding rules, edge cases
- **Follows**: SKILL_BA §2 Rule Format
- **Example**: `/ba logic` → "Thiết kế công thức chia tiền ship"

## /ba ac {TASK-ID}
- **Purpose**: Viết Acceptance Criteria (Gherkin format) cho một task
- **Input**: Task ID hoặc feature description
- **Output**: Given/When/Then scenarios — positive + ≥3 negative/boundary cases
- **Follows**: SKILL_BA §3 AC Quality Checklist
- **Example**: `/ba ac FEAT-00005` → "Viết AC cho shipping calculation"

## /ba workflow
- **Purpose**: Vẽ user flow diagram cho feature
- **Input**: Feature description hoặc user story
- **Output**: Mermaid diagram + text description (happy path + error flows)
- **Example**: `/ba workflow` → "Vẽ flow đặt món từ khi mở phòng"

## /ba ticket
- **Purpose**: Viết sprint ticket (Story/Task/Bug/Spike) chuẩn format
- **Input**: Requirement description
- **Output**: Ticket với Summary, Context, AC, Out of Scope, Dependencies, Notes for Dev
- **Follows**: SKILL_BA §6 Ticket Template
- **Example**: `/ba ticket` → "Tạo ticket cho tính năng lock session"

## /ba cr
- **Purpose**: Change Request — document scope change + full impact analysis
- **Input**: What changed + business justification
- **Output**: CR template với Impact Analysis table (API, DB, UI, Logic, Tests) + effort estimate
- **Follows**: SKILL_BA §7 CR Template
- **Example**: `/ba cr` → "CR: thêm field split_type vào settlement"

## /ba audit
- **Purpose**: Gap analysis, edge case review, spec completeness check
- **Input**: Existing spec hoặc feature area
- **Output**: List of gaps, unhandled edge cases ("What If?" framework), proposed solutions
- **Follows**: SKILL_BA §5 Edge Case Analysis
- **Example**: `/ba audit` → "Audit logic thanh toán — concurrent users?"

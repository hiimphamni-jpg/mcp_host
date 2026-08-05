# /pm Commands

## /pm plan
- **Purpose**: Sprint planning — chọn tasks từ backlog theo priority và capacity
- **Input**: Backlog items, team capacity (story points)
- **Output**: Sprint backlog với goal, task list, capacity check (≤ 80% cap)
- **Follows**: SKILL_PM §2 Sprint Planning Process
- **Example**: `/pm plan` → "Lên kế hoạch Sprint 4"

## /pm registry
- **Purpose**: Quản lý Registry — tạo, cập nhật, hoặc query Task ID
- **Input**: Feature/bug description hoặc existing Task ID
- **Output**: Updated `docs/REGISTRY.md` với 5-digit ID (FEAT-00005)
- **ID formats**: FEAT / API / BUG / REFAC / OPS / UX / DOC / QA + `-00000`
- **Example**: `/pm registry` → "Tạo task FEAT-00010 cho settlement UI"

## /pm dor {TASK-ID}
- **Purpose**: Check Definition of Ready — verify task is ready to enter Sprint
- **Input**: Task ID
- **Output**: DoR checklist result — pass ✅ or list of missing items ❌
- **Follows**: SKILL_PM §3 Definition of Ready
- **Example**: `/pm dor FEAT-00010` → "FEAT-00010 ready for Sprint?"

## /pm sprint
- **Purpose**: Sprint lifecycle management (start / close / retro)
- **Input**: Sprint number hoặc action (`start` / `close` / `retro`)
- **Output**: Sprint status report, velocity, burndown summary
- **Example**: `/pm sprint close` → "Đóng Sprint 3, tổng kết velocity"

## /pm status
- **Purpose**: Report sprint progress, velocity, burndown
- **Input**: None (reads REGISTRY.md)
- **Output**: Status dashboard — tasks done / in-progress / todo + commitment accuracy
- **Example**: `/pm status` → "Report Sprint 3 progress hiện tại"

## /pm triage
- **Purpose**: Bug triage và priority assignment
- **Input**: Bug report(s) hoặc BUG-xxxxx ID
- **Output**: Severity + priority assigned, added to sprint or backlog
- **Follows**: SKILL_PM §6 Bug Triage Rules
- **Example**: `/pm triage BUG-00005` → "Đánh giá severity và assign sprint"

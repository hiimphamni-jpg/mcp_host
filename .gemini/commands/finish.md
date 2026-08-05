# /finish — Tổng kết & Sign-off

## Mục đích
Wrap-up sau khi hoàn thành công việc: verification, summary, documentation, QC sign-off.

## Cách dùng
```
/finish              # Tổng kết session hiện tại
/finish {TASK-ID}    # Tổng kết + sign-off task cụ thể
```

## Activation Protocol
Khi `/finish` được gọi, AI tự động áp dụng:
- **AGENT_QC** mindset cho sign-off checklist
- **AGENT_PM** mindset để cập nhật Registry status
- `SKILL_QC.md §Sign-off Checklist`
- `SKILL_WORKFLOW.md §Finish`
- Đọc `artifacts/superpowers/execution.md` để tổng hợp

## Quy trình thực hiện

### Bước 1: Thu thập
Đọc các artifacts có sẵn:
- `artifacts/superpowers/plan.md` — plan đã approve
- `artifacts/superpowers/execution.md` — log thực thi
- `artifacts/superpowers/review.md` — kết quả review (nếu có)

### Bước 2: Verification Pass
Chạy (hoặc liệt kê để user chạy):
- Test commands từng step trong plan
- Lint/build check
- Các manual validation steps

### Bước 3: Review Pass nhanh
Nếu chưa có `/review`, chạy review pass nhanh:
- Blockers? Major? → Dừng, yêu cầu fix
- OK → Tiếp tục

### Bước 4: Ghi Summary
Ghi vào `artifacts/superpowers/finish.md`

### Bước 5: Cập nhật Registry (nếu có TASK-ID)
- Cập nhật status trong `docs/REGISTRY.md` sang DONE hoặc status phù hợp
- Gợi ý `/qc sign-off {ID}` để QC approval chính thức

## Output Format

```markdown
## Finish Summary

### ✅ Verification Results
| Command | Expected | Result |
|---------|----------|--------|
| `test cmd` | pass | ✅ PASS |

### 📝 Summary of Changes
- [file]: [thay đổi gì]
- [file]: [thay đổi gì]

### 🔍 Review Pass
- Blockers: 0
- Majors: 0
- Overall: ✅ APPROVED

### 📋 Follow-ups
- [ ] Việc còn lại (nếu có)

### 🧪 Manual Validation Steps
1. [Hướng dẫn test thủ công nếu cần]

### 📌 Task Status
- TASK-ID: [ID] → [NEW STATUS]
- Next: `/qc sign-off {ID}` để QC approval
```

## Persist (bắt buộc)
```bash
mkdir -p artifacts/superpowers
```
Ghi output vào `artifacts/superpowers/finish.md`.
Xác nhận bằng cách list `artifacts/superpowers/`.

## Sau khi finish
- Nếu cần QC approval → `/qc sign-off {ID}`
- Nếu ready to deploy → `/devops deploy staging`
- Nếu có issues còn lại → tạo BUG-xxxxx trong Registry

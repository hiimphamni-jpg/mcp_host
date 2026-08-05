# /plan — Lập kế hoạch chi tiết (Plan Gate)

## Mục đích
Tạo kế hoạch thực thi từng bước với verification. Bắt buộc có approval trước khi code.

## Cách dùng
```
/plan [mô tả yêu cầu]          # Yêu cầu tự do
/plan [TASK-ID]                 # Từ Registry (ưu tiên)
/plan FEAT-00005 implement cart # Kết hợp ID + mô tả
```

## Preconditions
- Nếu có `TASK-ID`: đọc `docs/REGISTRY.md` → lấy AC từ `/ba` spec → lấy TDD từ `/architect`
- Nếu không có `TASK-ID`: yêu cầu user mô tả rõ mục tiêu trong 1 câu

## Quy trình thực hiện

### Bước 1: Thu thập context
1. Đọc `artifacts/superpowers/brainstorm.md` nếu có (từ `/think`)
2. Nếu có TASK-ID:
   - Đọc `docs/REGISTRY.md` → trạng thái task
   - Đọc `docs/api_spec.md` → BA spec
   - Đọc `docs/tdd/` → Architect TDD
3. Scan codebase liên quan (đọc files, không sửa)

### Bước 2: Tạo plan
Mỗi step phải có:
- **Files**: File nào sẽ tạo/sửa
- **Change**: Thay đổi gì cụ thể
- **Agent**: Agent nào chịu trách nhiệm step này
- **Verify**: Lệnh/cách verify step hoạt động đúng
- **Duration**: Ước tính 2–10 phút

### Bước 3: Phân tích parallel
Sau khi có plan, kiểm tra:
- Có ≥2 steps độc lập nhau không? (không cùng sửa 1 file, không phụ thuộc nhau)
- Nếu có → ghi rõ "⚡ Step X, Y có thể chạy song song"

### Bước 4: Persist
Ghi plan vào `artifacts/superpowers/plan.md`

### Bước 5: Approval Gate ⛔
Hỏi user:
> **"Plan đã sẵn sàng. Gõ APPROVED để tiến hành, hoặc yêu cầu điều chỉnh."**

Nếu user trả lời `APPROVED`:
- KHÔNG implement ngay
- Trả lời: **"Plan approved ✅. Dùng `/build` để bắt đầu thực thi."**

## Output Format (bắt buộc)

```markdown
## Goal
[Mục tiêu cụ thể]

## Assumptions
[Các giả định đã được chấp nhận]

## Plan

### Step 1: [Tên ngắn gọn]
- **Files**: `path/to/file.go`
- **Agent**: /dev (hoặc /ba, /architect, /test...)
- **Change**: Mô tả thay đổi cụ thể
- **Verify**: `go test ./...` hoặc lệnh verify cụ thể
- **Duration**: ~5 min

### Step 2: ...

## Risks & Mitigations
[Rủi ro + cách xử lý]

## Rollback Plan
[Cách rollback nếu có lỗi]

## Parallel Opportunities
[Step nào có thể chạy song song, nếu có]
```

## Persist (bắt buộc)
```bash
mkdir -p artifacts/superpowers
```
Ghi toàn bộ output vào `artifacts/superpowers/plan.md`.
Xác nhận file bằng cách list `artifacts/superpowers/`.

## Lưu ý
- KHÔNG viết code trong workflow này — chỉ plan
- Plan steps phải nhỏ (2–10 phút mỗi step)
- Nếu yêu cầu phức tạp → gọi `/think` trước
- Nếu cần spec nghiệp vụ → `/ba spec` → rồi mới `/plan`
- Nếu feature lớn (≥3 tables hoặc ≥2 layers) → `/architect design` → rồi mới `/plan`

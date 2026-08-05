# /build — Thực thi Plan từng bước

## Mục đích
Thực thi plan đã được approve, từng step một, với verification sau mỗi step.

## Cách dùng
```
/build              # Thực thi toàn bộ plan
/build step 2       # Chỉ thực thi step 2 (resume/retry)
/build parallel     # Thực thi các step độc lập song song
```

## Preconditions (không bỏ qua)

1. User đã trả lời `APPROVED` cho plan
2. File `artifacts/superpowers/plan.md` phải tồn tại

Nếu không có `artifacts/superpowers/plan.md`:
- **Dừng ngay**
- Yêu cầu user chạy `/plan [yêu cầu]` trước
- KHÔNG viết code

## Quy trình thực hiện

### Bước 1: Load plan
- Đọc `artifacts/superpowers/plan.md`
- Tóm tắt plan trong 1–2 dòng
- Liệt kê các steps sẽ thực hiện

### Bước 2: Kiểm tra Parallel (nếu plan có đánh dấu ⚡)
Nếu plan có "Parallel Opportunities":
> "Tôi thấy Step X, Y có thể chạy song song. Bạn muốn:
> - `PARALLEL` — nhanh hơn ~60% nhưng khó debug hơn
> - `SEQUENTIAL` — tuần tự, dễ theo dõi hơn (mặc định)"

Chờ user trả lời. Nếu không có parallel → tiếp tục sequential.

### Bước 3: Thực thi từng Step

Cho mỗi step, áp dụng Agent + Skill phù hợp:

| Step Type | Agent Kích hoạt | Skill Áp dụng |
|---|---|---|
| Implement code | **AGENT_DEV** | `SKILL_DEV.md` + `SKILL_WORKFLOW.md §TDD` |
| Viết tests | **AGENT_TESTER** | `SKILL_TESTER.md` + `SKILL_WORKFLOW.md §TDD` |
| Database migration | **AGENT_DEV** | `SKILL_DEV.md` + `rules/database.md` |
| API design update | **AGENT_BA** | `SKILL_BA.md` |
| DevOps/infra | **AGENT_DEVOPS** | `SKILL_DEVOPS.md` |
| Architecture decision | **AGENT_ARCHITECT** | `SKILL_ARCHITECT.md` |

**Sau mỗi step:**
1. Chạy lệnh verify trong plan (hoặc cung cấp lệnh + expected output nếu không tự chạy được)
2. Append vào `artifacts/superpowers/execution.md`:
   ```markdown
   ## Step N: [Tên step]
   - Files thay đổi: [...]
   - Thay đổi: [1–3 bullets]
   - Verify: `[lệnh]` → [kết quả: PASS/FAIL]
   ```
3. Nếu FAIL:
   - **Dừng ngay**
   - Switch sang `/fix [mô tả lỗi]`
   - KHÔNG tiếp tục step tiếp theo

### Bước 4: Finish
Sau khi tất cả steps hoàn thành:
1. Chạy review pass nhanh (Blocker/Major/Minor/Nit)
2. Ghi summary vào `artifacts/superpowers/finish.md`
3. Gợi ý: dùng `/finish` để QC sign-off nếu cần

## Persist (bắt buộc)

```bash
mkdir -p artifacts/superpowers
```

- Append từng step vào `artifacts/superpowers/execution.md`
- Ghi summary cuối vào `artifacts/superpowers/finish.md`
- Xác nhận bằng cách list `artifacts/superpowers/`

## Execution Rules (nghiêm ngặt)
1. **Một step tại một thời điểm** (trừ khi chạy parallel)
2. **Không bỏ qua verification** — nếu không chạy được lệnh, cung cấp lệnh + expected output rõ ràng
3. **Minimal changes** — chỉ sửa đúng những gì plan quy định
4. **Nếu plan sai/thiếu step** → dừng, cập nhật `plan.md`, hỏi approval lại nếu thay đổi lớn
5. **Tuân thủ TDD** — ưu tiên viết test trước khi implement (xem `SKILL_WORKFLOW.md §TDD`)

## Sau khi xong
- Nếu là FEAT/API task → gợi ý `/test cases {ID}` để viết test cases
- Nếu là bug fix → gợi ý `/qc review {ID}`
- Nếu push production → gợi ý `/finish` để QC sign-off

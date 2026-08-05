# /think — Brainstorm & Khám phá ý tưởng

## Mục đích
Brainstorm ý tưởng, phân tích yêu cầu, nhận diện rủi ro trước khi lập kế hoạch.
Tự động kích hoạt mindset phù hợp dựa trên loại câu hỏi.

## Cách dùng
```
/think [mô tả ý tưởng hoặc vấn đề]
```

## Activation Protocol
Khi `/think` được gọi, AI phân loại câu hỏi và áp dụng mindset:

| Loại yêu cầu | Mindset kích hoạt | Skill áp dụng |
|---|---|---|
| Thiết kế hệ thống, kiến trúc | **AGENT_ARCHITECT** | `skills/SKILL_ARCHITECT.md` |
| Yêu cầu nghiệp vụ, user story | **AGENT_BA** | `skills/SKILL_BA.md` |
| Lập kế hoạch sprint, task | **AGENT_PM** | `skills/SKILL_PM.md` |
| Tâm tư chung / khám phá tự do | **WORKFLOW mindset** | `skills/SKILL_WORKFLOW.md` §Brainstorm |

## Quy trình thực hiện

1. **Nhận input** từ user
2. **Phân loại** ý tưởng → chọn mindset phù hợp
3. **Đặt câu hỏi làm rõ** (2–5 câu) nếu cần
4. **Output** theo format chuẩn bên dưới
5. **Persist** → ghi file `artifacts/superpowers/brainstorm.md`
6. **Không implement code** — dừng lại sau bước persist

## Output Format (dùng đúng thứ tự)

```markdown
## Goal
[Mục tiêu thực sự là gì?]

## Constraints
[Ràng buộc kỹ thuật / nghiệp vụ / thời gian]

## Known Context
[Thông tin từ codebase/spec hiện tại nếu có]

## Risks
[Rủi ro tiềm ẩn — kỹ thuật, nghiệp vụ, UX]

## Options (2–4 phương án)
[Liệt kê các hướng tiếp cận khác nhau]

## Recommendation
[Hướng đề xuất + lý do]

## Acceptance Criteria
[Điều kiện để coi là "done"]
```

## Persist (bắt buộc)
Sau khi output xong, ghi vào disk:
```bash
mkdir -p artifacts/superpowers
```
Ghi nội dung brainstorm vào `artifacts/superpowers/brainstorm.md`.
Xác nhận file tồn tại bằng cách liệt kê `artifacts/superpowers/`.

## Bước tiếp theo
Sau khi brainstorm xong, gợi ý:
> "Bạn có muốn tôi tạo plan chi tiết? Dùng `/plan [tóm tắt yêu cầu]`"

## Lưu ý
- KHÔNG viết code trong workflow này
- Nếu cần spec nghiệp vụ chi tiết → gọi `/ba spec`
- Nếu cần system design → gọi `/architect design {feature}`

# Workflow Rules — Plan Gate & Quality Enforcement

Các quy tắc này áp dụng cho **TẤT CẢ công việc** trong session, tự động kích hoạt song song với rules hiện tại.

Nguồn: Tích hợp từ [Gemini Superpowers](https://github.com/obra/superpowers) framework.

---

## 1. Plan Gate — Bắt buộc có plan trước khi code

### Quy tắc

> ⛔ **Không được viết/sửa code nếu chưa có plan được APPROVED**

- **Non-trivial task**: Plan đầy đủ + hỏi `APPROVED` → chờ user confirm → mới implement
- **Tiny task** (1 file, rõ ràng, low risk): Mini-plan 3–5 steps vẫn phải có
- Sau khi user reply `APPROVED`: KHÔNG implement ngay → hướng dẫn `/build`

### Định nghĩa "tiny"
- Sửa 1 file duy nhất và thay đổi rõ ràng
- Không có business logic phức tạp
- Không có DB migration
- Không có breaking change

### Quy tắc vàng
```
Think first → Plan next → Get APPROVED → Then build
```

---

## 2. Verification — Bắt buộc sau mỗi bước

Sau mỗi implementation step, phải cung cấp:
- **Exact commands** để verify (test/lint/run)
- **Expected output** nếu không thể tự chạy
- **Actual results** nếu tự chạy được

Tuyệt đối không được nói "done" mà không có verification evidence.

---

## 3. TDD — Ưu tiên viết test trước

- Fixing bug → **bắt buộc** thêm regression test nếu thực tế
- Adding feature → thêm/adjust tests khi có thể
- Nếu test không thực tế → cung cấp concrete verification path thay thế

---

## 4. Review Pass — Bắt buộc trước khi finish

Trước khi declare "done", chạy review pass và liệt kê:
- 🔴 Blockers
- 🟠 Majors
- 🟡 Minors
- ⚪ Nits

Có Blocker → KHÔNG declare done → fix trước.

---

## 5. Artifact Persistence — Bắt buộc lưu ra disk

| Khi nào | Ghi vào |
|---|---|
| Sau brainstorm (`/think`) | `artifacts/superpowers/brainstorm.md` |
| Sau planning (`/plan`) | `artifacts/superpowers/plan.md` |
| Sau mỗi execution step (`/build`) | `artifacts/superpowers/execution.md` |
| Sau debug (`/fix`) | `artifacts/superpowers/debug.md` |
| Sau review (`/review`) | `artifacts/superpowers/review.md` |
| Sau finish (`/finish`) | `artifacts/superpowers/finish.md` |

**Không để output chỉ tồn tại trong chat.**  
Sau khi ghi file, xác nhận bằng cách list `artifacts/superpowers/`.

---

## 6. Safety Rules

- ❌ Không log secrets, tokens, passwords
- ⏱️ API automations phải có timeout + retry + idempotency
- 💾 Fail safe — không cho phép silent data loss
- 🔒 Mọi external API call phải được timeout

---

## Mối quan hệ với Rules hiện tại

Rule file này **bổ sung** vào các rules hiện tại, không thay thế:
- `rules/api-convention.md` — vẫn áp dụng đầy đủ
- `rules/code-style.md` — vẫn áp dụng đầy đủ  
- `rules/database.md` — vẫn áp dụng đầy đủ
- `rules/error-handling.md` — vẫn áp dụng đầy đủ
- `rules/git-workflow.md` — vẫn áp dụng đầy đủ
- `rules/testing.md` — vẫn áp dụng đầy đủ
- `rules/security.md` — vẫn áp dụng đầy đủ

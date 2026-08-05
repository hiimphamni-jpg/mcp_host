---
name: SKILL_WORKFLOW
description: Workflow methodology - tích hợp Superpowers framework (brainstorm/plan/execute/debug/review/finish) vào bộ Agent. Áp dụng khi dùng /think, /plan, /build, /fix, /review, /finish.
---

# SKILL_WORKFLOW — Workflow Methodology

Skill này được tích hợp từ framework **Gemini Superpowers** (adapted from Claude Superpowers).  
Áp dụng tự động khi dùng các lệnh: `/think`, `/plan`, `/build`, `/fix`, `/review`, `/finish`.

---

## § Brainstorm

### Khi nào dùng
- Trước khi plan một feature mới
- Khi có nhiều cách tiếp cận, cần phân tích trade-offs
- Khi yêu cầu chưa rõ ràng

### Quy tắc
- KHÔNG viết code trong giai đoạn brainstorm
- Đặt câu hỏi làm rõ nếu yêu cầu mơ hồ (2–4 câu)
- Output phải gồm: Goal, Constraints, Risks, Options, Recommendation, AC
- Persist vào `artifacts/superpowers/brainstorm.md`

---

## § Planning

### Quy tắc Plan Gate
> ⛔ **Bắt buộc**: Không được viết code trước khi có plan được **APPROVED**

1. Với **tiny changes** (1 file, obvious edit, low risk): mini-plan 3–5 steps vẫn phải có
2. Với **non-trivial changes**: Full plan + hỏi APPROVED
3. Sau khi user reply APPROVED: KHÔNG implement → hướng dẫn dùng `/build`

### Format Plan chuẩn
Mỗi step trong plan phải có:
```
### Step N: [Tên]
- Files: [path]
- Agent: /dev | /ba | /test | ...
- Change: [mô tả cụ thể]
- Verify: [lệnh hoặc cách verify]
- Duration: ~X min
```

### Phân tích Parallel
Sau khi có plan, kiểm tra:
- Step nào không phụ thuộc nhau (khác file, khác module)?
- Đánh dấu `⚡ Parallel: Step X, Y` nếu có
- ≥3 independent steps → đánh giá có nên gợi ý parallel mode

---

## § TDD (Test-Driven Development)

### Khi nào áp dụng
- Feature mới có thể unit test
- Bug fix (luôn add regression test nếu có thể)
- Refactor (bảo vệ behavior hiện tại trước)

### Quy trình Red → Green → Refactor
1. Define behavior change (sau khi làm xong, điều gì phải đúng?)
2. Viết/adjust test để capture nó (make it fail first)
3. Implement minimal change để pass test
4. Refactor nếu cần (giữ nguyên passing)
5. Chạy test suite + linters

### Khi test không thực tế
Vẫn phải có verification alternative:
- Minimal repro script
- Integration test
- Manual steps rõ ràng + expected output

### Output requirements
Khi thay đổi code, phải ghi rõ:
- Tests nào đã thêm/thay đổi
- Cách chạy: `[exact command]`
- Tests chứng minh điều gì

---

## § Execution

### Strict Rules
1. **Một step tại một thời điểm** — không nhảy cóc
2. **Verification bắt buộc** sau mỗi step
3. **Stop on fail** — nếu verify fail → switch sang `/fix`
4. **Minimal changes** — chỉ đúng scope plan
5. **Artifact logging** — append mỗi step vào `execution.md`

### Execution Log Format (per step)
```markdown
## Step N: [Tên]
- Files: [file đã sửa]
- Changes:
  - [bullet 1]
  - [bullet 2]
- Verify: `[lệnh]` → [PASS / FAIL / NOT_RUN]
- Notes: [ghi chú nếu có deviation]
```

### Parallel Execution
Khi chạy parallel (user chọn PARALLEL):
1. Xác định batches (nhóm steps có thể chạy cùng nhau)
2. Spawn subagents của Antigravity cho mỗi batch
3. Chờ tất cả hoàn thành trước khi tiếp batch tiếp theo
4. Verify integration sau khi merge kết quả

---

## § Debug

### Quy trình (không bỏ bước)
1. **Reproduce** — exact error + inputs + environment
2. **Minimize** — smallest repro case
3. **Hypotheses** — 2–5 giả thuyết, xếp hạng theo likelihood
4. **Instrument** — logging/assertions tạm thời
5. **Fix** — smallest change loại bỏ root cause
6. **Prevent** — regression test hoặc guard vĩnh viễn
7. **Verify** — chạy lại failing case + related suite

### Debug Report Format
```markdown
### Symptom: [...]
### Repro Steps: [...]
### Root Cause: [...]
### Fix: [...]
### Regression Protection: [...]
### Verification: [command] → [PASS/FAIL]
```

---

## § Review

### Severity Levels
| Level | Định nghĩa |
|---|---|
| 🔴 **Blocker** | Wrong behavior, security issue, data loss, broken build/tests |
| 🟠 **Major** | Likely bug, missing edge cases, poor reliability |
| 🟡 **Minor** | Style, clarity, small maintainability |
| ⚪ **Nit** | Optional polish |

### Checklist
1. Correctness vs requirements & AC
2. Edge cases & error handling (`rules/error-handling.md`)
3. Tests (coverage, assertions meaningful)
4. Security (STRIDE: secrets, auth, IDOR, injection — `rules/security.md`)
5. Performance (N+1, timeouts, unnecessary work)
6. Readability & maintainability (`rules/code-style.md`)
7. Docs/comments cập nhật nếu cần

---

## § Finish

### Checklist hoàn thành
1. Verification commands chạy pass
2. Review pass done (không có Blocker)
3. `execution.md` đầy đủ
4. `finish.md` được ghi
5. Follow-ups được list rõ
6. Manual validation steps được document

### Artifact Persistence Summary
| Workflow | File |
|---|---|
| `/think` output | `artifacts/superpowers/brainstorm.md` |
| `/plan` output | `artifacts/superpowers/plan.md` |
| `/build` step logs | `artifacts/superpowers/execution.md` |
| `/fix` debug report | `artifacts/superpowers/debug.md` |
| `/review` output | `artifacts/superpowers/review.md` |
| `/finish` summary | `artifacts/superpowers/finish.md` |

---

## § Safety Rules

- ❌ **Never log secrets** — không log tokens, passwords, keys
- ⏱️ **Timeouts mandatory** — mọi external API call phải có timeout
- 🔄 **Idempotency** — automation scripts phải idempotent
- ❌ **No silent failures** — luôn surface errors, không fail silently
- 💾 **Fail safe** — không bao giờ silent data loss

# /review — Review chất lượng code

## Mục đích
Review code trước khi merge/ship, phân loại issue theo severity, đảm bảo tuân thủ standards.

## Cách dùng
```
/review                  # Review code hiện tại (git diff)
/review {TASK-ID}        # Review task cụ thể
/review security         # Chỉ focus security audit
/review perf             # Chỉ focus performance
```

## Activation Protocol
Khi `/review` được gọi, AI tự động áp dụng:
- **AGENT_QC** mindset chính (code quality, compliance)
- **AGENT_DEV** mindset cho code correctness
- `SKILL_QC.md` methodology đầy đủ
- `SKILL_WORKFLOW.md §Review` checklist
- `rules/code-style.md`, `rules/security.md`, `rules/testing.md`

## Checklist Review

### 🔴 Correctness
- [ ] Logic đúng với requirements/AC?
- [ ] Edge cases được handle?
- [ ] Error handling đầy đủ (tuân thủ `rules/error-handling.md`)?

### 🟠 Security (STRIDE)
- [ ] Không hardcode secrets/credentials?
- [ ] Input validation đúng chỗ?
- [ ] IDOR risks được address?
- [ ] Auth/authz đúng?
- [ ] CORS config an toàn?

### 🟡 Tests
- [ ] Coverage đủ cho business logic?
- [ ] Assertions có ý nghĩa (không chỉ assert not nil)?
- [ ] Edge cases có test?

### 🔵 Performance
- [ ] N+1 queries?
- [ ] Unnecessary loops/allocations?
- [ ] Timeouts cho external calls?

### ⚪ Style & Maintainability
- [ ] Code style tuân thủ `rules/code-style.md`?
- [ ] Naming rõ ràng?
- [ ] Comments/docs cập nhật?

## Severity Levels

| Level | Ý nghĩa | Action |
|---|---|---|
| **🔴 Blocker** | Wrong behavior, security hole, data loss, broken build | Phải fix TRƯỚC khi merge |
| **🟠 Major** | Likely bug, missing edge case, poor reliability | Nên fix trong PR này |
| **🟡 Minor** | Style, clarity, small maintainability | Fix nếu có thời gian |
| **⚪ Nit** | Optional polish | Take it or leave it |

## Output Format

```markdown
## Code Review

### 🔴 Blockers
- [file:line] Mô tả vấn đề + cách fix

### 🟠 Majors  
- [file:line] Mô tả vấn đề

### 🟡 Minors
- [file:line] Mô tả

### ⚪ Nits
- [file:line] Suggestion

### ✅ Summary
Tổng X blockers, Y majors. [APPROVED / NEEDS WORK]
Next actions: [...]
```

## Persist (bắt buộc)
```bash
mkdir -p artifacts/superpowers
```
Ghi output vào `artifacts/superpowers/review.md`.

## Sau khi review
- Nếu có Blocker → yêu cầu fix + `/review` lại
- Nếu APPROVED → gợi ý `/finish` để QC sign-off chính thức
- Nếu Task có ID → gợi ý `/qc sign-off {ID}`

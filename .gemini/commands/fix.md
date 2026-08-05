# /fix — Debug có hệ thống

## Mục đích
Debug lỗi theo quy trình có cấu trúc: tái hiện → phân tích → sửa → ngăn chặn tái phát.

## Cách dùng
```
/fix [mô tả lỗi/triệu chứng]
/fix BUG-00003           # Bug đã có ID trong Registry
/fix "TypeError: ..."    # Paste error message trực tiếp
```

## Activation Protocol
Khi `/fix` được gọi, AI tự động áp dụng:
- **AGENT_DEV** mindset cho code-level bugs
- **AGENT_QC** mindset để đánh giá impact  
- `SKILL_WORKFLOW.md §Debug` methodology
- `rules/error-handling.md` để đảm bảo fix đúng chuẩn

## Quy trình Debug (không bỏ bước)

### 1. Reproduce 🔴
- Capture chính xác: error message, inputs, môi trường, lệnh chạy
- Xác nhận bug tái hiện được nhất quán

### 2. Minimize 🔬
- Thu nhỏ về repro case nhỏ nhất (1 file, 1 function, dataset nhỏ nhất)
- Bỏ bớt các yếu tố không liên quan

### 3. Hypotheses 💡
- Đặt ra 2–5 giả thuyết về nguyên nhân
- Xếp hạng theo khả năng xảy ra (1 = cao nhất)

### 4. Instrument 🔧
- Thêm logging tạm thời / assertions / diagnostics
- Dùng kết quả để loại bỏ/xác nhận giả thuyết

### 5. Fix 🛠️
- Thay đổi nhỏ nhất loại bỏ root cause
- Không fix nhiều thứ cùng lúc

### 6. Prevent 🛡️
- Thêm regression test (nếu thực tế)
- Thêm guard/validation vĩnh viễn
- Tuân thủ `rules/error-handling.md`

### 7. Verify ✅
- Chạy lại failing case
- Chạy test suite liên quan

## Output Format (bắt buộc)

```markdown
## Bug Report

### Symptom
[Mô tả lỗi quan sát được]

### Repro Steps
1. ...
2. ...

### Root Cause
[Nguyên nhân gốc rễ]

### Fix
[File thay đổi + mô tả thay đổi]

### Regression Protection
[Test case hoặc guard thêm vào]

### Verification
- Command: `[lệnh]`
- Expected: `[kết quả mong đợi]`
- Result: PASS / FAIL
```

## Persist (bắt buộc)
```bash
mkdir -p artifacts/superpowers
```
Ghi output vào `artifacts/superpowers/debug.md`.
Xác nhận bằng cách list `artifacts/superpowers/`.

## Sau khi fix xong
- Nếu bug có ID trong Registry → cập nhật status sang FIXED
- Gợi ý: `/test verify BUG-{ID}` để verify fix
- Gợi ý: `/review` để đảm bảo fix không gây side effects

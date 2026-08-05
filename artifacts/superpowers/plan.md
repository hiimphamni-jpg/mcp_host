# 🗓️ Implementation Plan — Custom MCP Host

## 1. Goal
Xây dựng thành công một Custom MCP Host có khả năng kết nối LLM với MCP Server thông qua stdio, thực hiện được vòng lặp Agentic hoàn chỉnh.

## 2. Tasks Breakdown (from REGISTRY.md)

### Phase 1: Go Backend (Sprint 1)
- **FEAT-00001**: Khởi tạo project Go, setup `mark3labs/mcp-go`.
- **FEAT-00002**: Implement MCP Client và Tool Mapping sang Gemini format.
- **FEAT-00003**: Xây dựng Agentic Loop với Gemini Go SDK.

### Phase 2: Next.js Frontend (Sprint 1)
- **FEAT-00004**: Khởi tạo Next.js App, xây dựng UI Chat cơ bản.
- **FEAT-00005**: Tích hợp SSE để nhận kết quả realtime từ Go Backend.

### Phase 3: Quality & Refinement (Sprint 1/2)
- **FEAT-00007**: Robust error handling (JSON-RPC errors, LLM hallucination on tool args).
- **QA-00001**: Viết E2E tests tích hợp thực tế với `@modelcontextprotocol/server-filesystem`.

## 3. Milestones
- **M1**: Host có thể list được tools từ một MCP Server bất kỳ.
- **M2**: Host có thể gửi tool list cho LLM và nhận lại `tool_call` hợp lệ.
- **M3**: Agentic Loop hoạt động (LLM gọi tool -> Host thực thi -> LLM tổng hợp kết quả).

## 4. Verification Strategy
- **Unit Test**: Mock MCP Server và LLM API.
- **Integration Test**: Chạy thực tế với local filesystem.
- **Manual Test**: Prompt LLM đọc một file cụ thể trong máy và tóm tắt nội dung.

## 5. Next Steps
1. Yêu cầu user phê duyệt kế hoạch (`APPROVED`).
2. Sử dụng lệnh `/build` để bắt đầu thực hiện `FEAT-00001`.

# 📄 Business Logic — Custom MCP Host

## 1. Overview
Hệ thống đóng vai trò là một "Orchestrator" (người điều phối) kết nối LLM (bộ não) với các MCP Server (công cụ thực thi).

## 2. Core Workflows

### 2.1 Full-Stack Workflow
1. **User Interaction**: User nhập prompt trên giao diện Next.js.
2. **Request**: Next.js gửi request tới Go API Gateway.
3. **MCP Interaction**: Go Host (MCP Client) tương tác với MCP Servers qua stdio.
4. **Reasoning**: Go Host điều phối Gemini API để quyết định hành động.
5. **Streaming**: Kết quả (text + tool logs) được stream ngược về Next.js qua SSE.

### 2.2 LLM Tool Mapping
- **Input**: MCP Tool Schema (dựa trên JSON Schema).
- **Transformation**: Chuyển đổi sang định dạng `tools` của Gemini/DeepSeek (thường là `function_declarations`).
- **Output**: Một danh sách các công cụ mà LLM có thể gọi.

### 2.3 Agentic Loop (Recursive Reasoning)
1. **Prompt**: User gửi yêu cầu.
2. **LLM Inference**: LLM phân tích yêu cầu và quyết định gọi tool hay trả lời text.
3. **Tool Call**: 
   - Nếu LLM yêu cầu `tool_call`, Host trích xuất tham số.
   - Host gọi `tools/call` xuống MCP Server.
   - Host nhận kết quả từ MCP Server.
4. **Iteration**: Host nạp kết quả vào lịch sử hội thoại và gọi lại LLM.
5. **Final Response**: Khi LLM không gọi tool nữa, trả về kết quả cuối cùng cho User.

## 3. Business Rules
- **Security**: Tuyệt đối không thực thi các lệnh shell trực tiếp từ LLM mà không thông qua MCP Server đã định nghĩa.
- **Timeout**: Mọi kết nối MCP Server phải có timeout (mặc định 30s).
- **Concurrency**: Host hỗ trợ kết nối đồng thời nhiều MCP Server (Filesystem + DB).
- **Privacy**: Không lưu trữ log nhạy cảm từ MCP Server vào log của LLM trừ khi cần thiết cho context.

## 4. Acceptance Criteria (AC)
- **AC-01**: Kết nối thành công với `@modelcontextprotocol/server-filesystem` qua stdio.
- **AC-02**: Mapping chính xác schema từ MCP sang LLM.
- **AC-03**: Hoàn thành ít nhất 3 vòng lặp tool call liên tục cho một nhiệm vụ phức tạp.
- **AC-04**: Xử lý lỗi khi MCP Server bị treo hoặc crash.

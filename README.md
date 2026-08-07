# MCP Host

Một **MCP (Model Context Protocol) host agentic loop** viết bằng Go: kết nối với LLM Gemini để điều khiển các MCP tools (mặc định là filesystem server). Agent chạy vòng lặp có giới hạn: LLM đề xuất tool call → host thực thi qua MCP → kết quả feedback lại → lặp đến khi model trả lời cuối cùng.

## Yêu cầu

- **Go 1.26+**
- **Node.js + npx** (MCP filesystem server chạy qua `npx`)
- **Gemini API key**

## Cấu hình (`.env`)

Copy từ `.env.example` và điền key:

```dotenv
# Required Gemini configuration
GEMINI_API_KEY=YOUR_GEMINI_API_KEY
GEMINI_MODEL=gemini-3.1-flash-lite

# Filesystem MCP config (JSON array)
MCP_FILESYSTEM_COMMAND=npx
MCP_FILESYSTEM_ARGS_JSON=["-y","@modelcontextprotocol/server-filesystem"]
MCP_ALLOWED_ROOTS=D:\Mihi

# Execution limits
MCP_TIMEOUT=30s
LLM_TIMEOUT=45s
AGENT_MAX_ITERATIONS=8
MCP_MAX_RESULT_BYTES=65536
HOST_LOG_LEVEL=info
```

> Lưu ý: `GEMINI_MODEL` phải là model hỗ trợ tool-calling (vd `gemini-3.1-flash-lite`). `MCP_ALLOWED_ROOTS` giới hạn vùng filesystem agent được truy cập.

## Chạy dự án

**1. Chạy với `--prompt`:**
```powershell
go run ./cmd/server --prompt "Liệt kê các file trong thư mục sandbox"
```

**2. Hoặc pipe prompt qua stdin:**
```powershell
"Liệt kê các file trong thư mục sandbox" | go run ./cmd/server
```

**3. Hoặc build và chạy binary:**
```powershell
go build -o mcp-host.exe ./cmd/server
.\mcp-host.exe --prompt "Đọc nội dung file sandbox/hello.txt"
```

Chạy không có prompt sẽ in bootstrap: `MCP Host bootstrap complete. Integrations are not configured or started.`

## Ví dụ prompt

```powershell
# Liệt kê / đọc / ghi file trong sandbox
go run ./cmd/server --prompt "Liệt kê các file trong thư mục sandbox"
go run ./cmd/server --prompt "Tạo file test.txt nội dung 'xin chào' rồi đọc lại"
go run ./cmd/server --prompt "Tổng kích thước các file trong D:\Mihi"
```

## Chạy HTTP Gateway (SSE)

Gateway mở `POST /v1/chat` (SSE) authenticate bằng Bearer token, kèm `GET /healthz`. Cấu hình thêm trong `.env`:

```dotenv
# Gateway
GATEWAY_ADDR=:8080
GATEWAY_API_TOKEN=dev-gateway-token-00008
GATEWAY_API_TOKENS=            # optional, dấu phẩy: token phụ cho rotation
GATEWAY_MAX_CONCURRENT=8
```

Thiếu `GATEWAY_API_TOKEN` thì gateway từ chối chạy.

**Khởi động:**
```powershell
go run ./cmd/gateway
```

**Test `healthz` (không cần token):**
```powershell
Invoke-WebRequest http://localhost:8080/healthz
# => {"status":"ok"}
```

**Gọi chat** (stream SSE events `start → stream* → final|error`):
```powershell
$token = "dev-gateway-token-00008"
curl.exe -N -X POST http://localhost:8080/v1/chat `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json" `
  -d '{"prompt":"List cac file trong thu muc sandbox"}'
```

Kết quả SSE mẫu:
```
event: start
data: {"model":"","request_id":"req"}

event: stream
data: {"bytes":0,"name":"list_directory","status":"start","type":"tool"}

event: final
data: {"text":"..."}
```

Lỗi trước khi stream trả JSON kèm status: `401` (thiếu token), `400` (body malformed), `422` (prompt trống), `429` (quá `GATEWAY_MAX_CONCURRENT`). Lỗi trong stream trả `event: error`.

## Kiểm thử

```powershell
go test ./...
go vet ./...
```

## Cấu trúc

```
cmd/server/          # Composition root: CLI entry, discovery + mapping, agent loop
cmd/gateway/         # HTTP/SSE gateway entrypoint (FEAT-00008)
internal/agent/      # Agentic loop (LLM <-> tools), neutral types, bounded loop, OnEvent observer
internal/llm/        # LLM interface + Gemini adapter, neutral<->genai convert
internal/mapping/    # Map MCP tool schemas -> Gemini function declarations
internal/mcpclient/  # MCP stdio client (start/init/list/call tool)
internal/config/     # Load & validate .env
internal/policy/     # Filesystem policy (allowed roots)
internal/server/     # SDK-free HTTP layer: auth, SSE, healthz, Handler seam
internal/runner/     # Per-request composition: MCP lifecycle + agent loop + event mapping
sandbox/             # Root để filesystem server thao tác
```

## Lưu ý khi gặp lỗi

- **`401 Unauthorized`**: API key sai hoặc hết hạn — kiểm tra `GEMINI_API_KEY`.
- **`Function call is missing a thought_signature`**: Fix đã được xử lý trong `internal/llm/convert.go` (preserve `ThoughtSignature` khi replay tool call); nếu model mới lại sinh lỗi này hãy kiểm tra SDK `genai`.
- **Model không trả lời**: model cũ/không hỗ trợ tool-calling — đổi `GEMINI_MODEL`.
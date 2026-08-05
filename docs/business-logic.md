# Business Logic - Custom MCP Host

## 1. Overview

Custom MCP Host la mot orchestrator headless ket noi Gemini voi MCP Server. MVP chi ho tro Filesystem MCP qua `stdio` va nhan prompt tu CLI. Host khong tu thuc thi shell command, SQL, hay file operation theo du lieu do LLM cung cap; moi thao tac phai di qua MCP tool da duoc discovery va kiem tra policy.

## 2. Core Workflow

1. User goi CLI voi `--prompt` hoac gui mot prompt qua stdin.
2. Host kiem tra configuration, khoi tao request context, va spawn Filesystem MCP Server da duoc cau hinh.
3. Host hoan thanh initialize handshake va goi `tools/list`.
4. Host map schema MCP tool sang Gemini function declarations.
5. Host gui prompt, conversation history, va tool declarations toi Gemini.
6. Neu Gemini tra ve tool call, Host validate ten tool va arguments, sau do goi `tools/call` qua MCP.
7. Host dua tool result da gioi han kich thuoc, hoac loi an toan, vao conversation history va lap lai buoc 5.
8. Khi Gemini tra text ma khong co tool call, Host in text do ra stdout va dong MCP process.

## 3. Agent State and Limits

| State | Meaning | Exit or transition |
|---|---|---|
| `ConfigValidation` | Validate secrets, command, roots, and limits. | Invalid config returns a non-zero exit code. |
| `MCPConnecting` | Spawn process and initialize MCP. | Success goes to `ToolDiscovery`; error goes to `Failed`. |
| `ToolDiscovery` | Request and map `tools/list`. | Success goes to `LLMInference`; unsupported schema goes to `Failed`. |
| `LLMInference` | Send history and tools to Gemini. | Text goes to `Completed`; tool call goes to `ToolExecution`. |
| `ToolExecution` | Validate and call a discovered MCP tool. | Append result/error, then return to `LLMInference`. |
| `Completed` | Return final text. | Close MCP process. |
| `Failed` | Return a concise user-safe error. | Close MCP process. |

The host enforces the following per prompt:

- A parent context covers the complete invocation.
- Every Gemini request uses `LLM_TIMEOUT`.
- Initialize, discovery, and every MCP call use `MCP_TIMEOUT`.
- The loop cannot exceed `AGENT_MAX_ITERATIONS`.
- Each tool result copied into history cannot exceed `MCP_MAX_RESULT_BYTES`.

## 4. Business Rules

- Only the configured Filesystem MCP executable may be spawned. The LLM cannot choose the executable, process arguments, or working directory.
- Only tools returned by `tools/list` may be called.
- Tool arguments must conform to the discovered schema before they are sent to MCP.
- Filesystem access is restricted to configured canonical roots. A path outside those roots is denied.
- Tool errors, MCP crashes, invalid protocol messages, and timeouts must be turned into safe bounded context for Gemini. The host must remain running and clean up child processes.
- API keys, raw sensitive prompts, authorization data, stack traces, and unbounded tool output must not be logged or fed back to users.
- The MVP handles one MCP server per invocation. Concurrent multi-server execution is deferred until each server has an explicit policy and conflict-resolution design.

## 5. Error Handling Contract

| Failure | Host behavior | Context sent to Gemini |
|---|---|---|
| Invalid configuration | Stop before spawning a process; return non-zero. | None. |
| MCP startup or initialize failure | Close process and return a safe error. | None for initial discovery. |
| `tools/list` failure or unsupported schema | Close process and return a safe error. | None for initial discovery. |
| Unknown or unauthorized tool call | Do not call MCP. | A tool result stating the action is unavailable. |
| Invalid tool arguments | Do not call MCP. | A tool result stating arguments failed validation. |
| MCP tool error, crash, or timeout | Preserve host process; append bounded error result. | A concise error code and retry-safe message. |
| Gemini timeout or provider error | Close process and return a safe error. | None. |
| Iteration limit reached | Close process and return a safe error. | None after the limit is reached. |
| Caller cancellation | Cancel LLM/MCP work, close process, and return cancellation status. | None. |

## 6. Acceptance Criteria

- **AC1 - Filesystem connection:** Given a valid Filesystem MCP command and sandbox root, the host completes MCP initialize within `MCP_TIMEOUT`.
- **AC2 - Discovery and mapping:** The host returns discovered Filesystem tools and maps supported JSON Schema fields, including `required`, nested properties, arrays, and enums, to Gemini declarations.
- **AC3 - Complete agent loop:** Given a Gemini response requesting a discovered Filesystem tool, the host validates and executes the call, adds its result to history, and returns the subsequent final Gemini text.
- **AC4 - Safe failure:** Given an MCP tool error, timeout, or child-process crash, the host exits the invocation cleanly without panic and does not leave a child process running.
- **AC5 - Extensible architecture:** `internal/agent` uses MCP and LLM interfaces only; fake implementations can exercise the complete loop without `mcp-go` or the Gemini SDK.

## 7. Out of Scope for MVP

- HTTP API, SSE streaming, Next.js UI, and authentication for remote callers.
- Multiple concurrent MCP servers.
- PostgreSQL, Git/SVN, Docker/CLI, and remote HTTP/SSE MCP transports.
- DeepSeek and Ollama providers.

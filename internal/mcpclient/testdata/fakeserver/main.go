// Command fakeserver is a tiny stdio JSON-RPC MCP server used only by the
// mcpclient integration tests. It is not part of the host. Behavior is
// selected by argv[1]:
//
//	normal   respond to initialize, tools/list, tools/call
//	crash    exit(1) on the first request
//	timeout  never respond (but keep reading stdin for EOF-sense)
//	error    return a JSON-RPC error for tools/call
//
// argv[2] (if present) is the allowed root directory; on a clean stdin EOF the
// server writes an "exit-marker" file there so tests can prove it terminated
// (i.e. no orphaned child).
package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	modeNormal  = "normal"
	modeCrash   = "crash"
	modeTimeout = "timeout"
	modeError   = "error"

	exitMarker = "exit-marker.txt"
)

func main() {
	mode := modeNormal
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	root := ""
	if len(os.Args) > 2 {
		root = os.Args[2]
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line == "" {
			continue
		}

		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.ID == nil {
			continue // malformed line or notification (e.g. initialized)
		}

		switch mode {
		case modeCrash:
			os.Exit(1)
		case modeTimeout:
			continue // never respond; keep reading stdin for EOF
		}

		switch msg.Method {
		case "initialize":
			respond(msg.ID, map[string]any{
				"protocolVersion": "2025-11-25",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "fake-mcp", "version": "1.0.0"},
			})
		case "tools/list":
			respond(msg.ID, map[string]any{
				"tools": []map[string]any{{
					"name":        "echo",
					"description": "echo back the message",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"message": map[string]any{"type": "string"},
						},
						"required": []string{"message"},
					},
				}},
			})
		case "tools/call":
			if mode == modeError {
				respondError(msg.ID, -32602, "invalid params")
			} else {
				respond(msg.ID, map[string]any{
					"content": []map[string]any{{"type": "text", "text": "ok"}},
				})
			}
		default:
			respondError(msg.ID, -32601, "method not found")
		}
	}

	// Clean stdin EOF: write the exit marker then exit normally. This is how
	// the integration test proves the child terminated rather than leaked.
	if root != "" {
		_ = os.WriteFile(filepath.Join(root, exitMarker), []byte("exited"), 0o600)
	}
}

func respond(id json.RawMessage, result any) {
	writeJSON(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func respondError(id json.RawMessage, code int, message string) {
	writeJSON(map[string]any{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": code, "message": message},
	})
}

func writeJSON(v any) {
	b, _ := json.Marshal(v)
	b = append(b, '\n')
	_, _ = os.Stdout.Write(b)
}

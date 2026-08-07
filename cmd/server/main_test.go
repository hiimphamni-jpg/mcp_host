package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GEMINI_API_KEY", "test-secret")
	t.Setenv("GEMINI_MODEL", "gemini-2.0-flash")
	t.Setenv("MCP_FILESYSTEM_COMMAND", "npx")
	t.Setenv("MCP_FILESYSTEM_ARGS_JSON", `["-y","@modelcontextprotocol/server-filesystem"]`)
	t.Setenv("MCP_ALLOWED_ROOTS", os.TempDir())
	t.Setenv("MCP_TIMEOUT", "30s")
	t.Setenv("LLM_TIMEOUT", "45s")
	t.Setenv("AGENT_MAX_ITERATIONS", "10")
	t.Setenv("MCP_MAX_RESULT_BYTES", "65536")
}

func TestRun_WritesSafeBootstrapStatus(t *testing.T) {
	setValidEnv(t)

	var output bytes.Buffer
	if err := run(&output, strings.NewReader("")); err != nil {
		t.Fatalf("run returned an error: %v", err)
	}

	const want = "MCP Host bootstrap complete. Integrations are not configured or started.\n"
	if output.String() != want {
		t.Fatalf("run output = %q, want %q", output.String(), want)
	}
}

func TestRun_SmokeUnconfiguredInvocationEndsCleanly(t *testing.T) {
	// Smoke: with a valid configuration but no --prompt, the agent loop is not
	// requested, so run() must exit cleanly without touching real Gemini/MCP.
	setValidEnv(t)

	var output bytes.Buffer
	if err := run(&output, strings.NewReader("")); err != nil {
		t.Fatalf("run returned an error: %v", err)
	}
	if !strings.Contains(output.String(), "bootstrap complete") {
		t.Fatalf("run output = %q, want bootstrap status", output.String())
	}
}

func TestRun_InvalidConfigReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("GEMINI_API_KEY", "")

	var output bytes.Buffer
	if err := run(&output, strings.NewReader("")); err == nil {
		t.Fatal("run succeeded with invalid configuration")
	}
	if strings.Contains(output.String(), "bootstrap complete") {
		t.Error("run wrote bootstrap status despite invalid configuration")
	}
}

func TestRun_InvalidConfigDoesNotLeakSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("GEMINI_API_KEY", "topsecret-xyz")
	t.Setenv("MCP_FILESYSTEM_COMMAND", "")

	var output bytes.Buffer
	err := run(&output, strings.NewReader(""))
	if err == nil {
		t.Fatal("run succeeded with invalid configuration")
	}
	if strings.Contains(err.Error(), "topsecret-xyz") {
		t.Errorf("run error leaked the secret: %v", err)
	}
	if strings.Contains(output.String(), "topsecret-xyz") {
		t.Errorf("run output leaked the secret: %q", output.String())
	}
}

// errReader fails on the first Read call.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read stdin: boom") }

func TestReadPromptFromStdin_NonEmpty_ReturnsTrimmedPrompt(t *testing.T) {
	got, err := readPromptFromStdin(strings.NewReader("  list files\n"))
	if err != nil {
		t.Fatalf("readPromptFromStdin: %v", err)
	}
	if got != "list files" {
		t.Fatalf("readPromptFromStdin = %q, want %q", got, "list files")
	}
}

func TestReadPromptFromStdin_Empty_ReturnsEmptyString(t *testing.T) {
	got, err := readPromptFromStdin(strings.NewReader(""))
	if err != nil {
		t.Fatalf("readPromptFromStdin: %v", err)
	}
	if got != "" {
		t.Fatalf("readPromptFromStdin = %q, want empty", got)
	}
}

func TestReadPromptFromStdin_ReadError_IsSurfaced(t *testing.T) {
	_, err := readPromptFromStdin(errReader{})
	if err == nil {
		t.Fatal("readPromptFromStdin swallowed the read error")
	}
}

func TestRun_EmptyStdin_Bootstraps(t *testing.T) {
	setValidEnv(t)
	var output bytes.Buffer
	if err := run(&output, strings.NewReader("")); err != nil {
		t.Fatalf("run returned an error: %v", err)
	}
	if !strings.Contains(output.String(), "bootstrap complete") {
		t.Fatalf("run output = %q, want bootstrap status for empty stdin", output.String())
	}
}

func TestRun_NonEmptyStdin_TriggersLoop(t *testing.T) {
	setValidEnv(t)
	// A piped non-empty stdin must take the agent-loop path, which first spawns the
	// MCP process first. Point the policy at an unresolvable command so the loop fails
	// fast and deterministically with an MCP start error proving the loop path was taken without
	// touching real Gemini/MCP/npx.
	t.Setenv("MCP_FILESYSTEM_COMMAND", "nonexistent-cmd-does-not-exist")

	var output bytes.Buffer
	err := run(&output, strings.NewReader("list files"))
	if err == nil {
		t.Fatal("run succeeded with a piped prompt but no Gemini/MCP configured")
	}
	if !strings.Contains(err.Error(), "start MCP client") {
		t.Fatalf("run error = %v, want an MCP start failure proving the loop path was taken", err)
	}
	if strings.Contains(output.String(), "bootstrap complete") {
		t.Error("run wrote bootstrap status despite a piped prompt")
	}
}

func TestRun_FlagParseErrorReturnsCleanError(t *testing.T) {
	setValidEnv(t)

	var output bytes.Buffer
	if err := run(&output, strings.NewReader(""), "--unknown-flag"); err == nil {
		t.Fatal("run succeeded with an unknown flag")
	}
}

// fakeClient is a scripted mcpclient.Client for adapter unit tests. It never
// touches the network or a subprocess.
type fakeClient struct {
	result *mcp.CallToolResult
	err    error
	got    string
	args   map[string]any
	ctx    context.Context
}

func (f *fakeClient) Initialize(ctx context.Context) error { return nil }
func (f *fakeClient) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}
func (f *fakeClient) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	f.got, f.args, f.ctx = name, args, ctx
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}
func (f *fakeClient) Close() error { return nil }

func TestMCPToolCaller_ConvertsTextContent(t *testing.T) {
	fc := &fakeClient{result: &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "hello world"}},
	}}
	tr, err := (&mcpToolCaller{client: fc}).CallTool(context.Background(), "read_file", map[string]any{"path": "a"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if tr.Name != "read_file" {
		t.Errorf("name = %q, want read_file", tr.Name)
	}
	content, _ := tr.Response["content"].(string)
	if content != "hello world" {
		t.Errorf("content = %q, want the text payload", content)
	}
	if _, ok := tr.Response["error"]; ok {
		t.Error("non-error result must not carry the error flag")
	}
}

func TestMCPToolCaller_ConvertsEmbeddedResource(t *testing.T) {
	fc := &fakeClient{result: &mcp.CallToolResult{
		Content: []mcp.Content{mcp.EmbeddedResource{
			Resource: mcp.TextResourceContents{URI: "file:///x", Text: "embedded text"},
		}},
	}}
	tr, err := (&mcpToolCaller{client: fc}).CallTool(context.Background(), "list_dir", nil)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	content, _ := tr.Response["content"].(string)
	if content != "embedded text" {
		t.Errorf("content = %q, want the embedded text", content)
	}
}

func TestMCPToolCaller_MarksIsError(t *testing.T) {
	fc := &fakeClient{result: &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Text: "boom"}},
		IsError: true,
	}}
	tr, err := (&mcpToolCaller{client: fc}).CallTool(context.Background(), "write_file", nil)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	flag, _ := tr.Response["error"].(bool)
	if !flag {
		t.Error("isError result did not carry the error flag")
	}
}

func TestMCPToolCaller_PropagatesErrorAndContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fc := &fakeClient{err: errors.New("boom")}
	_, err := (&mcpToolCaller{client: fc}).CallTool(ctx, "read_file", nil)
	if err == nil {
		t.Fatal("CallTool swallowed the underlying error")
	}
	if fc.ctx == nil || fc.ctx.Err() == nil {
		t.Error("CallTool did not forward the caller context")
	}
}

func TestNeutralSchema_RoundTrip(t *testing.T) {
	tool := mcp.Tool{
		Name: "read_file",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{"type": "string"},
			},
			Required: []string{"path"},
		},
	}
	schema, err := neutralSchema(tool)
	if err != nil {
		t.Fatalf("neutralSchema: %v", err)
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["path"]; !ok {
		t.Errorf("schema %v lost the path property", schema)
	}
	if typ, _ := schema["type"].(string); typ != "object" {
		t.Errorf("schema type = %q, want object", typ)
	}
}

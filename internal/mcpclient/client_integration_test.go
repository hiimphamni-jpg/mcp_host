package mcpclient_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mcp-host/internal/mcpclient"
	"mcp-host/internal/policy"

	"github.com/mark3labs/mcp-go/mcp"
)

var fakeServerBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "mcp-fakeserver-")
	if err != nil {
		os.Exit(1)
	}
	name := "mcp-fake-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	fakeServerBin = filepath.Join(tmp, name)

	cmd := exec.Command("go", "build", "-o", fakeServerBin, "./testdata/fakeserver")
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Stderr.Write([]byte("failed to build fake server: " + err.Error() + "\n" + string(out)))
		_ = os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// fakePolicy builds a FilesystemPolicy that launches the fake stdio server with
// a bare command name (satisfying policy command validation) and resolves it
// via PATH. The fake server binary lives in a temp dir we prepend to PATH.
func fakePolicy(t *testing.T, mode string) *policy.FilesystemPolicy {
	t.Helper()

	binDir := filepath.Dir(fakeServerBin)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	p, err := policy.New(filepath.Base(fakeServerBin), []string{mode}, []string{t.TempDir()})
	if err != nil {
		t.Fatalf("policy.New: %v", err)
	}
	return p
}

func newFakeClient(t *testing.T, mode string, timeout time.Duration) (*mcpclient.StdioClient, *policy.FilesystemPolicy) {
	t.Helper()
	p := fakePolicy(t, mode)
	c, err := mcpclient.NewStdio(p, timeout)
	if err != nil {
		t.Fatalf("NewStdio: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, p
}

func newFakeClientAndRoot(t *testing.T, mode string, timeout time.Duration) (*mcpclient.StdioClient, string) {
	t.Helper()
	p := fakePolicy(t, mode)
	root := p.Roots()[0]
	c, err := mcpclient.NewStdio(p, timeout)
	if err != nil {
		t.Fatalf("NewStdio: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, root
}

func TestStdioClient_HappyPath_InitializeListCall(t *testing.T) {
	c, _ := newFakeClient(t, "normal", 5*time.Second)
	ctx := context.Background()

	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	result, err := c.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("ListTools tools = %d, want 1", len(result.Tools))
	}
	if result.Tools[0].Name != "echo" {
		t.Errorf("ListTools tool name = %q, want echo", result.Tools[0].Name)
	}

	call, err := c.CallTool(ctx, "echo", map[string]any{"message": "hi"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if call.IsError {
		t.Error("CallTool IsError = true, want false")
	}
	if len(call.Content) == 0 {
		t.Fatal("CallTool returned no content")
	}
	if tc, ok := call.Content[0].(mcp.TextContent); ok {
		if tc.Text != "ok" {
			t.Errorf("CallTool text = %q, want ok", tc.Text)
		}
	}
}

func TestStdioClient_InvalidPolicyAndTimeoutArg(t *testing.T) {
	if c, err := mcpclient.NewStdio(nil, time.Second); err == nil {
		t.Error("NewStdio with nil policy succeeded")
		_ = c
	}
	p := fakePolicy(t, "normal")
	if c, err := mcpclient.NewStdio(p, 0); err == nil {
		t.Error("NewStdio with non-positive timeout succeeded")
		_ = c
	}
}

func TestStdioClient_Crash_ReturnsTypedErrorAndCloseIsSafe(t *testing.T) {
	c, _ := newFakeClient(t, "crash", 5*time.Second)

	err := c.Initialize(context.Background())
	if err == nil {
		t.Fatal("Initialize succeeded against a crashing child")
	}
	if !errors.Is(err, mcpclient.ErrProcessExit) {
		t.Errorf("errors.Is(err, ErrProcessExit) = false for %v", err)
	}

	if cerr := c.Close(); cerr != nil && !errors.Is(cerr, mcpclient.ErrProcessExit) {
		t.Logf("first Close returned: %v (acceptable)", cerr)
	}
	if cerr := c.Close(); cerr != nil {
		t.Errorf("second Close returned error: %v (want nil, idempotent)", cerr)
	}
}

func TestStdioClient_Timeout_ReturnsErrTimeout(t *testing.T) {
	c, _ := newFakeClient(t, "timeout", 200*time.Millisecond)

	err := c.Initialize(context.Background())
	if err == nil {
		t.Fatal("Initialize succeeded against a never-responding child")
	}
	if !errors.Is(err, mcpclient.ErrTimeout) {
		t.Errorf("errors.Is(err, ErrTimeout) = %v, want true (got %v)", errors.Is(err, mcpclient.ErrTimeout), err)
	}
	// Host remains usable after a timeout: error is typed and non-fatal.
	if cerr := c.Close(); cerr != nil {
		t.Errorf("Close after timeout returned error: %v", cerr)
	}
}

func TestStdioServer_InvalidResponse_ReturnsErrInvalidResponse(t *testing.T) {
	c, _ := newFakeClient(t, "error", 5*time.Second)
	ctx := context.Background()

	if err := c.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	_, err := c.CallTool(ctx, "echo", map[string]any{})
	if err == nil {
		t.Fatal("CallTool succeeded despite server protocol error")
	}
	if !errors.Is(err, mcpclient.ErrInvalidResponse) {
		t.Errorf("errors.Is(err, ErrInvalidResponse) = %v, want true (got %v)", errors.Is(err, mcpclient.ErrInvalidResponse), err)
	}
}

func TestStdioServer_Close_LeavesNoOrphan(t *testing.T) {
	c, root := newFakeClientAndRoot(t, "timeout", time.Second)

	if err := c.Initialize(context.Background()); !errors.Is(err, mcpclient.ErrTimeout) {
		t.Fatalf("Initialize: %v (want ErrTimeout)", err)
	}

	// Close closes stdin, which delivers EOF to the fake server. It must write
	// the exit marker (proving the child terminated) rather than be orphaned.
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	marker := filepath.Join(root, "exit-marker.txt")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return // child terminated cleanly; no orphan
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("exit-marker was not written; the child was orphaned or not terminated")
}

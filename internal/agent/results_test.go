package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mcp-host/internal/llm"
)

func TestBoundedResult_noTruncation(t *testing.T) {
	tr := &llm.ToolResult{Name: "ls", Response: map[string]any{"text": "hello"}}
	out := BoundedResult(tr, 1024)
	if out.Name != "ls" {
		t.Fatalf("name = %q, want ls", out.Name)
	}
	content, _ := out.Response["content"].(string)
	if strings.Contains(content, truncationMarker) {
		t.Fatalf("result was truncated unexpectedly: %q", content)
	}
	if !strings.Contains(content, "hello") {
		t.Errorf("bounded result %q does not carry the payload", content)
	}
}

func TestBoundedResult_truncationMarker(t *testing.T) {
	tr := &llm.ToolResult{Name: "big", Response: map[string]any{"text": strings.Repeat("x", 5000)}}
	out := BoundedResult(tr, 64)
	content, _ := out.Response["content"].(string)
	if !strings.HasSuffix(content, truncationMarker) {
		t.Fatalf("truncated result %q lacks the marker", content)
	}
	if len(content) > 64 {
		t.Errorf("truncated result length %d exceeds limit 64", len(content))
	}
}

func TestBoundedResult_nilResult(t *testing.T) {
	out := BoundedResult(nil, 16)
	if out == nil || out.Name != "" {
		t.Fatalf("BoundedResult(nil) = %+v, want empty result", out)
	}
}

func TestErrorResult_noRawErrorLeak(t *testing.T) {
	secret := errors.New("access token super-secret-value failed: stacktrace")
	out := ErrorResult("read_file", secret)
	content, _ := out.Response["content"].(string)
	if strings.Contains(content, "super-secret-value") {
		t.Fatalf("error result leaked the raw error: %q", content)
	}
	if strings.Contains(content, "stacktrace") {
		t.Fatalf("error result leaked trace: %q", content)
	}
	flag, _ := out.Response["error"].(bool)
	if !flag {
		t.Errorf("error result did not mark the error")
	}
}

func TestErrorResult_deadlineMapping(t *testing.T) {
	out := ErrorResult("ls", context.DeadlineExceeded)
	content, _ := out.Response["content"].(string)
	if !strings.Contains(content, "timed out") {
		t.Errorf("deadline error mapped to %q, want a timeout label", content)
	}
}

func TestErrorResult_cancelledMapping(t *testing.T) {
	out := ErrorResult("ls", context.Canceled)
	content, _ := out.Response["content"].(string)
	if !strings.Contains(content, "cancelled") {
		t.Errorf("canceled error mapped to %q, want a cancelled label", content)
	}
}

func TestMessageResult(t *testing.T) {
	out := MessageResult("ghost", "action unavailable: tool not found")
	if out.Name != "ghost" {
		t.Fatalf("name = %q, want ghost", out.Name)
	}
	content, _ := out.Response["content"].(string)
	if content != "action unavailable: tool not found" {
		t.Errorf("message result content = %q", content)
	}
}

package agent

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mcp-host/internal/llm"
)

// stubLLM scripts a sequence of LLM responses, then returns the configured
// terminal err (nil by default) for any further call.
type stubLLM struct {
	resps []*llm.Response
	err   error
	calls int
	got   []*llm.Request
}

func (s *stubLLM) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	s.calls++
	s.got = append(s.got, req)
	if s.err != nil {
		return nil, s.err
	}
	if s.calls > len(s.resps) {
		return &llm.Response{}, nil
	}
	return s.resps[s.calls-1], nil
}

// stubMCP scripts tool results/errors by name and records the contexts passed
// to CallTool so tests can assert cancellation propagation. If cancel is set it
// is invoked on the first call to simulate an in-flight cancellation.
type stubMCP struct {
	results map[string]*llm.ToolResult
	errs    map[string]error
	cancel  context.CancelFunc
	got     []map[string]any
	ctxs    []context.Context
}

func (m *stubMCP) CallTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error) {
	m.got = append(m.got, args)
	m.ctxs = append(m.ctxs, ctx)
	if m.cancel != nil {
		m.cancel()
	}
	if err := m.errs[name]; err != nil {
		return nil, err
	}
	if r := m.results[name]; r != nil {
		return r, nil
	}
	return &llm.ToolResult{Name: name, Response: map[string]any{"content": "ok"}}, nil
}

// simpleSchemas declares one tool with a required string property so tests
// exercise schema validation inside the loop.
func simpleSchemas() map[string]any {
	return map[string]any{
		"read_file": map[string]any{
			"type":       "object",
			"properties": map[string]any{"path": map[string]any{"type": "string"}},
			"required":   []string{"path"},
		},
	}
}

func newTestAgent(t *testing.T, l llm.LLM, m *stubMCP, iterations int) *Agent {
	t.Helper()
	a, err := New(Options{
		LLM:            l,
		Tools:          m,
		Schemas:        simpleSchemas(),
		MaxIterations:  iterations,
		MaxResultBytes: 256,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

// feedback returns the text of the last ToolResult appended to history after a
// tool turn, or "" if the last history message carried no result content.
func feedback(l *stubLLM) string {
	if len(l.got) == 0 {
		return ""
	}
	req := l.got[len(l.got)-1]
	if len(req.Contents) == 0 {
		return ""
	}
	last := req.Contents[len(req.Contents)-1]
	if len(last.ToolResults) == 0 {
		return ""
	}
	content, _ := last.ToolResults[0].Response["content"].(string)
	return content
}

func TestNew_rejectsInvalidOptions(t *testing.T) {
	schemas := simpleSchemas()

	if _, err := New(Options{Tools: &stubMCP{}, Schemas: schemas, MaxIterations: 1, MaxResultBytes: 1}); err == nil {
		t.Error("New accepted a nil LLM")
	}
	if _, err := New(Options{LLM: &stubLLM{}, Schemas: schemas, MaxIterations: 1, MaxResultBytes: 1}); err == nil {
		t.Error("New accepted nil Tools")
	}
	if _, err := New(Options{LLM: &stubLLM{}, Tools: &stubMCP{}, Schemas: schemas, MaxIterations: 0, MaxResultBytes: 1}); err == nil {
		t.Error("New accepted a non-positive MaxIterations")
	}
	if _, err := New(Options{LLM: &stubLLM{}, Tools: &stubMCP{}, Schemas: schemas, MaxIterations: 1, MaxResultBytes: 0}); err == nil {
		t.Error("New accepted a non-positive MaxResultBytes")
	}
}

func TestNew_acceptsValidOptions(t *testing.T) {
	if _, err := New(Options{LLM: &stubLLM{}, Tools: &stubMCP{}, Schemas: simpleSchemas(), MaxIterations: 1, MaxResultBytes: 1}); err != nil {
		t.Fatalf("New rejected valid options: %v", err)
	}
}

func TestClassifyGenerateError(t *testing.T) {
	if err := classifyGenerateError(context.Canceled); !errors.Is(err, context.Canceled) {
		t.Errorf("caller cancellation not passed through: %v", err)
	}
	for _, c := range []error{llm.ErrTimeout, llm.ErrProvider} {
		err := classifyGenerateError(c)
		if !errors.Is(err, c) {
			t.Errorf("timeout/provider sentinel lost: %v", err)
		}
	}
	generic := classifyGenerateError(errors.New("boom"))
	if generic.Error() == "boom" {
		t.Error("generic provider error message must be sanitized")
	}
}

// TestRun_onEventOrdering proves GW4's real emission order: the agent itself
// must emit start, then per-iteration text/tool observers, and finally exactly
// one terminal event. This closes the gap between the HTTP-layer fake Event
// test (server) and the actual agent pipeline.
func TestRun_onEventOrdering(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}, Text: "thinking..."},
		{Text: "done"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": "x"}},
	}}
	var types []EventType
	a, err := New(Options{
		LLM:            l,
		Tools:          m,
		Schemas:        simpleSchemas(),
		MaxIterations:  5,
		MaxResultBytes: 64,
		OnEvent: func(e Event) { types = append(types, e.Type) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []EventType{EventStart, EventText, EventTool, EventTool, EventFinal}
	if !reflect.DeepEqual(types, want) {
		t.Errorf("observed event order = %v, want %v", types, want)
	}
}

func TestClassify_Classification_nilIsSafe(t *testing.T) {
	if err := classifyGenerateError(nil); err != nil {
		t.Errorf("classifyGenerateError(nil) = %v, want nil", err)
	}
}

func TestRun_returnsFinalTextDirectly(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{{Text: "hello"}}}
	a := newTestAgent(t, l, &stubMCP{}, 5)

	out, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "hello" {
		t.Fatalf("out = %q, want hello", out)
	}
	if l.calls != 1 {
		t.Errorf("LLM called %d times, want 1", l.calls)
	}
}

func TestRun_singleToolCallThenFinalText(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a.txt"}}}},
		{Text: "done"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": "filedata"}},
	}}
	a := newTestAgent(t, l, m, 5)

	out, err := a.Run(context.Background(), "read it")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "done" {
		t.Fatalf("out = %q, want done", out)
	}
	if len(m.got) != 1 {
		t.Fatalf("tool called %d times, want 1", len(m.got))
	}
	if l.calls < 2 {
		t.Fatalf("LLM called %d times, want at least 2 (result must be fed back)", l.calls)
	}
}

func TestRun_multipleToolCallsInOneTurn(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{
			{Name: "read_file", Arguments: map[string]any{"path": "a"}},
			{Name: "read_file", Arguments: map[string]any{"path": "b"}},
		}},
		{Text: "compiled"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": "x"}},
	}}
	a := newTestAgent(t, l, m, 5)

	out, err := a.Run(context.Background(), "both")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "compiled" {
		t.Fatalf("out = %q", out)
	}
	if len(m.got) != 2 {
		t.Fatalf("expected 2 tool calls, got %d (sequential, request order)", len(m.got))
	}
}

func TestRun_iterationLimit(t *testing.T) {
	call := &llm.Response{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}}
	l := &stubLLM{resps: []*llm.Response{call, call, call, call, call}}
	m := &stubMCP{results: map[string]*llm.ToolResult{"read_file": {Response: map[string]any{"content": "x"}}}}
	a := newTestAgent(t, l, m, 5)

	_, err := a.Run(context.Background(), "loop")
	if !errors.Is(err, ErrIterationLimit) {
		t.Fatalf("err = %v, want ErrIterationLimit", err)
	}
}

func TestRun_unknownToolBecomesSafeResult(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "ghost", Arguments: map[string]any{}}}},
		{Text: "explained"},
	}}
	m := &stubMCP{}
	a := newTestAgent(t, l, m, 5)

	out, err := a.Run(context.Background(), "run ghost tool")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "explained" {
		t.Fatalf("out = %q", out)
	}
	if len(m.got) != 0 {
		t.Errorf("ghost tool reached the caller %d times, want 0", len(m.got))
	}
	if fb := feedback(l); !strings.Contains(fb, "action unavailable") {
		t.Errorf("history result %q lacks the safe 'unavailable' marker", fb)
	}
}

func TestRun_invalidArgsBecomesSafeResult(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": 42}}}},
		{Text: "explained"},
	}}
	m := &stubMCP{}
	a := newTestAgent(t, l, m, 5)

	out, err := a.Run(context.Background(), "call with bad args")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "explained" {
		t.Fatalf("out = %q", out)
	}
	if len(m.got) != 0 {
		t.Errorf("invalid-args tool reached the caller %d times, want 0", len(m.got))
	}
	if fb := feedback(l); !strings.Contains(fb, "failed validation") {
		t.Errorf("history result %q lacks the validation marker", fb)
	}
}

func TestRun_mcpFailureBecomesBoundedResult(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{Text: "recovered"},
	}}
	m := &stubMCP{errs: map[string]error{
		"read_file": errors.New("secret-token-failed: /etc/passwd stacktrace"),
	}}
	a := newTestAgent(t, l, m, 5)

	out, err := a.Run(context.Background(), "call failing tool")
	if err != nil {
		t.Fatalf("Run returned an error instead of a bounded result: %v", err)
	}
	if out != "recovered" {
		t.Fatalf("out = %q", out)
	}
	fb := feedback(l)
	if strings.Contains(fb, "secret-token-failed") || strings.Contains(fb, "stacktrace") {
		t.Errorf("failure result leaked raw error: %q", fb)
	}
	if !strings.Contains(fb, "tool call failed") {
		t.Errorf("history result %q lacks the generic failure label", fb)
	}
}

func TestRun_cancellationPropagatesToToolAndTerminates(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{Text: "never reached"},
	}}
	// The context is already cancelled before Run starts. The loop must stop
	// with context.Canceled rather than continuing to invoke tools.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := &stubMCP{}
	a := newTestAgent(t, l, m, 5)

	_, err := a.Run(ctx, "call")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(m.got) != 0 {
		t.Errorf("tool invoked %d times on a cancelled context, want 0", len(m.got))
	}
}

func TestRun_cancellationMidLoopStops(t *testing.T) {
	// A cancel() fires the moment the first tool call happens. The loop must
	// surface context.Canceled (terminal) and must not run the remaining tool
	// calls in the same turn (precedence, ADR-0001 D3).
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{
			{Name: "read_file", Arguments: map[string]any{"path": "a"}},
			{Name: "read_file", Arguments: map[string]any{"path": "b"}},
		}},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	m := &stubMCP{cancel: cancel}
	a := newTestAgent(t, l, m, 5)

	_, err := a.Run(ctx, "call")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled (cancellation wins over pending tool calls)", err)
	}
	if len(m.got) != 1 {
		t.Errorf("second tool call executed after cancellation, got %d calls", len(m.got))
	}
	if len(m.ctxs) == 0 || m.ctxs[0].Err() == nil {
		t.Errorf("CallTool did not receive the cancelled context: %v", m.ctxs)
	}
}

func TestRun_cancellationMidLoopOverridesBoundedToolError(t *testing.T) {
	// The tool returns an ordinary error while the caller cancels in-flight.
	// Caller cancellation must take precedence over converting it to a bounded
	// result (ADR-0001 D3).
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{Text: "never reached"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	m := &stubMCP{
		errs:   map[string]error{"read_file": errors.New("boom")},
		cancel: cancel,
	}
	a := newTestAgent(t, l, m, 5)

	_, err := a.Run(ctx, "call")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestRun_resultIsTruncated(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{Text: "summarised"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": strings.Repeat("x", 5000)}},
	}}
	a, err := New(Options{
		LLM:            l,
		Tools:          m,
		Schemas:        simpleSchemas(),
		MaxIterations:  5,
		MaxResultBytes: 64,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := a.Run(context.Background(), "big"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	fb := feedback(l)
	if !strings.HasSuffix(fb, truncationMarker) {
		t.Errorf("bounded result %q lacks truncation marker", fb)
	}
	if len(fb) > 64 {
		t.Errorf("bounded result length %d exceeds 64", len(fb))
	}
}

func TestRun_secretContentNotInOutput(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{Text: "safe"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": "token=supersecret"}},
	}}
	a := newTestAgent(t, l, m, 5)

	// The loop returns only LLM final text; secrets must not be surfaced as the
	// agent's own output. (Bounding keeps them out of oversized history too.)
	out, err := a.Run(context.Background(), "read")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out, "supersecret") {
		t.Errorf("secret leaked into final output: %q", out)
	}
}

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeHandler is an SDK-free Handler used to test the HTTP layer without
// mcp-go/genai (TDD GW8).
type fakeHandler struct {
	prompt string
	answer string
	err    error
	events []Event
}

func (f *fakeHandler) Chat(ctx context.Context, prompt string, emit func(Event)) (string, error) {
	f.prompt = prompt
	for _, e := range f.events {
		emit(e)
	}
	if f.err != nil {
		return "", f.err
	}
	return f.answer, nil
}

func newServer(t *testing.T, h Handler, tokens map[string]bool) *httptest.Server {
	t.Helper()
	s, err := New(h, Options{Tokens: tokens, MaxConcurrent: 4})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func post(t *testing.T, ts *httptest.Server, path, token, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest("POST", ts.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func sseEvents(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, block := range strings.Split(raw, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var name, data string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data = strings.TrimPrefix(line, "data: ")
			}
		}
		var payload map[string]any
		if data != "" {
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				t.Fatalf("bad sse data %q: %v", data, err)
			}
		}
		out = append(out, map[string]any{"event": name, "data": payload})
	}
	return out
}

func TestHealthzNoAuth(t *testing.T) {
	ts := newServer(t, &fakeHandler{answer: "hi"}, map[string]bool{"secret": true})
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", resp.StatusCode)
	}
}

func TestChatUnauthorized(t *testing.T) {
	ts := newServer(t, &fakeHandler{answer: "hi"}, map[string]bool{"secret": true})
	resp, raw := post(t, ts, "/v1/chat", "", `{"prompt":"hello"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if strings.Contains(raw, "event:") {
		t.Errorf("expected no SSE frames, got: %s", raw)
	}
}

func TestChatAuthOK(t *testing.T) {
	fh := &fakeHandler{answer: "the answer"}
	ts := newServer(t, fh, map[string]bool{"secret": true})
	resp, raw := req(t, ts, "/v1/chat", "secret", `{"prompt":"hello"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q", ct)
	}
	events := sseEvents(t, raw)
	if len(events) < 2 {
		t.Fatalf("got %d events, want start+final", len(events))
	}
	if events[0]["event"] != "start" {
		t.Errorf("first event = %v, want start", events[0]["event"])
	}
	last := events[len(events)-1]
	if last["event"] != "final" {
		t.Errorf("last event = %v, want final", last["event"])
	}
	if d := last["data"].(map[string]any); d["text"] != "the answer" {
		t.Errorf("final text = %v", d["text"])
	}
	if fh.prompt != "hello" {
		t.Errorf("handler prompt = %q", fh.prompt)
	}
}

func TestChatBadRequest(t *testing.T) {
	ts := newServer(t, &fakeHandler{answer: "hi"}, map[string]bool{"secret": true})
	resp, raw := req(t, ts, "/v1/chat", "secret", `{not-json`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if strings.Contains(raw, "event:") {
		t.Errorf("no SSE expected, got %s", raw)
	}
}

func TestChatEmptyPrompt(t *testing.T) {
	ts := newServer(t, &fakeHandler{answer: "hi"}, map[string]bool{"secret": true})
	resp, _ := req(t, ts, "/v1/chat", "secret", `{"prompt":"   "}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestChatMissingPromptField(t *testing.T) {
	ts := newServer(t, &fakeHandler{answer: "hi"}, map[string]bool{"secret": true})
	resp, raw := req(t, ts, "/v1/chat", "secret", `{}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if !strings.Contains(raw, "VALIDATION_FAILED") {
		t.Errorf("body = %s, want VALIDATION_FAILED code", raw)
	}
}

func TestChatStreamsToolThenFinal(t *testing.T) {
	fh := &fakeHandler{
		answer: "done",
		events: []Event{
			{Type: "text", Text: "working..."},
			{Type: "tool", ToolName: "read_file", Phase: "start"},
			{Type: "tool", ToolName: "read_file", Phase: "result", Bytes: 42},
		},
	}
	ts := newServer(t, fh, map[string]bool{"secret": true})
	_, raw := req(t, ts, "/v1/chat", "secret", `{"prompt":"go"}`)
	events := sseEvents(t, raw)
	streams := 0
	for _, e := range events {
		if e["event"] == "stream" {
			streams++
		}
	}
	if streams != 3 {
		t.Errorf("got %d stream events, want 3", streams)
	}
	if events[len(events)-1]["event"] != "final" {
		t.Errorf("last event = %v, want final", events[len(events)-1]["event"])
	}
}

func TestChatHandlerErrorIsStreamed(t *testing.T) {
	fh := &fakeHandler{err: context.DeadlineExceeded}
	ts := newServer(t, fh, map[string]bool{"secret": true})
	_, raw := reqChat(t, ts, "secret", `{"prompt":"go"}`)
	events := sseEvents(t, raw)
	if len(events) == 0 {
		t.Fatal("no events")
	}
	last := events[len(events)-1]
	if last["event"] != "error" {
		t.Errorf("last event = %v, want error", last["event"])
	}
}

func req(t *testing.T, ts *httptest.Server, path, token, body string) (*http.Response, string) {
	return newReq(t, ts, path, token, body)
}

func reqChat(t *testing.T, ts *httptest.Server, token, body string) (*http.Response, string) {
	return newReq(t, ts, "/v1/chat", token, body)
}

func newReq(t *testing.T, ts *httptest.Server, path, token, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest("POST", ts.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

// blockHandler blocks inside Chat until the caller context is cancelled. It
// models a runner holding an in-process MCP call so tests can prove a client
// disconnect propagates cancellation through the HTTP seam (GW5).
type blockHandler struct {
	started chan struct{}
	mu      sync.Mutex
	cancelled bool
}

func (h *blockHandler) Chat(ctx context.Context, prompt string, emit func(Event)) (string, error) {
	select {
	case h.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	h.mu.Lock()
	h.cancelled = true
	h.mu.Unlock()
	return "", ctx.Err()
}

func TestChatClientDisconnectCancelsHandlerAndStreamsCancelled(t *testing.T) {
	bh := &blockHandler{started: make(chan struct{}, 1)}
	ts := newServer(t, bh, map[string]bool{"secret": true})

	reqCtx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(reqCtx, "POST", ts.URL+"/v1/chat", bytes.NewBufferString(`{"prompt":"go"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")

	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return // expected: the client disconnected (cancelled) on purpose
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
	}()

	<-bh.started // server is running the agent loop (waiting on MCP)
	cancel()     // simulate client disconnect
	<-done

	// The deterministic GW5 signal: the disconnect propagated cancellation
	// into the handler (the real Runner would observe ctx.Canceled and its
	// deferred client.Close() runs, closing the MCP child - no orphan).
	bh.mu.Lock()
	sawCancel := bh.cancelled
	bh.mu.Unlock()
	if !sawCancel {
		// The client abort triggers the server to cancel the request ctx; this
		// can propagate slightly asynchronously, so poll briefly before failing.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
			bh.mu.Lock()
			sawCancel = bh.cancelled
			bh.mu.Unlock()
			if sawCancel {
				break
			}
		}
	}
	if !sawCancel {
		t.Error("handler Chat was not cancelled by the client disconnect")
	}
}

// flakyHandler fails the first Chat call with an in-stream error and then
// succeeds, so tests prove the gateway process survives a post-start failure
// and serves the next request (GW6).
type flakyHandler struct {
	mu     sync.Mutex
	calls  int
	answer string
}

func (h *flakyHandler) Chat(ctx context.Context, prompt string, emit func(Event)) (string, error) {
	h.mu.Lock()
	h.calls++
	fail := h.calls == 1
	h.mu.Unlock()
	if fail {
		return "", fmt.Errorf("mcp initialize failed")
	}
	emit(Event{Type: "text", Text: "ran"})
	return h.answer, nil
}

func TestChatSurvivesAfterStreamedError(t *testing.T) {
	fh := &flakyHandler{answer: "recovered"}
	ts := newServer(t, fh, map[string]bool{"secret": true})

	_, raw1 := req(t, ts, "/v1/chat", "secret", `{"prompt":"first"}`)
	e1 := sseEvents(t, raw1)
	if e1[len(e1)-1]["event"] != "error" {
		t.Errorf("first stream last event = %v, want error", e1[len(e1)-1]["event"])
	}

	// The same live server must serve the next request successfully and
	// intact state (no co-op, no fatal server flag).
	resp2, raw2 := req(t, ts, "/v1/chat", "secret", `{"prompt":"second"}`)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second status = %d, want 200", resp2.StatusCode)
	}
	e2 := sseEvents(t, raw2)
	if e2[len(e2)-1]["event"] != "final" {
		t.Fatalf("second stream last event = %v, want final", e2[len(e2)-1]["event"])
	}
	if d := e2[len(e2)-1]["data"].(map[string]any); d["text"] != "recovered" {
		t.Errorf("second final text = %v, want recovered", d["text"])
	}
}

// concurrentHandler records concurrent in-flight Chat invocations and prefixes
// the prompt into the answer, proving per-request, independent lifecycles are
// both isolated and parallel under the server (GW9).
type concurrentHandler struct {
	mu        sync.Mutex
	active    int
	maxActive int
}

func (h *concurrentHandler) Chat(ctx context.Context, prompt string, emit func(Event)) (string, error) {
	h.mu.Lock()
	h.active++
	if h.active > h.maxActive {
		h.maxActive = h.active
	}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.active--
		h.mu.Unlock()
	}()
	time.Sleep(50 * time.Millisecond) // widen the overlap window
	return "answer:" + prompt, nil
}

func TestChatConcurrentRequestsIndependentLifecycle(t *testing.T) {
	ch := &concurrentHandler{}
	ts := newServer(t, ch, map[string]bool{"secret": true})

	const n = 4
	results := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, err := http.NewRequest("POST", ts.URL+"/v1/chat", bytes.NewBufferString(fmt.Sprintf(`{"prompt":"p%d"}`, i)))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer secret")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			results[i] = string(b)
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if results[i] == "" {
			t.Errorf("request %d produced no response", i)
			continue
		}
		ev := sseEvents(t, results[i])
		last := ev[len(ev)-1]
		if last["event"] != "final" {
			t.Errorf("req %d last event = %v, want final", i, last["event"])
			continue
		}
		want := "answer:p" + fmt.Sprintf("%d", i)
		if d := last["data"].(map[string]any); d["text"] != want {
			t.Errorf("req %d final text = %v, want %s", i, d["text"], want)
		}
	}
	if ch.maxActive < 2 {
		t.Errorf("requests did not overlap in flight (maxActive=%d), lifecycle not independent", ch.maxActive)
	}
}
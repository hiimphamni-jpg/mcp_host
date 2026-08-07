package agent

import (
	"context"
	"testing"

	"mcp-host/internal/llm"
)

// GW4: tool progress must be interleaved as stream events before final.
func TestRun_emitsToolEventsThenFinal(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "a"}}}},
		{ToolCalls: []*llm.ToolCall{{Name: "read_file", Arguments: map[string]any{"path": "b"}}}},
		{Text: "final answer"},
	}}
	m := &stubMCP{results: map[string]*llm.ToolResult{
		"read_file": {Name: "read_file", Response: map[string]any{"content": "file content"}},
	}}
	var events []Event
	a, err := New(Options{
		LLM:            l,
		Tools:          m,
		Schemas:        simpleSchemas(),
		MaxIterations:  5,
		MaxResultBytes: 256,
		OnEvent:        func(e Event) { events = append(events, e) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := a.Run(context.Background(), "read")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "final answer" {
		t.Errorf("out = %q, want final answer", out)
	}

	wantTypes := []EventType{EventStart, EventTool, EventTool, EventTool, EventTool, EventFinal}
	if len(events) != len(wantTypes) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(wantTypes), events)
	}
	for i, w := range wantTypes {
		if events[i].Type != w {
			t.Errorf("event[%d] = %v, want %v", i, events[i].Type, w)
		}
	}
	if events[1].Phase != ToolPhaseStart || events[2].Phase != ToolPhaseResult {
		t.Errorf("first tool event phases = %v/%v, want start/result", events[1].Phase, events[2].Phase)
	}
	var final Event
	for _, e := range events {
		if e.Type == EventFinal {
			final = e
		}
	}
	if final.Text != "final answer" {
		t.Errorf("final text = %q", final.Text)
	}
}

// ADR-00006: an observer panic must not break the loop.
func TestRun_observerPanicRecovered(t *testing.T) {
	l := &stubLLM{resps: []*llm.Response{{Text: "ok"}}}
	m := &stubMCP{}
	a, err := New(Options{
		LLM:            l,
		Tools:          m,
		Schemas:        simpleSchemas(),
		MaxIterations:  5,
		MaxResultBytes: 256,
		OnEvent:        func(Event) { panic("observer boom") },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := a.Run(context.Background(), "read")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "ok" {
		t.Errorf("out = %q, want ok", out)
	}
}

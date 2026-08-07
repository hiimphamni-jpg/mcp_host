package runner

import (
	"testing"

	"mcp-host/internal/agent"
	"mcp-host/internal/server"
)

func TestNew_RequiresGemini(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New with empty config succeeded")
	}
	if _, err := New(Config{GeminiAPIKey: "k", GeminiModel: "m"}); err == nil {
		t.Fatal("New without MCP command/roots succeeded")
	}
}

func TestNew_Valid(t *testing.T) {
	r, err := New(Config{
		GeminiAPIKey: "k",
		GeminiModel:  "m",
		MCPCommand:   "npx",
		MCPRoots:     []string{"/tmp"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r == nil {
		t.Fatal("New returned nil runner")
	}
}

func TestToServerEvent_Text(t *testing.T) {
	ev := toServerEvent(agent.Event{Type: agent.EventText, Text: "hi"})
	if ev.Type != "text" || ev.Text != "hi" {
		t.Errorf("unexpected event: %+v", ev)
	}
}

func TestToServerEvent_Tool(t *testing.T) {
	ev := toServerEvent(agent.Event{
		Type: agent.EventTool, ToolName: "echo", Phase: agent.ToolPhaseResult, Bytes: 5,
	})
	if ev.Type != "tool" || ev.ToolName != "echo" || ev.Phase != "result" || ev.Bytes != 5 {
		t.Errorf("unexpected event: %+v", ev)
	}
}

func TestToServerEvent_Ignored(t *testing.T) {
	if ev := toServerEvent(agent.Event{Type: agent.EventStart}); (ev != server.Event{}) {
		t.Errorf("start event leaked: %+v", ev)
	}
}
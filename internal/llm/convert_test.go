package llm

import (
	"testing"

	"google.golang.org/genai"
)

func TestToGemContentUserText(t *testing.T) {
	c := toGemContent(&Message{
		Role: RoleUser,
		Text: "list the files",
	})
	if c == nil {
		t.Fatal("toGemContent returned nil")
	}
	if c.Role != string(RoleUser) {
		t.Fatalf("role = %q, want %q", c.Role, RoleUser)
	}
	if len(c.Parts) != 1 || c.Parts[0].Text != "list the files" {
		t.Fatalf("parts = %#v, want single text part", c.Parts)
	}
}

func TestToGemContentModelFunctionCall(t *testing.T) {
	c := toGemContent(&Message{
		Role: RoleModel,
		ToolCalls: []*ToolCall{{
			Name:      "read_file",
			Arguments: map[string]any{"path": "/tmp/a.txt"},
		}},
	})
	if c == nil {
		t.Fatal("toGemContent returned nil")
	}
	if c.Role != string(RoleModel) {
		t.Fatalf("role = %q, want %q", c.Role, RoleModel)
	}
	if len(c.Parts) != 1 || c.Parts[0].FunctionCall == nil {
		t.Fatalf("parts = %#v, want a single FunctionCall part", c.Parts)
	}
	if got := c.Parts[0].FunctionCall; got.Name != "read_file" || got.Args["path"] != "/tmp/a.txt" {
		t.Fatalf("FunctionCall = %#v, want name read_file path /tmp/a.txt", got)
	}
}

func TestToGemContentUserToolResult(t *testing.T) {
	c := toGemContent(&Message{
		Role: RoleUser,
		ToolResults: []*ToolResult{{
			Name:     "read_file",
			Response: map[string]any{"content": "hello"},
		}},
	})
	if c == nil {
		t.Fatal("toGemContent returned nil")
	}
	if c.Role != string(RoleUser) {
		t.Fatalf("role = %q, want %q", c.Role, RoleUser)
	}
	if len(c.Parts) != 1 || c.Parts[0].FunctionResponse == nil {
		t.Fatalf("parts = %#v, want a single FunctionResponse part", c.Parts)
	}
	if got := c.Parts[0].FunctionResponse; got.Name != "read_file" || got.Response["content"] != "hello" {
		t.Fatalf("FunctionResponse = %#v, want name read_file content hello", got)
	}
}

func TestToGemContentMixedParts(t *testing.T) {
	c := toGemContent(&Message{
		Role:      RoleModel,
		Text:      "calling tools",
		ToolCalls: []*ToolCall{{Name: "ls"}},
	})
	if len(c.Parts) != 2 {
		t.Fatalf("parts = %d, want 2 (text + FunctionCall)", len(c.Parts))
	}
	if c.Parts[0].Text != "calling tools" || c.Parts[1].FunctionCall == nil {
		t.Fatalf("parts = %#v, want text then FunctionCall", c.Parts)
	}
}

func TestToolCallToPart(t *testing.T) {
	p := toolCallToPart(&ToolCall{Name: "ls", Arguments: map[string]any{"all": true}})
	if p == nil || p.FunctionCall == nil {
		t.Fatalf("part = %#v, want FunctionCall part", p)
	}
	if p.FunctionCall.Name != "ls" || p.FunctionCall.Args["all"] != true {
		t.Fatalf("FunctionCall = %#v, want ls all=true", p.FunctionCall)
	}
}

func TestToolResultToPart(t *testing.T) {
	p := toolResultToPart(&ToolResult{Name: "ls", Response: map[string]any{"output": []any{"a"}}})
	if p == nil || p.FunctionResponse == nil {
		t.Fatalf("part = %#v, want FunctionResponse part", p)
	}
	if p.FunctionResponse.Name != "ls" {
		t.Fatalf("FunctionResponse name = %q, want ls", p.FunctionResponse.Name)
	}
}

func TestResponseToResponseText(t *testing.T) {
	resp := responseToResponse(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{
				Parts: []*genai.Part{{Text: "done"}},
			},
		}},
	})
	if resp == nil || resp.Text != "done" {
		t.Fatalf("resp = %#v, want text done", resp)
	}
	if len(resp.ToolCalls) != 0 {
		t.Fatalf("tool calls = %d, want 0", len(resp.ToolCalls))
	}
}

func TestResponseToResponseFunctionCall(t *testing.T) {
	resp := responseToResponse(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{
				Parts: []*genai.Part{{
					FunctionCall: &genai.FunctionCall{
						Name: "read_file",
						Args: map[string]any{"path": "/tmp/a.txt"},
					},
				}},
			},
		}},
	})
	if resp == nil || len(resp.ToolCalls) != 1 {
		t.Fatalf("resp = %#v, want 1 tool call", resp)
	}
	got := resp.ToolCalls[0]
	if got.Name != "read_file" || got.Arguments["path"] != "/tmp/a.txt" {
		t.Fatalf("ToolCall = %#v, want read_file path /tmp/a.txt", got)
	}
}

func TestResponseToResponseEmptyCandidates(t *testing.T) {
	resp := responseToResponse(&genai.GenerateContentResponse{})
	if resp == nil || resp.Text != "" || len(resp.ToolCalls) != 0 {
		t.Fatalf("resp = %#v, want empty", resp)
	}
}

func TestResponseToResponseNil(t *testing.T) {
	resp := responseToResponse(nil)
	if resp == nil || resp.Text != "" || len(resp.ToolCalls) != 0 {
		t.Fatalf("resp = %#v, want empty", resp)
	}
}

func TestResponseToResponseMixed(t *testing.T) {
	resp := responseToResponse(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{
				Parts: []*genai.Part{
					{Text: "running"},
					{FunctionCall: &genai.FunctionCall{Name: "ls"}},
				},
			},
		}},
	})
	if resp == nil || resp.Text != "running" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "ls" {
		t.Fatalf("resp = %#v, want text running + ls call", resp)
	}
}

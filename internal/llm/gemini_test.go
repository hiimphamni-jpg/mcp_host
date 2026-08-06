package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/genai"
)

// fakeModels is a modelsClient seam: it records the last call and returns a
// scripted response or error without any network or credentials.
type fakeModels struct {
	resp     *genai.GenerateContentResponse
	err      error
	model    string
	contents []*genai.Content
	config   *genai.GenerateContentConfig
}

func (f *fakeModels) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	f.model = model
	f.contents = contents
	f.config = config
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func newTestGemini(fake *fakeModels) *Gemini {
	return &Gemini{
		model:     "gemini-test-model",
		timeout:   time.Second,
		generator: fake,
		declByTool: map[string]*genai.FunctionDeclaration{
			"read_file": {Name: "read_file"},
			"ls":        {Name: "ls"},
		},
	}
}

func TestGenerateTextOnly(t *testing.T) {
	fake := &fakeModels{resp: &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "hello"}}}}},
	}}
	g := newTestGemini(fake)

	resp, err := g.Generate(context.Background(), &Request{
		Contents: []*Message{{Role: RoleUser, Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Text != "hello" || len(resp.ToolCalls) != 0 {
		t.Fatalf("resp = %#v, want text hello", resp)
	}
	if fake.model != "gemini-test-model" {
		t.Fatalf("model = %q, want gemini-test-model", fake.model)
	}
}

func TestGenerateToolCall(t *testing.T) {
	fake := &fakeModels{resp: &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{
			FunctionCall: &genai.FunctionCall{Name: "read_file", Args: map[string]any{"path": "/tmp/a.txt"}},
		}}}}},
	}}
	g := newTestGemini(fake)

	resp, err := g.Generate(context.Background(), &Request{})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read_file" || resp.ToolCalls[0].Arguments["path"] != "/tmp/a.txt" {
		t.Fatalf("resp = %#v, want read_file tool call", resp)
	}
}

func TestGenerateTimeoutMapsToErrTimeout(t *testing.T) {
	fake := &fakeModels{err: context.DeadlineExceeded}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("errors.Is(err, ErrTimeout) = false, err = %v", err)
	}
}

func TestGenerateContextCancelPassesThrough(t *testing.T) {
	cancelErr := errors.New("generator canceled")
	fake := &fakeModels{err: context.Canceled}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if errors.Is(err, ErrTimeout) {
		t.Fatalf("cancel must not map to ErrTimeout, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled to pass through", err)
	}
	_ = cancelErr
}

func TestGenerateProviderError(t *testing.T) {
	fake := &fakeModels{err: errors.New("boom")}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("errors.Is(err, ErrProvider) = false, err = %v", err)
	}
}

func TestGenerateBuildsToolConfigForNamedTools(t *testing.T) {
	fake := &fakeModels{resp: &genai.GenerateContentResponse{}}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{ToolNames: []string{"read_file", "ls"}})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if fake.config == nil || len(fake.config.Tools) != 1 {
		t.Fatalf("config = %#v, want 1 tool", fake.config)
	}
	decls := fake.config.Tools[0].FunctionDeclarations
	if len(decls) != 2 || decls[0].Name != "read_file" || decls[1].Name != "ls" {
		t.Fatalf("decls = %#v, want read_file + ls", decls)
	}
}

func TestGenerateNoToolsWhenNoneNamed(t *testing.T) {
	fake := &fakeModels{resp: &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "x"}}}}},
	}}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{ToolNames: nil})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if fake.config != nil && len(fake.config.Tools) != 0 {
		t.Fatalf("config tools = %#v, want none", fake.config.Tools)
	}
}

func TestGenerateBuildsContents(t *testing.T) {
	fake := &fakeModels{resp: &genai.GenerateContentResponse{}}
	g := newTestGemini(fake)

	_, err := g.Generate(context.Background(), &Request{Contents: []*Message{{Role: RoleUser, Text: "hi"}}})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(fake.contents) != 1 || fake.contents[0].Parts[0].Text != "hi" {
		t.Fatalf("contents = %#v, want one user text part", fake.contents)
	}
}

package llm

import (
	"context"
	"errors"
	"time"

	"google.golang.org/genai"
)

// modelsClient is the seam the Gemini adapter calls to generate content. It
// matches *genai.Models.GenerateContent's signature so the real client (via
// client.Models) and a test fake both satisfy it (plan Step 5 — fake-driven
// tests, no network or credentials).
type modelsClient interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

// Gemini is a provider adapter implementing LLM on top of the Gemini SDK. It
// owns the neutral<->genai conversion and the per-request LLM_TIMEOUT applied
// via context; internal/agent (FEAT-00006) depends only on the LLM interface.
type Gemini struct {
	model      string
	timeout    time.Duration
	generator  modelsClient
	declByTool map[string]*genai.FunctionDeclaration
}

// New builds a Gemini-backed LLM. client supplies the real generator (its
// Models service); model is the provider model name; timeout is LLM_TIMEOUT
// applied per request; decls maps tool names to already-prepared function
// declarations (built once via internal/mapping; prepared by the agent in
// FEAT-00006 — open decision recorded for /architect).
func New(client *genai.Client, model string, timeout time.Duration, decls map[string]*genai.FunctionDeclaration) *Gemini {
	return &Gemini{
		model:      model,
		timeout:    timeout,
		generator:  client.Models,
		declByTool: decls,
	}
}

// Generate converts the neutral request into genai.content, wires the declared
// tools into the config, and calls the provider under LLM_TIMEOUT. A deadline
// maps to ErrTimeout; caller cancellation passes through as context.Canceled.
func (g *Gemini) Generate(ctx context.Context, req *Request) (*Response, error) {
	contents := make([]*genai.Content, 0, len(req.Contents))
	for _, msg := range req.Contents {
		if c := toGemContent(msg); c != nil {
			contents = append(contents, c)
		}
	}

	config := &genai.GenerateContentConfig{}
	if decls := g.selectedDecls(req.ToolNames); len(decls) > 0 {
		config.Tools = []*genai.Tool{{FunctionDeclarations: decls}}
	}

	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	resp, err := g.generator.GenerateContent(ctx, g.model, contents, config)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			return nil, timeoutError("Generate")
		case errors.Is(err, context.Canceled):
			return nil, err
		default:
			return nil, providerError("Generate")
		}
	}
	return responseToResponse(resp), nil
}

// selectedDecls resolves the requested tool names to their prepared
// declarations, preserving the request's order. Only explicitly named tools are
// wired into the request; an empty list carries no tool declarations (plan Step
// 5 — tools are built for named tools).
func (g *Gemini) selectedDecls(names []string) []*genai.FunctionDeclaration {
	if len(names) == 0 {
		return nil
	}
	out := make([]*genai.FunctionDeclaration, 0, len(names))
	for _, n := range names {
		if d, ok := g.declByTool[n]; ok {
			out = append(out, d)
		}
	}
	return out
}

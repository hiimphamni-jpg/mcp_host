package llm

import "context"

// LLM is the provider-neutral generation surface that internal/agent
// (FEAT-00006) depends on. The interface carries no provider SDK type, so a
// fake or an alternative provider can satisfy it without touching the Gemini
// SDK (TDD §2.1, AC5).
type LLM interface {
	// Generate produces a response for the given request within the configured
	// timeout, or a typed error. Caller cancellation surfaces as context.Canceled.
	Generate(ctx context.Context, req *Request) (*Response, error)
}

// Request is a neutral generation request. ToolNames is an indirection to the
// adapter's prepared declarations: the agent lists which tools are available and
// the adapter maps each name to an already-built declaration in one place
// (open decision for /architect before FEAT-00006).
type Request struct {
	Contents  []*Message
	ToolNames []string
}

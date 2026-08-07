package llm

import "google.golang.org/genai"

// This file holds pure, side-effect-free conversion helpers between the
// neutral conversation types (message.go) and the genai SDK graph (content,
// parts, response). They have no I/O or credentials, so they are trivially
// unit-testable (plan Step 4, TDD Red -> Green).

// toGemContent converts a neutral Message into a genai.Content. Text is carried
// as a text part; model ToolCalls become FunctionCall parts in the model role,
// and user ToolResults become FunctionResponse parts in the user role.
func toGemContent(msg *Message) *genai.Content {
	if msg == nil {
		return nil
	}
	c := &genai.Content{Role: string(msg.Role)}
	if msg.Text != "" {
		c.Parts = append(c.Parts, genai.NewPartFromText(msg.Text))
	}
	for _, tc := range msg.ToolCalls {
		if p := toolCallToPart(tc); p != nil {
			c.Parts = append(c.Parts, p)
		}
	}
	for _, tr := range msg.ToolResults {
		if p := toolResultToPart(tr); p != nil {
			c.Parts = append(c.Parts, p)
		}
	}
	return c
}

// toolCallToPart converts a neutral tool call into a genai FunctionCall part.
// The provider-mandated thought signature is carried back on the part so the
// replayed FunctionCall part is accepted by the API.
func toolCallToPart(tc *ToolCall) *genai.Part {
	if tc == nil {
		return nil
	}
	p := genai.NewPartFromFunctionCall(tc.Name, tc.Arguments)
	p.ThoughtSignature = tc.ThoughtSignature
	return p
}

// toolResultToPart converts a neutral tool result into a genai FunctionResponse
// part (fed back to the model as a user role part).
func toolResultToPart(tr *ToolResult) *genai.Part {
	if tr == nil {
		return nil
	}
	return genai.NewPartFromFunctionResponse(tr.Name, tr.Response)
}

// responseToResponse converts a provider response into a neutral Response. The
// first candidate's concatenated text is used when present; FunctionCall parts
// are surfaced as neutral ToolCalls for the agent to execute.
func responseToResponse(r *genai.GenerateContentResponse) *Response {
	if r == nil {
		return &Response{}
	}
	resp := &Response{Text: r.Text()}
	if len(r.Candidates) == 0 || r.Candidates[0].Content == nil {
		return resp
	}
	for _, part := range r.Candidates[0].Content.Parts {
		fc := part.FunctionCall
		if fc == nil {
			continue
		}
		resp.ToolCalls = append(resp.ToolCalls, &ToolCall{
			Name:             fc.Name,
			Arguments:        fc.Args,
			ThoughtSignature: part.ThoughtSignature,
		})
	}
	return resp
}

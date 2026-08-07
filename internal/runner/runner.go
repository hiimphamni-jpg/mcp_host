// Package runner implements the real per-request orchestration for the gateway
// (FEAT-00013): it composes internal/agent with mcpclient/mapping/policy/llm
// exactly once per request, wires agent events to the HTTP layer, and guarantees
// the MCP child process is closed on every path (no orphan, AC4). It is the only
// gateway code allowed to import the Go SDKs (mcp-go, genai), keeping the
// internal/server HTTP layer SDK-free (TDD GW8).
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"

	"mcp-host/internal/agent"
	"mcp-host/internal/llm"
	"mcp-host/internal/mapping"
	"mcp-host/internal/mcpclient"
	"mcp-host/internal/policy"
	"mcp-host/internal/server"
)

// Config carries the runtime knobs the Runner needs for one composition.
type Config struct {
	GeminiAPIKey   string
	GeminiModel    string
	MCPCommand     string
	MCPArgs        []string
	MCPRoots       []string
	MCPTimeout     time.Duration
	LLMTimeout     time.Duration
	MaxIterations  int
	MaxResultBytes int
}

// Runner is the server.Handler implementation built from Config.
type Runner struct {
	cfg Config
}

// New validates the Config minimally and returns a Runner.
func New(cfg Config) (*Runner, error) {
	if cfg.GeminiAPIKey == "" || cfg.GeminiModel == "" {
		return nil, errors.New("runner: Gemini API key and model are required")
	}
	if cfg.MCPCommand == "" || len(cfg.MCPRoots) == 0 {
		return nil, errors.New("runner: MCP command and allowed roots are required")
	}
	return &Runner{cfg: cfg}, nil
}

// Chat runs one bounded agent loop for prompt, emitting progress Events, and
// returns the final answer. It builds a fresh MCP client per request and always
// closes it (ADR-00005). Emit is best-effort; the caller decides what to stream.
func (r *Runner) Chat(ctx context.Context, prompt string, emit func(server.Event)) (string, error) {
	pol, err := policy.New(r.cfg.MCPCommand, r.cfg.MCPArgs, r.cfg.MCPRoots)
	if err != nil {
		return "", err
	}

	client, err := mcpclient.NewStdio(pol, r.cfg.MCPTimeout)
	if err != nil {
		return "", err
	}
	defer client.Close()

	if err := client.Initialize(ctx); err != nil {
		return "", err
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		return "", err
	}

	decls, err := mapping.MapTools(tools.Tools)
	if err != nil {
		return "", err
	}
	declsByName := make(map[string]*genai.FunctionDeclaration, len(decls))
	for _, d := range decls {
		declsByName[d.Name] = d
	}
	schemas := make(map[string]any, len(tools.Tools))
	for _, t := range tools.Tools {
		s, err := neutralSchema(t)
		if err != nil {
			return "", err
		}
		schemas[t.Name] = s
	}

	gen, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: r.cfg.GeminiAPIKey})
	if err != nil {
		return "", err
	}

	var onEvent func(agent.Event)
	if emit != nil {
		onEvent = func(ev agent.Event) {
			emit(toServerEvent(ev))
		}
	}

	loop, err := agent.New(agent.Options{
		LLM:            llm.New(gen, r.cfg.GeminiModel, r.cfg.LLMTimeout, declsByName),
		Tools:          &mcpToolCaller{client: client},
		Schemas:        schemas,
		MaxIterations:  r.cfg.MaxIterations,
		MaxResultBytes: r.cfg.MaxResultBytes,
		OnEvent:        onEvent,
	})
	if err != nil {
		return "", err
	}

	return loop.Run(ctx, prompt)
}

// toServerEvent converts an agent observation into the HTTP layer's neutral
// Event. No secrets are copied (ADR-1 D3).
func toServerEvent(ev agent.Event) server.Event {
	switch ev.Type {
	case agent.EventText:
		return server.Event{Type: "text", Text: ev.Text}
	case agent.EventTool:
		return server.Event{Type: "tool", ToolName: ev.ToolName, Phase: string(ev.Phase), Bytes: ev.Bytes}
	default:
		return server.Event{}
	}
}

// mcpToolCaller adapts mcpclient.Client to the agent's ToolCaller seam,
// converting the mcp-go result to a neutral llm.ToolResult (mirrors cmd/server,
// ADR-1 D2).
type mcpToolCaller struct {
	client mcpclient.Client
}

func (c *mcpToolCaller) CallTool(ctx context.Context, name string, args map[string]any) (*llm.ToolResult, error) {
	res, err := c.client.CallTool(ctx, name, args)
	if err != nil {
		return nil, err
	}
	tr := &llm.ToolResult{Name: name}
	if res == nil {
		tr.Response = map[string]any{"content": ""}
		return tr, nil
	}
	var text strings.Builder
	for _, content := range res.Content {
		switch v := content.(type) {
		case mcp.TextContent:
			text.WriteString(v.Text)
		case mcp.EmbeddedResource:
			if rc, ok := v.Resource.(mcp.TextResourceContents); ok {
				text.WriteString(rc.Text)
			}
		}
	}
	response := map[string]any{"content": text.String()}
	if res.IsError {
		response["error"] = true
	}
	tr.Response = response
	return tr, nil
}

func neutralSchema(t mcp.Tool) (map[string]any, error) {
	b, err := json.Marshal(t.InputSchema)
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}
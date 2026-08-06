package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"

	"mcp-host/internal/agent"
	"mcp-host/internal/config"
	"mcp-host/internal/llm"
	"mcp-host/internal/mapping"
	"mcp-host/internal/mcpclient"
	"mcp-host/internal/policy"
)

func main() {
	if err := run(os.Stdout, os.Stdin, os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run is the composition root (ADR-0001 D1). Without a --prompt it reads one
// prompt from stdin only when stdin is not a terminal (piped/non-interactive);
// with a --prompt it discovers the MCP tools, maps them once (declarations for
// the Gemini adapter and neutral schemas for the agent), and runs one bounded
// agent loop. The MCP child process is closed on every path so no orphan
// survives (AC4).
func run(out io.Writer, stdin io.Reader, args ...string) error {
	fs := flag.NewFlagSet("mcp-host", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prompt := fs.String("prompt", "", "run one agent loop with the given prompt")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	p := *prompt
	if p == "" && !isTerminal(stdin) {
		line, err := readPromptFromStdin(stdin)
		if err != nil {
			return fmt.Errorf("read prompt from stdin: %w", err)
		}
		p = line
	}

	if p == "" {
		return writeBootstrap(out)
	}
	return runAgentLoop(context.Background(), out, cfg, p)
}

// readPromptFromStdin reads the whole stdin as a single prompt, trimmed of
// surrounding whitespace. An empty stream yields an empty string, which the
// caller treats as "no prompt" (bootstrap path).
func readPromptFromStdin(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// isTerminal reports whether r is an interactive terminal (character device).
// Non-*os.File readers (injected buffers, pipes) are never terminals, so the
// stdin fallback stays testable without a real TTY.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func writeBootstrap(out io.Writer) error {
	_, err := fmt.Fprintln(out, "MCP Host bootstrap complete. Integrations are not configured or started.")
	if err != nil {
		return fmt.Errorf("write bootstrap status: %w", err)
	}
	return nil
}

// runAgentLoop performs discovery + mapping in the composition root, then runs
// one bounded agent loop for the prompt, printing the final model text.
func runAgentLoop(ctx context.Context, out io.Writer, cfg *config.Config, prompt string) error {
	pol, err := policy.New(cfg.MCPFilesystemCommand, cfg.MCPFilesystemArgs, cfg.MCPAllowedRoots)
	if err != nil {
		return fmt.Errorf("build policy: %w", err)
	}

	client, err := mcpclient.NewStdio(pol, cfg.MCPTimeout)
	if err != nil {
		return fmt.Errorf("start MCP client: %w", err)
	}
	defer client.Close()

	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize MCP: %w", err)
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list MCP tools: %w", err)
	}

	decls, err := mapping.MapTools(tools.Tools)
	if err != nil {
		return fmt.Errorf("map tool schemas: %w", err)
	}
	declsByName := make(map[string]*genai.FunctionDeclaration, len(decls))
	for _, d := range decls {
		declsByName[d.Name] = d
	}

	schemas := make(map[string]any, len(tools.Tools))
	for _, t := range tools.Tools {
		s, err := neutralSchema(t)
		if err != nil {
			return fmt.Errorf("neutral schema for %q: %w", t.Name, err)
		}
		schemas[t.Name] = s
	}

	gen, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.GeminiAPIKey})
	if err != nil {
		return fmt.Errorf("create Gemini client: %w", err)
	}

	loop, err := agent.New(agent.Options{
		LLM:            llm.New(gen, cfg.GeminiModel, cfg.LLMTimeout, declsByName),
		Tools:          &mcpToolCaller{client: client},
		Schemas:        schemas,
		MaxIterations:  cfg.MaxAgentIterations,
		MaxResultBytes: cfg.MaxResultBytes,
	})
	if err != nil {
		return fmt.Errorf("build agent: %w", err)
	}

	answer, err := loop.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent run: %w", err)
	}
	_, err = fmt.Fprintln(out, answer)
	if err != nil {
		return fmt.Errorf("write answer: %w", err)
	}
	return nil
}

// mcpToolCaller adapts mcpclient.Client to the agent's SDK-free ToolCaller seam
// (ADR-0001 D2): the raw *mcp.CallToolResult content is converted to a neutral
// llm.ToolResult before the agent sees it, keeping internal/agent free of
// mcp-go. Caller context is passed straight through to the MCP call so a
// cancellation reaches the in-flight call.
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

// neutralSchema converts a discovered tool's mcp-go input schema into the
// agent's neutral JSON Schema object (map[string]any) via JSON round-trip so
// the agent never imports mcp-go (ADR-0001 D1).
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

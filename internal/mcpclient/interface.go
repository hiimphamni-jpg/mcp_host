package mcpclient

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// Client is the MCP stdio lifecycle surface used by the agent loop. The future
// internal/agent (FEAT-00006) depends only on this interface so a fake can
// stand in for tests (TDD §2.1, AC5). This package must not depend on
// internal/llm or internal/agent.
type Client interface {
	// Initialize completes the MCP initialize handshake within MCP_TIMEOUT.
	Initialize(ctx context.Context) error
	// ListTools discovers the server's tools within MCP_TIMEOUT.
	ListTools(ctx context.Context) (*mcp.ListToolsResult, error)
	// CallTool invokes the named tool with arguments within MCP_TIMEOUT.
	CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error)
	// Close terminates the child process and closes the transport. It is safe
	// to call multiple times, including on partially-initialized clients.
	Close() error
}

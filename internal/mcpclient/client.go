package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"mcp-host/internal/policy"
)

const (
	clientName    = "mcp-host"
	clientVersion = "0.1.0"
)

// Compile-time assertion that *StdioClient satisfies the lifecycle Client
// interface (AC5), so a future internal/agent can depend on the interface only.
var _ Client = (*StdioClient)(nil)

// StdioClient wraps an mcp-go stdio client and enforces a per-operation
// MCP_TIMEOUT. It isolates the mcp-go dependency from the rest of the host so
// transport upgrades never leak into internal/agent.
type StdioClient struct {
	raw     *client.Client
	timeout time.Duration
	mu      sync.Mutex
	closed  bool
}

// NewStdio spawns the configured MCP child process. The argv is built exactly
// from the validated policy Command + Args + Roots; it is never LLM-supplied
// (TDD §8). It returns ErrProcessExit if the child cannot be started. The
// client is not yet initialized; callers must call Initialize first.
func NewStdio(p *policy.FilesystemPolicy, timeout time.Duration) (*StdioClient, error) {
	if p == nil {
		return nil, errors.New("mcpclient: nil filesystem policy")
	}
	if timeout <= 0 {
		return nil, errors.New("mcpclient: timeout must be positive")
	}

	args := append(p.Args(), p.Roots()...)
	raw, err := client.NewStdioMCPClientWithOptions(p.Command(), nil, args)
	if err != nil {
		return nil, newError(KindProcessExit, "spawn")
	}

	return &StdioClient{raw: raw, timeout: timeout}, nil
}

// Initialize completes the MCP initialize handshake within MCP_TIMEOUT.
func (c *StdioClient) Initialize(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return newError(KindProcessExit, "initialize")
	}
	c.mu.Unlock()

	opCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := mcp.InitializeRequest{}
	req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	req.Params.ClientInfo = mcp.Implementation{Name: clientName, Version: clientVersion}

	_, err := c.raw.Initialize(opCtx, req)
	return classify(err, KindInitFailed, "initialize", c.timeout)
}

// ListTools discovers the server's tools once within MCP_TIMEOUT.
func (c *StdioClient) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	if err := c.checkClosed("tools/list"); err != nil {
		return nil, err
	}

	opCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	result, err := c.raw.ListTools(opCtx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, classify(err, KindDiscoveryFailed, "tools/list", c.timeout)
	}
	return result, nil
}

func (c *StdioClient) checkClosed(op string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return newError(KindProcessExit, op)
	}
	return nil
}

// CallTool invokes a validated tool within MCP_TIMEOUT. A context deadline,
// child-process exit, or invalid response is returned as the corresponding
// typed error so the host remains usable (AC4).
func (c *StdioClient) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	if err := c.checkClosed("tools/call"); err != nil {
		return nil, err
	}

	opCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := c.raw.CallTool(opCtx, req)
	if err != nil {
		return nil, classify(err, KindCallFailed, "tools/call", c.timeout)
	}
	return result, nil
}

// Close terminates the child process and closes the transport. It is safe to
// call multiple times and on partially-initialized clients (spawn failure,
// cancelled context): a cancelled/failed context still yields a clean child
// termination with no orphaned process.
func (c *StdioClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	if c.raw == nil {
		return nil
	}
	if err := c.raw.Close(); err != nil {
		return fmt.Errorf("mcpclient: close: %w", err)
	}
	return nil
}

// classify maps an mcp-go error to a typed, user-safe mcpclient error.
func classify(err error, k Kind, op string, timeout time.Duration) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &TimeoutError{Operation: op, Timeout: timeout}
	}
	if errors.Is(err, transport.ErrTransportClosed) {
		return newError(KindProcessExit, op)
	}
	if isJSONRPCSentinel(err) {
		return newError(KindInvalidResponse, op)
	}
	return newError(k, op)
}

// isJSONRPCSentinel reports whether the error came from a server-side JSON-RPC
// error payload (mcp-go maps those to these sentinel errors via AsError).
func isJSONRPCSentinel(err error) bool {
	for _, sentinel := range []error{
		mcp.ErrParseError,
		mcp.ErrInvalidRequest,
		mcp.ErrMethodNotFound,
		mcp.ErrInvalidParams,
		mcp.ErrInternalError,
		mcp.ErrRequestInterrupted,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}

package mcpclient

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors classify MCP lifecycle failures for errors.Is. The typed
// errors returned by this package wrap these sentinels and carry only
// user-safe messages: no credentials, host paths, or stack traces (TDD §8,
// AC4), so the host never crashes on a child-process exit or protocol fault.
var (
	// ErrProcessExit is returned when the MCP child process exits unexpectedly
	// or cannot be spawned.
	ErrProcessExit = errors.New("MCP child process exited unexpectedly")
	// ErrInitFailed is returned when the MCP initialize handshake fails.
	ErrInitFailed = errors.New("MCP initialization failed")
	// ErrDiscoveryFailed is returned when tools/list fails.
	ErrDiscoveryFailed = errors.New("MCP tool discovery failed")
	// ErrCallFailed is returned when tools/call fails.
	ErrCallFailed = errors.New("MCP tool call failed")
	// ErrInvalidResponse is returned when the server responds with an
	// unparseable or protocol-level error payload.
	ErrInvalidResponse = errors.New("MCP returned an invalid response")
	// ErrTimeout is returned when an MCP operation exceeds MCP_TIMEOUT.
	ErrTimeout = errors.New("MCP operation timed out")
)

// Kind classifies the lifecycle operation that failed.
type Kind uint8

const (
	KindProcessExit Kind = iota
	KindInitFailed
	KindDiscoveryFailed
	KindCallFailed
	KindInvalidResponse
)

func sentinelFor(k Kind) error {
	switch k {
	case KindProcessExit:
		return ErrProcessExit
	case KindInitFailed:
		return ErrInitFailed
	case KindDiscoveryFailed:
		return ErrDiscoveryFailed
	case KindCallFailed:
		return ErrCallFailed
	case KindInvalidResponse:
		return ErrInvalidResponse
	}
	return errors.New("mcpclient: unknown error kind")
}

// Error is a typed, user-safe MCP client failure. Operation is a short,
// non-sensitive label such as "initialize" or "tools/call".
type Error struct {
	Kind      Kind
	Operation string
	message   string
}

func (e *Error) Error() string { return e.message }

func (e *Error) Unwrap() error { return sentinelFor(e.Kind) }

func newError(k Kind, op string) *Error {
	return &Error{Kind: k, Operation: op, message: fmt.Sprintf("%s: %s", sentinelFor(k), op)}
}

// TimeoutError reports that an MCP operation exceeded MCP_TIMEOUT.
type TimeoutError struct {
	Operation string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("%s: %s", ErrTimeout, e.Operation)
}

func (e *TimeoutError) Unwrap() error { return ErrTimeout }

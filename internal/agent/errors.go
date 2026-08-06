package agent

import (
	"errors"
	"fmt"
)

// Sentinel errors classify agent-loop failures for errors.Is. The typed *Error
// returned by this package wraps these sentinels and carries a short,
// non-sensitive message only — never a raw prompt, tool output, or provider
// stack trace (rules/security.md, rules/error-handling.md §5), so the host
// never leaks secrets through a failure.
var (
	// ErrIterationLimit is returned when the loop exhausts AGENT_MAX_ITERATIONS
	// without producing a final answer (business-logic §5).
	ErrIterationLimit = errors.New("agent: iteration limit reached")
)

// Error is a typed, user-safe agent-loop failure. Cause is one of this
// package's sentinels via Unwrap, so callers can classify with errors.Is
// without string-matching. Operation is a short label such as "Run" or
// "Generate".
type Error struct {
	Cause     error
	Operation string
	message   string
}

func (e *Error) Error() string { return e.message }

func (e *Error) Unwrap() error { return e.Cause }

// iterationLimitError builds an ErrIterationLimit-typed user-safe error.
func iterationLimitError() error {
	return &Error{
		Cause:     ErrIterationLimit,
		Operation: "Run",
		message:   fmt.Sprintf("%s", ErrIterationLimit),
	}
}

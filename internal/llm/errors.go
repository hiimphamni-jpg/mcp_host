package llm

import (
	"errors"
	"fmt"
)

// Sentinel errors classify LLM failures for errors.Is. The typed errors
// returned by this package wrap these sentinels and carry a short,
// non-sensitive operation label only: never the API key, raw prompt text, or
// provider stack traces (rules/security.md, rules/error-handling.md §5), so the
// host never leaks secrets or prompt content to a user.
var (
	// ErrTimeout is returned when a generation exceeds LLM_TIMEOUT.
	ErrTimeout = errors.New("LLM request timed out")
	// ErrProvider is returned for a non-timeout provider-facing failure.
	ErrProvider = errors.New("LLM provider request failed")
)

// Error is a typed, user-safe LLM failure. Operation is a short label such as
// "Generate"; Cause is one of this package's sentinels via Unwrap, so callers
// can classify with errors.Is without string-matching.
type Error struct {
	Cause     error
	Operation string
	message   string
}

func (e *Error) Error() string { return e.message }

func (e *Error) Unwrap() error { return e.Cause }

// timeoutError builds an ErrTimeout-typed user-safe error.
func timeoutError(op string) error {
	return &Error{
		Cause:     ErrTimeout,
		Operation: op,
		message:   fmt.Sprintf("%s: %s", ErrTimeout, op),
	}
}

// providerError builds an ErrProvider typed failure. The underlying provider
// detail is intentionally not surfaced to keep the message user-safe.
func providerError(op string) error {
	return &Error{
		Cause:     ErrProvider,
		Operation: op,
		message:   fmt.Sprintf("%s: %s", ErrProvider, op),
	}
}

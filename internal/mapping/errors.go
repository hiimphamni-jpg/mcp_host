package mapping

import (
	"errors"
	"fmt"
)

// Sentinel errors classify mapping failures for errors.Is. The typed
// *Error returned by this package wraps these sentinels and carries the
// offending tool name and field path so callers can trace a malformed
// schema back to the exact MCP tool (TDD §6 / business-logic §6).
var (
	// ErrUnsupportedType is returned when a JSON Schema type outside the
	// supported MVP subset (object/string/number/integer/boolean/array) is
	// encountered.
	ErrUnsupportedType = errors.New("mapping: unsupported schema type")
	// ErrDuplicateName is returned when two tools share the same name.
	ErrDuplicateName = errors.New("mapping: duplicate tool name")
	// ErrMalformed is returned when a schema is structurally invalid (e.g. a
	// property that is not an object, a non-string enum member, or recursion
	// exceeding the depth guard).
	ErrMalformed = errors.New("mapping: malformed tool schema")
)

// Error is a typed mapping failure carrying the tool name and field path
// involved. It implements error and also exposes Name and Path for
// structured handling without string parsing.
type Error struct {
	// Cause is one of the sentinel errors in this package (via Unwrap).
	Cause error
	// Tool is the MCP tool name that failed to map.
	Tool string
	// Path is the dot-delimited field path within the tool schema, e.g.
	// "config.inner.x", or "" for the tool root.
	Path string
	// Description is the human-readable reason.
	Description string
}

func (e *Error) Error() string {
	prefix := "mapping: "
	if e.Tool != "" {
		prefix = fmt.Sprintf("tool %q", e.Tool)
		if e.Path != "" {
			prefix += fmt.Sprintf(" field %q", e.Path)
		}
	}
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", prefix, e.Description)
	}
	return prefix
}

// Unwrap exposes the cause sentinel for errors.Is.
func (e *Error) Unwrap() error { return e.Cause }

func unsupportedType(tool, path, typeDesc string) error {
	return &Error{ErrUnsupportedType, tool, path, fmt.Sprintf("unsupported type %q", typeDesc)}
}

func malformed(tool, path, reason string) error {
	return &Error{ErrMalformed, tool, path, reason}
}

func duplicateName(tool string) error {
	return &Error{ErrDuplicateName, tool, "", "duplicate tool name"}
}

package agent

import (
	"errors"
	"testing"
)

func TestIterationLimitError_IsSentinel(t *testing.T) {
	err := iterationLimitError()
	if !errors.Is(err, ErrIterationLimit) {
		t.Fatalf("err = %v, want it to match ErrIterationLimit via errors.Is", err)
	}
}

func TestError_MessageDoesNotLeakCause(t *testing.T) {
	secret := "super-secret-provider-token stacktrace"
	err := &Error{Cause: errors.New(secret), Operation: "Generate", message: "LLM generation failed"}
	if errors.Is(err, ErrIterationLimit) {
		t.Fatal("a Generate error must not unwrap to ErrIterationLimit")
	}
	if err.Error() != "LLM generation failed" {
		t.Errorf("Error() = %q, want the safe message only", err.Error())
	}
}

func TestError_UnwrapPreservesSentinel(t *testing.T) {
	inner := errors.New("inner")
	err := &Error{Cause: inner, Operation: "Run", message: "bounded"}
	if !errors.Is(err, inner) {
		t.Fatal("Unwrap must surface the cause for errors.Is")
	}
}

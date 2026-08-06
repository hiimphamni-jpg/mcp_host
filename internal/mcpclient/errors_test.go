package mcpclient

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestError_UnwrapMatchesSentinel(t *testing.T) {
	cases := []struct {
		kind     Kind
		sentinel error
	}{
		{KindProcessExit, ErrProcessExit},
		{KindInitFailed, ErrInitFailed},
		{KindDiscoveryFailed, ErrDiscoveryFailed},
		{KindCallFailed, ErrCallFailed},
		{KindInvalidResponse, ErrInvalidResponse},
	}

	for _, tc := range cases {
		t.Run(tc.sentinel.Error(), func(t *testing.T) {
			err := newError(tc.kind, "test-operation")
			if !errors.Is(err, tc.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, tc.sentinel)
			}
			if errors.Is(err, ErrTimeout) {
				t.Error("errors.Is(err, ErrTimeout) = true, want false")
			}
		})
	}
}

func TestError_MessageIsUserSafe(t *testing.T) {
	err := newError(KindCallFailed, "tools/call")
	if err.Error() == "" {
		t.Fatal("error message is empty")
	}
	if strings.ContainsAny(err.Error(), "\\") {
		t.Errorf("error message may leak a host path: %q", err.Error())
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") {
		t.Errorf("error message may leak credentials: %q", err.Error())
	}
}

func TestTimeoutError_IsErrTimeout(t *testing.T) {
	err := &TimeoutError{Operation: "initialize", Timeout: 5 * time.Second}
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("errors.Is(err, ErrTimeout) = false, want true")
	}
	msg := err.Error()
	if !strings.Contains(msg, "initialize") {
		t.Errorf("message %q does not name the operation", msg)
	}
	if strings.Contains(msg, "secret") {
		t.Errorf("message may leak credentials: %q", msg)
	}
}

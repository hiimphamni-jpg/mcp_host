package main

import (
	"bytes"
	"testing"
)

func TestRun_WritesSafeBootstrapStatus(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-secret")

	var output bytes.Buffer
	if err := run(&output); err != nil {
		t.Fatalf("run returned an error: %v", err)
	}

	const want = "MCP Host bootstrap complete. Integrations are not configured or started.\n"
	if output.String() != want {
		t.Fatalf("run output = %q, want %q", output.String(), want)
	}
}

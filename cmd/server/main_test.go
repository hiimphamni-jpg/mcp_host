package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GEMINI_API_KEY", "test-secret")
	t.Setenv("GEMINI_MODEL", "gemini-2.0-flash")
	t.Setenv("MCP_FILESYSTEM_COMMAND", "npx")
	t.Setenv("MCP_FILESYSTEM_ARGS_JSON", `["-y","@modelcontextprotocol/server-filesystem"]`)
	t.Setenv("MCP_ALLOWED_ROOTS", os.TempDir())
	t.Setenv("MCP_TIMEOUT", "30s")
	t.Setenv("LLM_TIMEOUT", "45s")
	t.Setenv("AGENT_MAX_ITERATIONS", "10")
	t.Setenv("MCP_MAX_RESULT_BYTES", "65536")
}

func TestRun_WritesSafeBootstrapStatus(t *testing.T) {
	setValidEnv(t)

	var output bytes.Buffer
	if err := run(&output); err != nil {
		t.Fatalf("run returned an error: %v", err)
	}

	const want = "MCP Host bootstrap complete. Integrations are not configured or started.\n"
	if output.String() != want {
		t.Fatalf("run output = %q, want %q", output.String(), want)
	}
}

func TestRun_InvalidConfigReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("GEMINI_API_KEY", "")

	var output bytes.Buffer
	if err := run(&output); err == nil {
		t.Fatal("run succeeded with invalid configuration")
	}
	if strings.Contains(output.String(), "bootstrap complete") {
		t.Error("run wrote bootstrap status despite invalid configuration")
	}
}

func TestRun_InvalidConfigDoesNotLeakSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("GEMINI_API_KEY", "topsecret-xyz")
	t.Setenv("MCP_FILESYSTEM_COMMAND", "")

	var output bytes.Buffer
	err := run(&output)
	if err == nil {
		t.Fatal("run succeeded with invalid configuration")
	}
	if strings.Contains(err.Error(), "topsecret-xyz") {
		t.Errorf("run error leaked the secret: %v", err)
	}
	if strings.Contains(output.String(), "topsecret-xyz") {
		t.Errorf("run output leaked the secret: %q", output.String())
	}
}

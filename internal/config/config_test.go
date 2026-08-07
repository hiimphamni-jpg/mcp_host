package config_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"mcp-host/internal/config"
)

func validEnv(t *testing.T) {
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

func TestConfigLoad_Valid(t *testing.T) {
	validEnv(t)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned an error for valid env: %v", err)
	}
	if got.GeminiAPIKey != "test-secret" {
		t.Errorf("GeminiAPIKey = %q, want test-secret", got.GeminiAPIKey)
	}
	if got.GeminiModel != "gemini-2.0-flash" {
		t.Errorf("GeminiModel = %q", got.GeminiModel)
	}
	if got.MCPFilesystemCommand != "npx" {
		t.Errorf("MCPFilesystemCommand = %q", got.MCPFilesystemCommand)
	}
	if len(got.MCPFilesystemArgs) != 2 {
		t.Fatalf("MCPFilesystemArgs len = %d, want 2", len(got.MCPFilesystemArgs))
	}
	if got.MCPTimeout != 30*time.Second {
		t.Errorf("MCPTimeout = %v", got.MCPTimeout)
	}
	if got.LLMTimeout != 45*time.Second {
		t.Errorf("LLMTimeout = %v", got.LLMTimeout)
	}
	if got.MaxAgentIterations != 10 {
		t.Errorf("MaxAgentIterations = %d", got.MaxAgentIterations)
	}
	if got.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want info", got.LogLevel)
	}
}

func TestConfigLoad_MissingRequired(t *testing.T) {
	validEnv(t)
	t.Setenv("GEMINI_API_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but GEMINI_API_KEY is required")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("error should mention GEMINI_API_KEY, got: %v", err)
	}
}

func TestConfigLoad_MalformedArgsJSON(t *testing.T) {
	validEnv(t)
	t.Setenv("MCP_FILESYSTEM_ARGS_JSON", `["unterminated`)

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but MCP_FILESYSTEM_ARGS_JSON is malformed")
	}
	if !strings.Contains(err.Error(), "MCP_FILESYSTEM_ARGS_JSON") {
		t.Errorf("error should mention MCP_FILESYSTEM_ARGS_JSON, got: %v", err)
	}
}

func TestConfigLoad_BadDuration(t *testing.T) {
	validEnv(t)
	t.Setenv("MCP_TIMEOUT", "not-a-duration")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but MCP_TIMEOUT is invalid")
	}
	if !strings.Contains(err.Error(), "MCP_TIMEOUT") {
		t.Errorf("error should mention MCP_TIMEOUT, got: %v", err)
	}
}

func TestConfigLoad_ZeroOrNegativeRejected(t *testing.T) {
	validEnv(t)
	t.Setenv("AGENT_MAX_ITERATIONS", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but AGENT_MAX_ITERATIONS=0 is invalid")
	}
	if !strings.Contains(err.Error(), "AGENT_MAX_ITERATIONS") {
		t.Errorf("error should mention AGENT_MAX_ITERATIONS, got: %v", err)
	}
}

func TestConfigLoad_SecretNotLeaked(t *testing.T) {
	validEnv(t)
	t.Setenv("GEMINI_API_KEY", "supersecret-abc123")
	t.Setenv("MCP_FILESYSTEM_COMMAND", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for missing required field")
	}
	if strings.Contains(err.Error(), "supersecret-abc123") {
		t.Errorf("error leaked the secret: %v", err)
	}
}

func TestConfigLoad_BadInt(t *testing.T) {
	validEnv(t)
	t.Setenv("MCP_MAX_RESULT_BYTES", "not-an-int")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but MCP_MAX_RESULT_BYTES is invalid")
	}
	if !strings.Contains(err.Error(), "MCP_MAX_RESULT_BYTES") {
		t.Errorf("error should mention MCP_MAX_RESULT_BYTES, got: %v", err)
	}
}

func TestConfigLoad_GatewayDefaults(t *testing.T) {
	validEnv(t)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if got.GatewayAddr != ":8080" {
		t.Errorf("default GatewayAddr = %q, want :8080", got.GatewayAddr)
	}
	if got.GatewayMaxConcurrent != 8 {
		t.Errorf("default GatewayMaxConcurrent = %d, want 8", got.GatewayMaxConcurrent)
	}
}

func TestConfigLoad_GatewayOverrides(t *testing.T) {
	validEnv(t)
	t.Setenv("GATEWAY_ADDR", ":9090")
	t.Setenv("GATEWAY_API_TOKEN", "primary-token")
	t.Setenv("GATEWAY_API_TOKENS", "tok-a,tok-b")
	t.Setenv("GATEWAY_MAX_CONCURRENT", "16")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if got.GatewayAddr != ":9090" {
		t.Errorf("GatewayAddr = %q, want :9090", got.GatewayAddr)
	}
	if got.GatewayAPIToken != "primary-token" {
		t.Errorf("GatewayAPIToken = %q", got.GatewayAPIToken)
	}
	if len(got.GatewayAPITokens) != 2 || got.GatewayAPITokens[0] != "tok-a" || got.GatewayAPITokens[1] != "tok-b" {
		t.Errorf("unexpected GatewayAPITokens = %v", got.GatewayAPITokens)
	}
	if got.GatewayMaxConcurrent != 16 {
		t.Errorf("GatewayMaxConcurrent = %d, want 16", got.GatewayMaxConcurrent)
	}
}

func TestConfigLoad_BadGatewayConcurrent(t *testing.T) {
	validEnv(t)
	t.Setenv("GATEWAY_MAX_CONCURRENT", "zero")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load succeeded but GATEWAY_MAX_CONCURRENT is invalid")
	}
	if !strings.Contains(err.Error(), "GATEWAY_MAX_CONCURRENT") {
		t.Errorf("error should mention GATEWAY_MAX_CONCURRENT, got: %v", err)
	}
}

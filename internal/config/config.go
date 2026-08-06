package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey         string
	GeminiModel          string
	MCPFilesystemCommand string
	MCPFilesystemArgs    []string
	MCPAllowedRoots      []string
	MCPTimeout           time.Duration
	LLMTimeout           time.Duration
	MaxAgentIterations   int
	MaxResultBytes       int
	LogLevel             string
}

const defaultLogLevel = "info"

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          os.Getenv("GEMINI_MODEL"),
		MCPFilesystemCommand: os.Getenv("MCP_FILESYSTEM_COMMAND"),
		MCPAllowedRoots:      splitList(os.Getenv("MCP_ALLOWED_ROOTS")),
		LogLevel:             os.Getenv("HOST_LOG_LEVEL"),
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaultLogLevel
	}

	var errs []error

	if raw := os.Getenv("MCP_FILESYSTEM_ARGS_JSON"); raw != "" {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			errs = append(errs, fmt.Errorf("MCP_FILESYSTEM_ARGS_JSON must be a JSON array of strings: %w", err))
		} else {
			cfg.MCPFilesystemArgs = args
		}
	}

	if v := os.Getenv("MCP_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("MCP_TIMEOUT must be a valid duration: %w", err))
		} else {
			cfg.MCPTimeout = d
		}
	}

	if v := os.Getenv("LLM_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("LLM_TIMEOUT must be a valid duration: %w", err))
		} else {
			cfg.LLMTimeout = d
		}
	}

	if v, err := strconv.Atoi(os.Getenv("AGENT_MAX_ITERATIONS")); err == nil {
		cfg.MaxAgentIterations = v
	} else if os.Getenv("AGENT_MAX_ITERATIONS") != "" {
		errs = append(errs, fmt.Errorf("AGENT_MAX_ITERATIONS must be an integer: %w", err))
	}

	if v, err := strconv.Atoi(os.Getenv("MCP_MAX_RESULT_BYTES")); err == nil {
		cfg.MaxResultBytes = v
	} else if os.Getenv("MCP_MAX_RESULT_BYTES") != "" {
		errs = append(errs, fmt.Errorf("MCP_MAX_RESULT_BYTES must be an integer: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		errs = append(errs, err)
	}

	return cfg, errors.Join(errs...)
}

func (c *Config) Validate() error {
	var errs []error

	if c.GeminiAPIKey == "" {
		errs = append(errs, errors.New("GEMINI_API_KEY is required"))
	}
	if c.GeminiModel == "" {
		errs = append(errs, errors.New("GEMINI_MODEL is required"))
	}
	if c.MCPFilesystemCommand == "" {
		errs = append(errs, errors.New("MCP_FILESYSTEM_COMMAND is required"))
	}
	if len(c.MCPFilesystemArgs) == 0 {
		errs = append(errs, errors.New("MCP_FILESYSTEM_ARGS_JSON must contain at least one argument"))
	}
	if len(c.MCPAllowedRoots) == 0 {
		errs = append(errs, errors.New("MCP_ALLOWED_ROOTS is required"))
	}
	if c.MCPTimeout <= 0 {
		errs = append(errs, errors.New("MCP_TIMEOUT must be positive"))
	}
	if c.LLMTimeout <= 0 {
		errs = append(errs, errors.New("LLM_TIMEOUT must be positive"))
	}
	if c.MaxAgentIterations <= 0 {
		errs = append(errs, errors.New("AGENT_MAX_ITERATIONS must be positive"))
	}
	if c.MaxResultBytes <= 0 {
		errs = append(errs, errors.New("MCP_MAX_RESULT_BYTES must be positive"))
	}

	return errors.Join(errs...)
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

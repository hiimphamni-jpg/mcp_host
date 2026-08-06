package main

import (
	"fmt"
	"io"
	"os"

	"mcp-host/internal/config"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	if _, err := config.Load(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	_, err := fmt.Fprintln(out, "MCP Host bootstrap complete. Integrations are not configured or started.")
	if err != nil {
		return fmt.Errorf("write bootstrap status: %w", err)
	}
	return nil
}

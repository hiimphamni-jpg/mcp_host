package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-host: unable to write bootstrap status")
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	_, err := fmt.Fprintln(out, "MCP Host bootstrap complete. Integrations are not configured or started.")
	if err != nil {
		return fmt.Errorf("write bootstrap status: %w", err)
	}
	return nil
}

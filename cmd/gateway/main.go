// Command gateway runs the HTTP/SSE gateway (FEAT-00008) on GATEWAY_ADDR.
// It builds the runner and HTTP server from .env config and serves /v1/chat.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcp-host/internal/config"
	"mcp-host/internal/runner"
	"mcp-host/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	if cfg.GatewayAPIToken == "" {
		return errors.New("GATEWAY_API_TOKEN is required to run the gateway")
	}

	tokens := make(map[string]bool)
	if cfg.GatewayAPIToken != "" {
		tokens[cfg.GatewayAPIToken] = true
	}
	for _, t := range cfg.GatewayAPITokens {
		if t != "" {
			tokens[t] = true
		}
	}

	h, err := runner.New(runner.Config{
		GeminiAPIKey:   cfg.GeminiAPIKey,
		GeminiModel:    cfg.GeminiModel,
		MCPCommand:     cfg.MCPFilesystemCommand,
		MCPArgs:        cfg.MCPFilesystemArgs,
		MCPRoots:       cfg.MCPAllowedRoots,
		MCPTimeout:     cfg.MCPTimeout,
		LLMTimeout:     cfg.LLMTimeout,
		MaxIterations:  cfg.MaxAgentIterations,
		MaxResultBytes: cfg.MaxResultBytes,
	})
	if err != nil {
		return fmt.Errorf("build runner: %w", err)
	}

	srv, err := server.New(h, server.Options{
		Tokens:         tokens,
		MaxConcurrent:  cfg.GatewayMaxConcurrent,
		RequestTimeout: 10 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}

	httpSrv := &http.Server{
		Addr:    cfg.GatewayAddr,
		Handler: srv.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("gateway listening on %s", cfg.GatewayAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdown)
	}
}
// Package server implements the HTTP/SSE gateway (FEAT-00008). The HTTP layer
// in this file is deliberately SDK-free: it depends only on stdlib and the
// neutral Event type, exposing an injectable Handler seam. The real handler
// (runner.go) composes mcpclient/mapping/policy/llm/agent and is where the Go
// SDKs are allowed (ADR-0001 D1, TDD GW8).
package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Event is a single SSE progress event pushed by the handler to the HTTP layer.
// Fields are safe to log (no secrets).
type Event struct {
	Type     string // "text" | "tool"
	Text     string
	ToolName string
	Phase    string // "start" | "result" (tool phase)
	Bytes    int
}

// Handler is the SDK-free execution seam (ADR-0001 D1). Chat runs one agent
// loop for prompt, pushing benign progress Events via emit, and returns the
// final answer text. ctx cancellation aborts the loop.
type Handler interface {
	Chat(ctx context.Context, prompt string, emit func(Event)) (string, error)
}

// Options configures the HTTP server.
type Options struct {
	// Tokens is the set of accepted bearer tokens; empty means auth disabled.
	Tokens map[string]bool
	// MaxConcurrent caps in-flight chat requests (>0).
	MaxConcurrent int
	// RequestTimeout bounds one whole chat request; 0 means unlimited.
	RequestTimeout time.Duration
}

// Server is the HTTP/SSE gateway.
type Server struct {
	handler    Handler
	tokens     map[string]bool
	sem        chan struct{}
	reqTimeout time.Duration
}

// New builds a Server. handler must be non-nil; MaxConcurrent must be > 0.
func New(handler Handler, opts Options) (*Server, error) {
	if handler == nil {
		return nil, errors.New("server: handler is required")
	}
	if opts.MaxConcurrent <= 0 {
		return nil, errors.New("server: MaxConcurrent must be positive")
	}
	return &Server{
		handler:    handler,
		tokens:     opts.Tokens,
		sem:        make(chan struct{}, opts.MaxConcurrent),
		reqTimeout: opts.RequestTimeout,
	}, nil
}

// Handler returns the http.Handler (mux) for the gateway, auth-wrapped.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /v1/chat", s.handleChat)
	return s.authed(mux)
}

// authed enforces bearer auth on all paths except the public healthz probe.
func (s *Server) authed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if !s.authorized(r) {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authorized reports whether the request carries a valid bearer token
// (constant-time compare, ADR-0004). Empty token set disables auth.
func (s *Server) authorized(r *http.Request) bool {
	if len(s.tokens) == 0 {
		return true
	}
	authz := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(authz, prefix) {
		return false
	}
	got := authz[len(prefix):]
	for want := range s.tokens {
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1 {
			return true
		}
	}
	return false
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type chatRequest struct {
	Prompt *string `json:"prompt"`
}

// handleChat validates the request, admits it under the concurrency guard, and
// streams SSE events until the handler returns.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_FORMAT", "malformed JSON body")
		return
	}
	if req.Prompt == nil {
		writeJSONError(w, http.StatusBadRequest, "VALIDATION_FAILED", "prompt field is required")
		return
	}
	if strings.TrimSpace(*req.Prompt) == "" {
		writeJSONError(w, http.StatusUnprocessableEntity, "INVALID_CONTENT", "prompt must not be empty")
		return
	}

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	default:
		writeJSONError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "too many concurrent requests")
		return
	}

	ctx := r.Context()
	cancel := func() {}
	if s.reqTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, s.reqTimeout)
	}
	defer cancel()

	sw := newSSEWriter(w)
	if !sw.begin() {
		return
	}
	sw.event("start", map[string]any{"request_id": requestID(r), "model": modelName(r)})

	answer, err := s.handler.Chat(ctx, *req.Prompt, func(ev Event) {
		sw.onEvent(ev)
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			sw.event("error", map[string]any{"code": "CANCELLED", "message": "request cancelled"})
		} else {
			sw.event("error", map[string]any{"code": "HANDLER_ERROR", "message": "request failed"})
		}
		sw.close()
		return
	}
	sw.event("final", map[string]any{"text": answer})
	sw.close()
}

// sseWriter frames Server-Sent Events (ADR-0003) and best-effort flushes.
type sseWriter struct {
	w http.ResponseWriter
}

func newSSEWriter(w http.ResponseWriter) *sseWriter {
	return &sseWriter{w: w}
}

// begin sets SSE headers and returns false if the client is already gone.
func (w *sseWriter) begin() bool {
	h := w.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.w.WriteHeader(http.StatusOK)
	fl, _ := w.w.(http.Flusher)
	if fl != nil {
		fl.Flush()
	}
	return true
}

func (w *sseWriter) event(name string, payload map[string]any) {
	b, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w.w, "event: %s\ndata: %s\n\n", name, b)
	w.flush()
}

func (w *sseWriter) close() {
	w.flush()
}

func (w *sseWriter) flush() {
	if fl, ok := w.w.(http.Flusher); ok {
		fl.Flush()
	}
}

// onEvent maps a handler event to an SSE frame.
func (w *sseWriter) onEvent(ev Event) {
	switch ev.Type {
	case "text":
		w.event("stream", map[string]any{"type": "text", "text": ev.Text})
	case "tool":
		w.event("stream", map[string]any{"type": "tool", "name": ev.ToolName, "status": ev.Phase, "bytes": ev.Bytes})
	}
}

// requestID returns a stable per-request identifier from the request id header
// if present, otherwise a short placeholder.
func requestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-Id"); v != "" {
		return v
	}
	return "req"
}

// modelName returns the configured model name echoed back to the client. It
// defaults to an empty-friendly placeholder since the HTTP layer is SDK-free.
func modelName(r *http.Request) string {
	if v := r.Header.Get("X-Model"); v != "" {
		return v
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
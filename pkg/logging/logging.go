// Package logging provides a structured slog logger and trace-ID propagation
// shared by all MEMBRANE.AI services. JSON output; never log secrets or raw
// source code (see ENGINEERING-STANDARDS §5).
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

type ctxKey struct{}

// New returns a JSON slog.Logger writing to w at the given level
// ("debug"|"info"|"warn"|"error"; defaults to info on an unknown value).
func New(level string, w io.Writer) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithTraceID returns a context carrying the trace ID.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// TraceID extracts the trace ID from ctx, or "" if absent.
func TraceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// FromContext returns base enriched with the context's trace_id attribute when
// present, so every log line in a request is correlatable.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if id := TraceID(ctx); id != "" {
		return base.With(slog.String("trace_id", id))
	}
	return base
}

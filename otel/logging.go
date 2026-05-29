package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// TraceIDLogger wraps an *slog.Logger to inject the trace_id and span_id
// from the context into every log entry. Since slog doesn't support
// per-call middleware, use ContextLogger(logger, ctx) for per-entry
// trace injection, or use the returned logger with InfoContext.
//
// This adds "component"="cqrs" as a static field.
func TraceIDLogger(logger *slog.Logger) *slog.Logger {
	return logger.With(slog.String("component", "cqrs"))
}

// TraceIDFromContext extracts the trace ID from the context. Returns "none"
// if no span is active.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()

	if !sc.IsValid() {
		return "none"
	}

	return sc.TraceID().String()
}

// SpanIDFromContext extracts the span ID from the context. Returns "none"
// if no span is active.
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()

	if !sc.IsValid() {
		return "none"
	}

	return sc.SpanID().String()
}

// ContextLogger returns an *slog.Logger that includes trace_id and span_id
// from the given context. If no span is active, fields are set to "none".
func ContextLogger(logger *slog.Logger, ctx context.Context) *slog.Logger {
	return logger.With(
		slog.String("trace_id", TraceIDFromContext(ctx)),
		slog.String("span_id", SpanIDFromContext(ctx)),
	)
}

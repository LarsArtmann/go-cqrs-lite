package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// TraceIDLogger wraps an *slog.Logger to automatically inject the trace ID
// from the context into every log entry. If no span is active, the trace_id
// field is set to "none".
//
// Usage:
//
//	logger := slog.Default()
//	tLogger := otel.TraceIDLogger(logger)
//	tLogger.InfoContext(ctx, "handling command", "command_type", cmd.Type())
//	// Output includes trace_id=<hex> when a span is active
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

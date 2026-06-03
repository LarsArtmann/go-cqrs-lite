package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// ComponentLogger returns an *slog.Logger with "component"="cqrs" as a static
// field. For per-entry trace_id/span_id injection, use ContextLogger(logger, ctx).
func ComponentLogger(logger *slog.Logger) *slog.Logger {
	return logger.With(slog.String("component", "cqrs"))
}

// TraceIDFromContext extracts the trace ID from the context. Returns "none"
// if no span is active.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	if !spanCtx.IsValid() {
		return "none"
	}

	return spanCtx.TraceID().String()
}

// SpanIDFromContext extracts the span ID from the context. Returns "none"
// if no span is active.
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	if !spanCtx.IsValid() {
		return "none"
	}

	return spanCtx.SpanID().String()
}

// ContextLogger returns an *slog.Logger that includes trace_id and span_id
// from the given context. If no span is active, fields are set to "none".
func ContextLogger(logger *slog.Logger, ctx context.Context) *slog.Logger {
	return logger.With(
		slog.String("trace_id", TraceIDFromContext(ctx)),
		slog.String("span_id", SpanIDFromContext(ctx)),
	)
}

package otel_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/otel"
)

func newTestTracerProvider() *sdktrace.TracerProvider {
	recorder := tracetest.NewSpanRecorder()

	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
}

func TestTraceIDFromContext_NoSpan(t *testing.T) {
	id := otel.TraceIDFromContext(context.Background())
	require.Equal(t, "none", id)
}

func TestSpanIDFromContext_NoSpan(t *testing.T) {
	id := otel.SpanIDFromContext(context.Background())
	require.Equal(t, "none", id)
}

func TestTraceIDFromContext_WithSpan(t *testing.T) {
	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	id := otel.TraceIDFromContext(ctx)
	require.NotEqual(t, "none", id)
	require.Len(t, id, 32)
}

func TestSpanIDFromContext_WithSpan(t *testing.T) {
	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	id := otel.SpanIDFromContext(ctx)
	require.NotEqual(t, "none", id)
	require.Len(t, id, 16)
}

func TestContextLogger_NoSpan(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cl := otel.ContextLogger(logger, context.Background())
	cl.Info("test message")

	output := buf.String()
	require.Contains(t, output, "trace_id=none")
	require.Contains(t, output, "span_id=none")
}

func TestContextLogger_WithSpan(t *testing.T) {
	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cl := otel.ContextLogger(logger, ctx)
	cl.Info("test message")

	output := buf.String()
	require.NotContains(t, output, "trace_id=none")
}

func TestTraceIDLogger(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	wrapped := otel.TraceIDLogger(logger)
	require.NotNil(t, wrapped)

	wrapped.Info("test message")
	require.Contains(t, buf.String(), "component=cqrs")
}

package otel_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/otel"
)

func newTestTracerProvider() *sdktrace.TracerProvider {
	recorder := tracetest.NewSpanRecorder()

	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
}

func TestTraceIDFromContext_NoSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(otel.TraceIDFromContext(context.Background())).To(Equal("none"))
}

func TestSpanIDFromContext_NoSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(otel.SpanIDFromContext(context.Background())).To(Equal("none"))
}

func TestTraceIDFromContext_WithSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	id := otel.TraceIDFromContext(ctx)
	g.Expect(id).ToNot(Equal("none"))
	g.Expect(id).To(HaveLen(32))
}

func TestSpanIDFromContext_WithSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	id := otel.SpanIDFromContext(ctx)
	g.Expect(id).ToNot(Equal("none"))
	g.Expect(id).To(HaveLen(16))
}

func TestContextLogger_NoSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cl := otel.ContextLogger(logger, context.Background())
	cl.Info("test message")

	output := buf.String()
	g.Expect(output).To(ContainSubstring("trace_id=none"))
	g.Expect(output).To(ContainSubstring("span_id=none"))
}

func TestContextLogger_WithSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tp := newTestTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cl := otel.ContextLogger(logger, ctx)
	cl.Info("test message")

	output := buf.String()
	g.Expect(output).ToNot(ContainSubstring("trace_id=none"))
}

func TestTraceIDLogger(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	wrapped := otel.TraceIDLogger(logger)
	g.Expect(wrapped).ToNot(BeNil())

	wrapped.Info("test message")
	g.Expect(buf.String()).To(ContainSubstring("component=cqrs"))
}

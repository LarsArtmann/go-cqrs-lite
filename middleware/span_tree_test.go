package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// TestSpanTree_RetryAttemptsAreChildren verifies that a retried command
// produces the correct span tree: one parent command.handle span with child
// retry.attempt.N spans for each attempt. This is the critical validation
// that distributed tracing shows individual retry attempts, not one opaque
// span covering all retries.
func TestSpanTree_RetryAttemptsAreChildren(t *testing.T) {
	// NOT parallel — mutates global TracerProvider
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	mp := metric.NewMeterProvider()

	bundle, err := NewOTelBundle(tp.Tracer("test"), mp.Meter("test"))
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	callCount := 0

	cmdMws := bundle.Command()
	retryMw := CommandRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Millisecond,
		Multiplier:   2.0,
		IsRetryable:  func(err error) bool { return true },
	})

	handlerFunc := func(_ context.Context, _ command.Command) error {
		callCount++

		return errors.New("always fails")
	}

	handler := cmdMws[0](cmdMws[1](retryMw(handlerFunc)))
	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	_ = handler(context.Background(), cmd)

	tp.ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) < 4 {
		t.Fatalf("expected at least 4 spans (1 command + 3 attempts), got %d", len(spans))
	}

	var parentSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "command.handle" {
			parentSpan = &spans[i]
		}
	}
	if parentSpan == nil {
		t.Fatal("expected command.handle parent span")
	}

	childCount := 0
	for _, span := range spans {
		if span.Parent.SpanID() == parentSpan.SpanContext.SpanID() {
			childCount++
		}
	}
	if childCount < 3 {
		t.Errorf("expected at least 3 child spans under command.handle, got %d", childCount)
	}
	if callCount != 3 {
		t.Errorf("expected 3 handler calls, got %d", callCount)
	}
}

// TestSpanTree_BundleProducesAllSpanKinds verifies the bundle creates
// spans for both command and query handlers.
func TestSpanTree_BundleProducesAllSpanKinds(t *testing.T) {
	// NOT parallel — mutates global TracerProvider
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	mp := metric.NewMeterProvider()
	bundle, err := NewOTelBundle(tp.Tracer("test"), mp.Meter("test"))
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	cmdMws := bundle.Command()
	cmdHandler := cmdMws[0](cmdMws[1](NoopCommandHandler()))
	_ = cmdHandler(context.Background(), &testCommand{aggregateID: id.NewAggregateID()})

	qryMws := bundle.Query()
	qryHandler := qryMws[0](qryMws[1](noopQueryHandler()))
	_, _ = qryHandler(context.Background(), &testQuery{})

	tp.ForceFlush(context.Background())

	spans := exporter.GetSpans()
	names := make(map[string]bool)
	for _, span := range spans {
		names[span.Name] = true
	}
	if !names["command.handle"] {
		t.Error("expected command.handle span")
	}
	if !names["query.handle"] {
		t.Error("expected query.handle span")
	}
}

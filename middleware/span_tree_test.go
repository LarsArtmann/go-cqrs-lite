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

// TestSpanTree_CommandWithRetry verifies that a retried command produces
// the correct span tree: one parent command.handle span with child
// retry.attempt.N spans for each attempt.
func TestSpanTree_CommandWithRetry(t *testing.T) {
	// NOT parallel — mutates global TracerProvider

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

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
		if callCount < 2 {
			return errors.New("transient failure")
		}

		return nil
	}

	handler := cmdMws[0](cmdMws[1](retryMw(handlerFunc)))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err = handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tp.ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) < 2 {
		t.Fatalf("expected at least 2 spans (1 command.handle + 1+ retry.attempt), got %d", len(spans))
	}

	names := make(map[string]int)

	for _, span := range spans {
		names[span.Name]++
	}

	if names["command.handle"] != 1 {
		t.Errorf("expected 1 command.handle span, got %d", names["command.handle"])
	}

	totalAttempts := 0
	for name, count := range names {
		if len(name) > 13 && name[:13] == "retry.attempt." {
			totalAttempts += count
		}
	}

	if totalAttempts != 2 {
		t.Errorf("expected 2 retry.attempt spans (1 fail + 1 success), got %d", totalAttempts)
	}

	if callCount != 2 {
		t.Errorf("expected 2 handler calls, got %d", callCount)
	}
}

// TestSpanTree_CommandTracingProducesChildSpans verifies that the tracing
// middleware creates a server span and the retry middleware's attempt spans
// are children of it (linked via parent span context).
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
		t.Fatal("expected to find command.handle parent span")
	}

	childCount := 0
	for _, span := range spans {
		if span.Parent.SpanID() == parentSpan.SpanContext.SpanID() {
			childCount++
		}
	}

	if childCount != 3 {
		t.Errorf("expected 3 child spans (retry attempts) under command.handle, got %d", childCount)
	}
}

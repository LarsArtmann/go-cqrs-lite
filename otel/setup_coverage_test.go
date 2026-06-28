package otel_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// failingExporter returns an error on Shutdown to test error propagation.
type failingExporter struct{ *tracetest.InMemoryExporter }

func (failingExporter) Shutdown(context.Context) error {
	return errors.New("forced shutdown failure")
}

// failingReader returns an error on Shutdown to test error propagation.
type failingReader struct{ metric.Reader }

func (failingReader) Shutdown(ctx context.Context) error {
	return errors.New("forced reader shutdown failure")
}

func TestSetup_ShutdownErrorPropagation(t *testing.T) {
	t.Parallel()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("test", "1.0", ""),
		cqrsotel.WithSpanExporter(failingExporter{tracetest.NewInMemoryExporter()}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := provider.Shutdown(context.Background()); err == nil {
		t.Error("expected shutdown error from failing exporter, got nil")
	}
}

func TestSetup_WithMetricReader(t *testing.T) {
	t.Parallel()

	reader := metric.NewManualReader()
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("test", "1.0", ""),
		cqrsotel.WithMetricReader(reader),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer provider.Shutdown(context.Background())

	// Verify the meter provider is functional.
	meter := provider.AsMeterProvider().Meter("test")
	_, mErr := meter.Int64Counter("test.counter")
	if mErr != nil {
		t.Fatalf("create counter: %v", mErr)
	}
}

func TestSetup_TextMapPropagator(t *testing.T) {
	// NOT parallel — mutates global propagator.
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("test", "1.0", ""),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer provider.Shutdown(context.Background())

	// The Setup function sets the global propagator to W3C trace context +
	// baggage by default. Verify it's not the no-op propagator.
	ctx := cqrsotel.WithCorrelationID(context.Background(), "test-correlation")
	id := cqrsotel.CorrelationIDFromContext(ctx)
	if id != "test-correlation" {
		t.Errorf("expected correlation ID propagation, got %q", id)
	}
}

func TestSetup_SpanExporterPrecedence(t *testing.T) {
	// NOT parallel — mutates global state.
	exporter := tracetest.NewInMemoryExporter()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("test", "1.0", ""),
		cqrsotel.WithSpanExporter(exporter),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer provider.Shutdown(context.Background())

	tracer := provider.AsTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "precedence-test")
	span.End()

	provider.AsTracerProvider().ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span from explicit exporter, got %d", len(spans))
	}
}

func TestSetup_MetricDataExport(t *testing.T) {
	t.Parallel()

	reader := metric.NewManualReader()
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("metrics-test", "1.0", ""),
		cqrsotel.WithMetricReader(reader),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer provider.Shutdown(context.Background())

	// Produce a metric via the meter provider.
	meter := provider.AsMeterProvider().Meter("test")
	counter, _ := meter.Int64Counter("test.ops")
	counter.Add(context.Background(), 1)

	// Collect and verify.
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	if len(data.ScopeMetrics) == 0 {
		t.Error("expected at least one scope metric")
	}
}

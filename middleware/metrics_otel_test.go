package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

func TestOTelMetricsRecorder(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder.Observe(context.Background(), "test_op", 100, "type", "test", "status", "success")
}

func TestCommandOTelMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram("cqrs.command.duration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandOTelMetrics(h)
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err = handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventOTelMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram("cqrs.event.duration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := EventOTelMetrics(h)
	handler := mw(eventtest.NoopEventHandler())

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryOTelMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram("cqrs.query.duration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := QueryOTelMetrics(h)
	handler := mw(noopQueryHandler())

	_, err = handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandOTelMetrics_RecordsError(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram("cqrs.command.duration")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandOTelMetrics(h)
	handler := mw(failingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err = handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOTelMetricsRecorder_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ MetricsRecorder = (*OTelMetricsRecorder)(nil)

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var iface MetricsRecorder = recorder

	iface.Observe(context.Background(), "test", 100, "key", "value")
}

func TestCommandOTelMetrics_CollectsData(t *testing.T) {
	t.Parallel()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram("cqrs.command.duration.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandOTelMetrics(h)
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err = handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resourceMetrics metricdata.ResourceMetrics

	err = reader.Collect(context.Background(), &resourceMetrics)
	if err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	if len(resourceMetrics.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics to be collected")
	}

	found := false

	for _, sm := range resourceMetrics.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "cqrs.command.duration.test" {
				found = true

				break
			}
		}
	}

	if !found {
		t.Error("expected to find 'cqrs.command.duration.test' metric")
	}
}

var (
	_ = sdktrace.NewTracerProvider()
	_ = cqrsotel.Name
)

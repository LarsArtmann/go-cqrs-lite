package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestOTelMetricsRecorder(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder.Observe("test_op", 100, "type", "test", "status", "success")
}

func TestCommandOTelMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram(metricNameCommandDuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandOTelMetrics(h)
	handler := mw(testhelpers.NoopCommandHandler())

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

	h, err := meter.Float64Histogram(metricNameEventDuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := EventOTelMetrics(h)
	handler := mw(testhelpers.NoopEventHandler())

	evt, err := testhelpers.NewTestEvent()
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

	h, err := meter.Float64Histogram(metricNameQueryDuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := QueryOTelMetrics(h)
	handler := mw(testhelpers.NoopQueryHandler())

	_, err = handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandOTelMetrics_RecordsError(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	h, err := meter.Float64Histogram(metricNameCommandDuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandOTelMetrics(h)
	handler := mw(testhelpers.FailingCommandHandler("boom"))

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

	iface.Observe("test", 100, "key", "value")
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
	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err = handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rm metricdata.ResourceMetrics

	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics to be collected")
	}

	found := false

	for _, sm := range rm.ScopeMetrics {
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

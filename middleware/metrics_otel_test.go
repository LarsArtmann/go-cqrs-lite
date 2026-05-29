package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

func TestOTelMetricsRecorder_PanicsOnNilHistogram(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic when calling Observe on nil recorder")
		}
	}()

	var recorder OTelMetricsRecorder
	recorder.Observe("test", 100)
}

func TestNewOTelMetricsRecorder_Success(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}

	if recorder.histogram == nil {
		t.Fatal("expected non-nil histogram")
	}
}

func collectMetrics(t *testing.T, provider *metric.MeterProvider) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics

	err := provider.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	return rm
}

func TestOTelMetricsRecorder_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ MetricsRecorder = (*OTelMetricsRecorder)(nil)
	var _ MetricsRecorder = MetricsRecorder(nil)

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var iface MetricsRecorder = recorder

	iface.Observe("test", 100, "key", "value")
}

// Verify the metric provider works end-to-end with the OTel SDK.
func TestCommandOTelMetrics_CollectsData(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
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

	rm := collectMetrics(t, provider)

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

// Unused but keeps the import for sdktrace available if needed later.
var _ = sdktrace.NewTracerProvider()

// Unused but keeps the import for cqrsotel available.
var _ = cqrsotel.Name

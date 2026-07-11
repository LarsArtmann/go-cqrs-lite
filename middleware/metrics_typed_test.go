package middleware

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func TestOTelMetricsRecorder_ImplementsTypedInterface(t *testing.T) {
	t.Parallel()

	var _ TypedMetricsRecorder = (*OTelMetricsRecorder)(nil)

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typed TypedMetricsRecorder = recorder
	typed.ObserveTyped(
		context.Background(), "command", 50,
		cqrsotel.AttrString(cqrsotel.AttrCommandType, "user.create"),
		cqrsotel.AttrString(cqrsotel.AttrStatus, cqrsotel.StatusSuccess),
	)
}

func TestCommandTypedMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandTypedMetrics(recorder)
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	if err := handler(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandTypedMetrics_RecordsError(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := CommandTypedMetrics(recorder)
	handler := mw(failingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	if err := handler(context.Background(), cmd); err == nil {
		t.Fatal("expected error")
	}
}

func TestEventTypedMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := EventTypedMetrics(recorder)
	handler := mw(eventtest.NoopEventHandler())

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := handler(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryTypedMetrics(t *testing.T) {
	t.Parallel()

	provider := metric.NewMeterProvider()
	meter := provider.Meter("test")

	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mw := QueryTypedMetrics(recorder)
	handler := mw(noopQueryHandler())

	if _, err := handler(context.Background(), &testQuery{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// fakeTypedRecorder captures ObserveTyped calls for assertion.
type fakeTypedRecorder struct {
	operations []string
	attrs      [][]cqrsotel.KeyValue
}

func (f *fakeTypedRecorder) ObserveTyped(
	_ context.Context,
	operation string,
	_ time.Duration,
	attrs ...cqrsotel.KeyValue,
) {
	f.operations = append(f.operations, operation)
	f.attrs = append(f.attrs, attrs)
}

func TestCommandTypedMetrics_CapturesAttrs(t *testing.T) {
	t.Parallel()

	rec := &fakeTypedRecorder{}
	mw := CommandTypedMetrics(rec)
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	if err := handler(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rec.operations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(rec.operations))
	}

	if rec.operations[0] != kindCommand {
		t.Errorf("expected operation %q, got %q", kindCommand, rec.operations[0])
	}

	wantAttrs := map[string]string{
		cqrsotel.AttrMessageKind: kindCommand,
		cqrsotel.AttrStatus:      cqrsotel.StatusSuccess,
	}

	got := make(map[string]string, len(rec.attrs[0]))

	for _, kv := range rec.attrs[0] {
		got[string(kv.Key)] = kv.Value.AsString()
	}

	for k, v := range wantAttrs {
		if got[k] != v {
			t.Errorf("attr %q = %q, want %q", k, got[k], v)
		}
	}
}

package http

import (
	"context"
	"testing"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// TestReplayMetrics_NilSafe verifies that a nil *ReplayMetrics is a no-op
// (no-opts all instrument calls). This is the default when WithReplayMetrics
// is not supplied.
func TestReplayMetrics_NilSafe(t *testing.T) {
	t.Parallel()

	var m *ReplayMetrics
	m.RecordReplay(context.Background(), 12.5, 100, false)
	m.RecordReplay(context.Background(), 0, 0, true)
}

// TestReplayMetrics_NilMeterReturnsNil verifies that passing a nil meter to
// NewReplayMetrics returns (nil, nil) — so callers can safely dereference the
// returned pointer (nil-safe methods).
func TestReplayMetrics_NilMeterReturnsNil(t *testing.T) {
	t.Parallel()

	m, err := NewReplayMetrics(nil)
	if err != nil {
		t.Fatalf("expected nil error for nil meter, got %v", err)
	}

	if m != nil {
		t.Fatalf("expected nil metrics for nil meter, got %+v", m)
	}
}

// TestReplayMetrics_RecordsReplay verifies the OTel instruments are created
// without error and that RecordReplay accepts the value shapes produced by
// the replay path. Asserting exact OTel values requires a manual reader; here
// we confirm the no-panic contract and the wiring of all three instruments.
func TestReplayMetrics_RecordsReplay(t *testing.T) {
	t.Parallel()

	meter := cqrsotel.NewMeter("test-sse-metrics")
	metrics, err := NewReplayMetrics(meter)
	if err != nil {
		t.Fatalf("NewReplayMetrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("expected non-nil metrics for non-nil meter")
	}

	// A replay of 0 events (empty journal) shouldn't record event count but
	// should still record duration.
	metrics.RecordReplay(context.Background(), 1.0, 0, false)

	// A real replay records events + duration.
	metrics.RecordReplay(context.Background(), 5.0, 42, false)

	// A timed-out replay increments the incomplete counter.
	metrics.RecordReplay(context.Background(), 10.0, 7, true)
}

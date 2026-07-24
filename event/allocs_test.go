package event

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestAllocations_HotPaths asserts deterministic allocation counts for critical paths.
// These tests catch allocation regressions that benchmarks with ±15% variance might miss.
// Update the expected values ONLY when intentionally changing the allocation behavior.

func TestAllocs_NewEvent_NoOptions(t *testing.T) {
	aggID := id.NewStreamID()
	payload := []byte(`{"name":"test","value":42}`)

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = NewEvent("test.created", aggID, "Test", Version(1), payload)
	})

	if allocs != 3 {
		t.Errorf(
			"NewEvent allocations = %v, want 3 (ImmutableEvent + payload clone + eventOptions)",
			allocs,
		)
	}
}

func TestAllocs_NewEvent_WithCorrelationID(t *testing.T) {
	aggID := id.NewStreamID()
	payload := []byte(`{"name":"test","value":42}`)
	corrID := id.NewCorrelationID()

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = NewEvent(
			"test.created", aggID, "Test", Version(1), payload,
			WithCorrelationID(corrID),
		)
	})

	if allocs != 3 {
		t.Errorf("NewEvent with CorrelationID allocations = %v, want 3", allocs)
	}
}

func TestAllocs_NewMetadata_Empty(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = NewMetadata()
	})

	if allocs != 0 {
		t.Errorf("NewMetadata allocations = %v, want 0 (lazy map init)", allocs)
	}
}

func TestAllocs_Classify(t *testing.T) {
	err := errorfamily.NewRejection("test.op", "test rejection")

	allocs := testing.AllocsPerRun(100, func() {
		_ = errorfamily.Classify(err)
	})

	if allocs != 0 {
		t.Errorf("Classify allocations = %v, want 0", allocs)
	}
}

func TestAllocs_FilterByTimestamp(t *testing.T) {
	aggID := id.NewStreamID()
	events := make([]Event, 100)
	for i := range events {
		evt, _ := NewEvent("test.updated", aggID, "Test", Version(i+1),
			[]byte(`{"name":"test","value":0}`))
		events[i] = evt
	}

	ts := events[50].OccurredAt()

	allocs := testing.AllocsPerRun(10, func() {
		_ = FilterByTimestamp(events, ts)
	})

	if allocs != 1 {
		t.Errorf("FilterByTimestamp allocations = %v, want 1 (result slice)", allocs)
	}
}

func TestAllocs_NewStreamRef(t *testing.T) {
	aggID := id.NewStreamID()
	aggType := id.StreamType("Test")

	allocs := testing.AllocsPerRun(100, func() {
		_ = id.NewStreamRef(aggType, aggID)
	})

	if allocs != 0 {
		t.Errorf("id.NewStreamRef allocations = %v, want 0", allocs)
	}
}

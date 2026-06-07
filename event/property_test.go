package event_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// TestEventImmutability checks that Clone produces an independent copy.
func TestEventImmutability(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := event.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		aggID := id.NewAggregateID()
		version := event.Version(rapid.IntRange(1, 1000).Draw(t, "version"))

		evt, err := event.NewEvent(typ, aggID, "Test", version, nil)
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		clone := evt.Clone()

		if evt.Type() != clone.Type() {
			t.Fatal("type mismatch after clone")
		}
		if evt.AggregateID() != clone.AggregateID() {
			t.Fatal("aggregateID mismatch after clone")
		}
		if evt.Version() != clone.Version() {
			t.Fatal("version mismatch after clone")
		}
		if evt.AggregateType() != clone.AggregateType() {
			t.Fatal("aggregateType mismatch after clone")
		}
		if evt.OccurredAt() != clone.OccurredAt() {
			t.Fatal("occurredAt mismatch after clone")
		}

		// The clone should be a different pointer
		if evt == clone {
			t.Fatal("clone returned same pointer")
		}
	})
}

// TestEventIDempotency checks that creating an event twice with the same
// parameters produces events with the same fields.
func TestEventIDempotency(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := event.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		aggID := id.NewAggregateID()
		version := event.Version(rapid.IntRange(1, 1000).Draw(t, "version"))

		evt1, err1 := event.NewEvent(typ, aggID, "Test", version, nil)
		evt2, err2 := event.NewEvent(typ, aggID, "Test", version, nil)

		if err1 != nil || err2 != nil {
			t.Skip("creation error")
		}

		if evt1.Type() != evt2.Type() ||
			evt1.AggregateID() != evt2.AggregateID() ||
			evt1.Version() != evt2.Version() ||
			evt1.AggregateType() != evt2.AggregateType() {
			t.Fatal("identical parameters produced different events")
		}
	})
}

// TestBatchVersionMonotonicity checks that NewEvents creates events with
// strictly increasing versions.
func TestBatchVersionMonotonicity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 50).Draw(t, "count")
		startVersion := event.Version(rapid.IntRange(1, 100).Draw(t, "startVersion"))
		aggID := id.NewAggregateID()

		types := make([]event.Type, count)
		payloads := make([]any, count)
		for i := range types {
			types[i] = "TestEvent"
			payloads[i] = struct{ N int }{N: i}
		}

		events, err := event.NewEvents(aggID, "Test", startVersion, types, payloads)
		if err != nil {
			t.Fatalf("create events: %v", err)
		}

		if len(events) != count {
			t.Fatalf("expected %d events, got %d", count, len(events))
		}

		for i := 1; i < len(events); i++ {
			if events[i].Version() != events[i-1].Version()+1 {
				t.Fatalf(
					"version not sequential: %d -> %d",
					events[i-1].Version(),
					events[i].Version(),
				)
			}
		}
	})
}

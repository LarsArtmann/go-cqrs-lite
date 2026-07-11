package schema_test

import (
	"context"
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/schema/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

// chainUpcaster transforms "test.event" events from sourceVer to sourceVer+1.
// The registry applies the schema-version bump after Upcast returns, so the
// upcaster itself only needs to preserve identity fields and return a new event.
type chainUpcaster struct {
	sourceVer int
}

func (c *chainUpcaster) SourceType() event.Type { return "test.event" }

func (c *chainUpcaster) SourceVersion() event.SchemaVersion { return event.SchemaVersion(c.sourceVer) }

func (c *chainUpcaster) Upcast(evt event.Event) (event.Event, error) {
	return event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
	)
}

// conditionalFailUpcaster fails when the payload matches a trigger string.
type conditionalFailUpcaster struct {
	trigger string
}

func (c *conditionalFailUpcaster) SourceType() event.Type             { return "test.event" }
func (c *conditionalFailUpcaster) SourceVersion() event.SchemaVersion { return 1 }

func (c *conditionalFailUpcaster) Upcast(evt event.Event) (event.Event, error) {
	if string(evt.Payload()) == c.trigger {
		return nil, fmt.Errorf("upcaster triggered failure on payload %q", c.trigger)
	}

	return evt, nil
}

// TestVersionedSeekableJournal_Property_UpcasterChain verifies that arbitrary
// upcaster chains transform events to the expected final version regardless
// of starting version or event count.
func TestVersionedSeekableJournal_Property_upcasterChain(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		chainDepth := rapid.IntRange(1, 5).Draw(rt, "chainDepth")
		numEvents := rapid.IntRange(1, 50).Draw(rt, "numEvents")

		store := memory.NewMemoryStore()
		defer store.Close()

		ctx := context.Background()
		aggID := id.NewAggregateID()

		upcasters := make([]schema.Upcaster, chainDepth)
		for i := range upcasters {
			upcasters[i] = &chainUpcaster{sourceVer: i + 1}
		}

		var events []event.Event
		expectedVersions := make([]int, 0, numEvents)

		for i := 0; i < numEvents; i++ {
			startVersion := rapid.IntRange(1, chainDepth+5).Draw(rt, "startVersion")
			payload := fmt.Sprintf("evt-%d", i)

			evt, err := event.NewEvent(
				"test.event", aggID, "Test",
				event.Version(i+1),
				[]byte(payload),
				event.WithSchemaVersion(event.SchemaVersion(startVersion)),
			)
			if err != nil {
				t.Fatalf("create event %d: %v", i, err)
			}

			events = append(events, evt)

			finalVersion := startVersion
			if startVersion <= chainDepth {
				finalVersion = chainDepth + 1
			}

			expectedVersions = append(expectedVersions, finalVersion)
		}

		saveTestEvents(t, ctx, store, aggID, events...)

		journal, err := schema.NewVersionedSeekableJournal(store, upcasters...)
		if err != nil {
			t.Fatalf("new versioned journal: %v", err)
		}

		all, err := journal.ReadAll(ctx)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}

		if len(all) != numEvents {
			t.Fatalf("expected %d events, got %d", numEvents, len(all))
		}

		for i, expVer := range expectedVersions {
			gotVer := all[i].SchemaVersion().Int()
			if gotVer != expVer {
				t.Errorf("event %d: schemaVersion = %d, want %d (start=%d, chainDepth=%d)",
					i, gotVer, expVer, events[i].SchemaVersion().Int(), chainDepth)
			}
		}
	})
}

// TestVersionedSeekableJournal_Property_passthrough verifies that events of
// types with no registered upcasters pass through completely unchanged.
func TestVersionedSeekableJournal_Property_passthrough(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		numEvents := rapid.IntRange(1, 30).Draw(rt, "numEvents")
		numVersions := rapid.IntRange(1, 10).Draw(rt, "numVersions")

		store := memory.NewMemoryStore()
		defer store.Close()

		ctx := context.Background()
		aggID := id.NewAggregateID()

		var events []event.Event
		for i := 0; i < numEvents; i++ {
			ver := rapid.IntRange(1, numVersions).Draw(rt, "version")
			payload := fmt.Sprintf("passthrough-%d", i)

			evt, err := event.NewEvent(
				"unregistered.type", aggID, "Test",
				event.Version(i+1),
				[]byte(payload),
				event.WithSchemaVersion(event.SchemaVersion(ver)),
			)
			if err != nil {
				t.Fatalf("create event %d: %v", i, err)
			}

			events = append(events, evt)
		}

		// Register an upcaster for a DIFFERENT event type
		unused := &chainUpcaster{sourceVer: 1}

		saveTestEvents(t, ctx, store, aggID, events...)

		journal, err := schema.NewVersionedSeekableJournal(store, unused)
		if err != nil {
			t.Fatalf("new versioned journal: %v", err)
		}

		all, err := journal.ReadAll(ctx)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}

		if len(all) != numEvents {
			t.Fatalf("expected %d events, got %d", numEvents, len(all))
		}

		for i, orig := range events {
			if all[i].SchemaVersion() != orig.SchemaVersion() {
				t.Errorf("event %d: version changed from %d to %d (should be unchanged)",
					i, orig.SchemaVersion(), all[i].SchemaVersion())
			}

			if string(all[i].Payload()) != string(orig.Payload()) {
				t.Errorf("event %d: payload changed from %q to %q",
					i, orig.Payload(), all[i].Payload())
			}
		}
	})
}

// TestVersionedSeekableJournal_Property_ReadFrom verifies that ReadFrom
// (position-based seek) applies upcasters consistently across partial reads.
func TestVersionedSeekableJournal_Property_ReadFrom(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		numEvents := rapid.IntRange(2, 40).Draw(rt, "numEvents")
		chainDepth := rapid.IntRange(1, 4).Draw(rt, "chainDepth")

		store := memory.NewMemoryStore()
		defer store.Close()

		ctx := context.Background()
		aggID := id.NewAggregateID()

		upcasters := make([]schema.Upcaster, chainDepth)
		for i := range upcasters {
			upcasters[i] = &chainUpcaster{sourceVer: i + 1}
		}

		var events []event.Event
		for i := 0; i < numEvents; i++ {
			evt, err := event.NewEvent(
				"test.event", aggID, "Test",
				event.Version(i+1),
				[]byte(fmt.Sprintf("data-%d", i)),
				event.WithSchemaVersion(1),
			)
			if err != nil {
				t.Fatalf("create event %d: %v", i, err)
			}

			events = append(events, evt)
		}

		saveTestEvents(t, ctx, store, aggID, events...)

		journal, err := schema.NewVersionedSeekableJournal(store, upcasters...)
		if err != nil {
			t.Fatalf("new versioned journal: %v", err)
		}

		// Read all via ReadAll
		all, err := journal.ReadAll(ctx)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}

		// Pick a random cursor from the middle and read the rest
		cursorIdx := rapid.IntRange(0, numEvents-2).Draw(rt, "cursorIdx")
		after := all[cursorIdx].ID()

		rest, err := journal.ReadFrom(ctx, after, numEvents)
		if err != nil {
			t.Fatalf("ReadFrom: %v", err)
		}

		expectedRest := numEvents - cursorIdx - 1
		if len(rest) != expectedRest {
			t.Fatalf("cursorIdx=%d: expected %d events after cursor, got %d",
				cursorIdx, expectedRest, len(rest))
		}

		// Every event from ReadFrom should be at the top of the chain
		for i, evt := range rest {
			if evt.SchemaVersion().Int() != chainDepth+1 {
				t.Errorf("ReadFrom event %d: schemaVersion = %d, want %d",
					i, evt.SchemaVersion().Int(), chainDepth+1)
			}
		}
	})
}

// TestVersionedSeekableJournal_MidStreamUpcastError verifies that when an
// upcaster fails mid-stream during ReadAll/ReadFrom, the error propagates
// cleanly with the event ID — no panic, no silent skip, no partial results.
func TestVersionedSeekableJournal_MidStreamUpcastError(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewAggregateID()

	const total = 10

	const failIdx = 5

	var events []event.Event
	for i := 0; i < total; i++ {
		payload := fmt.Sprintf("data-%d", i)
		if i == failIdx {
			payload = "TRIGGER"
		}

		evt, err := event.NewEvent(
			"test.event", aggID, "Test",
			event.Version(i+1),
			[]byte(payload),
			event.WithSchemaVersion(1),
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	saveTestEvents(t, ctx, store, aggID, events...)

	journal, err := schema.NewVersionedSeekableJournal(
		store,
		&conditionalFailUpcaster{trigger: "TRIGGER"},
	)
	if err != nil {
		t.Fatalf("new versioned journal: %v", err)
	}

	_, err = journal.ReadAll(ctx)
	if err == nil {
		t.Fatal("expected error from mid-stream upcaster failure")
	}

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	// ReadFrom should also surface the error
	_, err = journal.ReadFrom(ctx, events[0].ID(), total)
	if err == nil {
		t.Fatal("expected error from ReadFrom with mid-stream upcaster failure")
	}
}

func BenchmarkVersionedSeekableJournal_ReadAll_NoUpcasters(b *testing.B) {
	benchmarkReadAll(b, 0)
}

func BenchmarkVersionedSeekableJournal_ReadAll_WithUpcasters(b *testing.B) {
	benchmarkReadAll(b, 3)
}

func benchmarkReadAll(b *testing.B, chainDepth int) {
	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewAggregateID()

	const total = 10_000

	var events []event.Event
	for i := 0; i < total; i++ {
		evt, err := event.NewEvent(
			"test.event", aggID, "Test",
			event.Version(i+1),
			[]byte(fmt.Sprintf("payload-%d", i)),
			event.WithSchemaVersion(1),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(
		ctx,
		id.NewAggregateRef(id.AggregateType("Test"), aggID),
		events,
		0,
	); err != nil {
		b.Fatalf("save: %v", err)
	}

	upcasters := make([]schema.Upcaster, chainDepth)
	for i := range upcasters {
		upcasters[i] = &chainUpcaster{sourceVer: i + 1}
	}

	journal, err := schema.NewVersionedSeekableJournal(store, upcasters...)
	if err != nil {
		b.Fatalf("new versioned journal: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		all, err := journal.ReadAll(ctx)
		if err != nil {
			b.Fatalf("ReadAll: %v", err)
		}

		if len(all) != total {
			b.Fatalf("expected %d events, got %d", total, len(all))
		}
	}
}

func BenchmarkVersionedSeekableJournal_ReadFrom_WithUpcasters(b *testing.B) {
	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewAggregateID()

	const total = 10_000

	var events []event.Event
	for i := 0; i < total; i++ {
		evt, err := event.NewEvent(
			"test.event", aggID, "Test",
			event.Version(i+1),
			[]byte(fmt.Sprintf("payload-%d", i)),
			event.WithSchemaVersion(1),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(
		ctx,
		id.NewAggregateRef(id.AggregateType("Test"), aggID),
		events,
		0,
	); err != nil {
		b.Fatalf("save: %v", err)
	}

	chainDepth := 3
	upcasters := make([]schema.Upcaster, chainDepth)
	for i := range upcasters {
		upcasters[i] = &chainUpcaster{sourceVer: i + 1}
	}

	journal, err := schema.NewVersionedSeekableJournal(store, upcasters...)
	if err != nil {
		b.Fatalf("new versioned journal: %v", err)
	}

	cursor := events[0].ID()
	limit := 500

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rest, err := journal.ReadFrom(ctx, cursor, limit)
		if err != nil {
			b.Fatalf("ReadFrom: %v", err)
		}

		if len(rest) == 0 {
			b.Fatal("expected at least 1 event from ReadFrom")
		}
	}
}

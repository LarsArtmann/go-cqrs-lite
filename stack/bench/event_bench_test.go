package bench

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

// BenchmarkBundle_EventSave saves events through Bundle.EventSink.
// This should have identical ns/op to BenchmarkDirect_EventSave because
// Bundle.EventSink is the same interface value as the underlying store.
func BenchmarkBundle_EventSave(b *testing.B) {
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
	)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	ctx := event.WithProcessingMode(b.Context(), event.ModeReplay)
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Bench", aggID)

	events, _ := event.NewEvents(
		aggID, "Bench", 0,
		[]event.Type{"bench.created"},
		[]any{map[string]any{"n": 1}},
	)

	b.ResetTimer()

	for i := range b.N {
		version := event.Version(i)
		events[0] = mustReversion(b, events[0], aggID, version+1)

		if err := bundle.EventSink.Save(ctx, ref, events, version); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

// BenchmarkDirect_EventSave saves events through the store directly.
func BenchmarkDirect_EventSave(b *testing.B) {
	store := memory.NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := event.WithProcessingMode(b.Context(), event.ModeReplay)
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Bench", aggID)

	events, _ := event.NewEvents(
		aggID, "Bench", 0,
		[]event.Type{"bench.created"},
		[]any{map[string]any{"n": 1}},
	)

	b.ResetTimer()

	for i := range b.N {
		version := event.Version(i)
		events[0] = mustReversion(b, events[0], aggID, version+1)

		if err := store.Save(ctx, ref, events, version); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func mustReversion(
	b *testing.B,
	evt event.Event,
	aggID id.AggregateID,
	version event.Version,
) event.Event {
	evt2, err := event.NewEvent(
		evt.Type(),
		aggID,
		evt.AggregateType(),
		version,
		event.PayloadReadOnly(evt),
	)
	if err != nil {
		b.Fatal(err)
	}

	return evt2
}

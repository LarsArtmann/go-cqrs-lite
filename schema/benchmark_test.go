package schema_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/schema/v2"
)

func benchEvent(tb testing.TB, aggID id.AggregateID, v event.Version) event.Event {
	tb.Helper()

	evt, err := event.NewEvent("UserCreated", aggID, "User", v, []byte(`{"name":"Alice"}`))
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func BenchmarkNewUpcaster(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		upcaster := schema.NewUpcaster(
			"UserCreated", 1,
			func(evt event.Event) (*event.ImmutableEvent, error) {
				return event.NewEvent(
					evt.Type(), evt.AggregateID(), evt.AggregateType(),
					evt.Version(), evt.Payload(),
					event.WithSchemaVersion(2),
				)
			},
		)
		_ = upcaster
	}
}

func BenchmarkVersionedStore_Load(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	upcaster := schema.NewUpcaster(
		"UserCreated", 1,
		func(evt event.Event) (*event.ImmutableEvent, error) {
			return event.NewEvent(
				evt.Type(), evt.AggregateID(), evt.AggregateType(),
				evt.Version(), evt.Payload(),
				event.WithSchemaVersion(2),
			)
		},
	)

	versionedStore, err := schema.NewVersionedStore(store, upcaster)
	if err != nil {
		b.Fatalf("NewVersionedStore: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()

	for i := range 100 {
		evt := benchEvent(b, aggID, event.Version(i+1))
		_ = store.AppendBatch(ctx, event.NewAggregateRef("User", aggID), []event.Event{evt})
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := versionedStore.Load(ctx, event.NewAggregateRef("User", aggID))
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

package schema_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func benchEvent(tb testing.TB, streamID id.StreamID, v event.Version) event.Event {
	tb.Helper()

	evt, err := event.NewEvent("UserCreated", streamID, "User", v, []byte(`{"name":"Alice"}`))
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func benchSchemaVersionUpgrade(evt event.Event) (event.Event, error) {
	return event.NewEvent(
		evt.Type(), evt.StreamID(), evt.StreamType(),
		evt.Version(), evt.Payload(),
		event.WithSchemaVersion(2),
	)
}

var benchUpcaster = schema.NewUpcaster("UserCreated", 1, benchSchemaVersionUpgrade)

func BenchmarkNewUpcaster(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		_ = schema.NewUpcaster("UserCreated", 1, benchSchemaVersionUpgrade)
	}
}

func BenchmarkVersionedStore_Load(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	versionedStore, err := schema.NewVersionedStore(store, benchUpcaster)
	if err != nil {
		b.Fatalf("NewVersionedStore: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()

	for i := range 100 {
		evt := benchEvent(b, streamID, event.Version(i+1))
		_ = store.AppendBatch(ctx, id.NewStreamRef("User", streamID), []event.Event{evt})
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := versionedStore.Load(ctx, id.NewStreamRef("User", streamID))
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

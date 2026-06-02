package snapshot_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func BenchmarkEveryNEvents(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		_, err := snapshot.EveryNEvents(100)
		if err != nil {
			b.Fatalf("EveryNEvents: %v", err)
		}
	}
}

func BenchmarkEveryNEvents_ShouldSnapshot(b *testing.B) {
	b.ReportAllocs()

	strategy, err := snapshot.EveryNEvents(100)
	if err != nil {
		b.Fatalf("EveryNEvents: %v", err)
	}

	sink := memory.NewMemorySnapshotStore()
	b.Cleanup(func() { _ = sink.Close() })

	b.ResetTimer()

	for b.Loop() {
		_ = snapshot.ShouldSnapshot(
			strategy, sink, codec.JSONCodec{},
			event.AggregateType("User"), event.Version(100),
		)
	}
}

func BenchmarkSaveSnapshot(b *testing.B) {
	b.ReportAllocs()

	sink := memory.NewMemorySnapshotStore()
	b.Cleanup(func() { _ = sink.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()
	state := []byte(`{"value":42}`)

	b.ResetTimer()

	for b.Loop() {
		err := snapshot.SaveSnapshot(
			ctx, sink,
			event.AggregateType("User"), aggID,
			event.Version(100), state,
		)
		if err != nil {
			b.Fatalf("SaveSnapshot: %v", err)
		}
	}
}

func BenchmarkMemorySnapshotStore_Load(b *testing.B) {
	b.ReportAllocs()

	sink := memory.NewMemorySnapshotStore()
	b.Cleanup(func() { _ = sink.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()

	state, _ := codec.JSONCodec{}.Encode(map[string]int{"value": 42})
	_ = sink.Save(ctx, snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       100,
		State:         state,
		CreatedAt:     time.Now(),
	})

	b.ResetTimer()

	for b.Loop() {
		_, err := sink.Load(ctx, event.NewAggregateRef("User", aggID))
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

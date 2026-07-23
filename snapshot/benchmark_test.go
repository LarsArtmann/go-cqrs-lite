package snapshot_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
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

	sink := newFakeStore()
	b.Cleanup(func() { _ = sink.Close() })

	b.ResetTimer()

	for b.Loop() {
		_ = snapshot.ShouldSnapshot(
			strategy, sink, codec.JSONCodec{},
			id.StreamType("User"), event.Version(100),
		)
	}
}

func BenchmarkReadPressure_ShouldSnapshotFor(b *testing.B) {
	b.ReportAllocs()

	rp, err := snapshot.NewReadPressure(5)
	if err != nil {
		b.Fatalf("NewReadPressure: %v", err)
	}

	ref := id.NewAggregateRef("User", id.NewAggregateID())
	for range 5 {
		rp.RecordRead(ref, event.Version(1))
	}

	b.ResetTimer()

	for b.Loop() {
		_ = rp.ShouldSnapshotFor(ref, event.Version(1))
	}
}

func BenchmarkReadPressure_RecordRead(b *testing.B) {
	b.ReportAllocs()

	rp, err := snapshot.NewReadPressure(1000)
	if err != nil {
		b.Fatalf("NewReadPressure: %v", err)
	}

	ref := id.NewAggregateRef("User", id.NewAggregateID())

	b.ResetTimer()

	for b.Loop() {
		rp.RecordRead(ref, event.Version(1))
	}
}

func BenchmarkSaveSnapshot(b *testing.B) {
	b.ReportAllocs()

	sink := newFakeStore()
	b.Cleanup(func() { _ = sink.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()
	state := []byte(`{"value":42}`)

	b.ResetTimer()

	for b.Loop() {
		err := snapshot.SaveSnapshot(
			ctx, sink,
			id.StreamType("User"), aggID,
			event.Version(100), state,
		)
		if err != nil {
			b.Fatalf("SaveSnapshot: %v", err)
		}
	}
}

func BenchmarkMemorySnapshotStore_Load(b *testing.B) {
	b.ReportAllocs()

	sink := newFakeStore()
	b.Cleanup(func() { _ = sink.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()

	state, _ := codec.JSONCodec{}.Encode(map[string]int{"value": 42})
	_ = sink.Save(ctx, snapshot.Snapshot{
		StreamID:   aggID,
		StreamType: "User",
		Version:       100,
		State:         state,
		CreatedAt:     time.Now(),
	})

	b.ResetTimer()

	for b.Loop() {
		_, err := sink.Load(ctx, id.NewAggregateRef("User", aggID))
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

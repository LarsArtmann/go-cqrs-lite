package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func BenchmarkNewEvent(b *testing.B) {
	b.ReportAllocs()
	streamID := id.NewStreamID()

	for b.Loop() {
		evt, err := event.NewEvent("BenchEvent", streamID, "Bench", 1, nil)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
		if evt == nil {
			b.Fatal("NewEvent returned nil")
		}
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	b.ReportAllocs()
	bus := eventtest.NewFakeBus()

	err := bus.Subscribe("BenchEvent", eventtest.NoopEventHandler())
	if err != nil {
		b.Fatalf("subscribe: %v", err)
	}

	streamID := id.NewStreamID()

	evt, err := event.NewEvent("BenchEvent", streamID, "Bench", 1, nil)
	if err != nil {
		b.Fatalf("new event: %v", err)
	}

	ctx := context.Background()

	for b.Loop() {
		err := bus.Publish(ctx, evt)
		if err != nil {
			b.Fatalf("publish: %v", err)
		}
	}
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	b.ReportAllocs()
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for b.Loop() {
		streamID := id.NewStreamID()
		ref := id.NewStreamRef(id.StreamType("Bench"), streamID)
		evt, err := event.NewEvent("BenchEvent", streamID, "Bench", 1, nil)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	b.ReportAllocs()
	store := memory.NewMemoryStore()
	streamID := id.NewStreamID()
	ctx := context.Background()

	for i := range 10 {
		evt, _ := event.NewEvent("BenchEvent", streamID, "Bench", 1, nil)
		if err := store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Bench"), streamID),
			[]event.Event{evt},
		); err != nil {
			b.Fatalf("seed AppendBatch %d: %v", i, err)
		}
	}

	ref := id.NewStreamRef(id.StreamType("Bench"), streamID)

	for b.Loop() {
		events, err := store.Load(ctx, ref)
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
		if len(events) != 10 {
			b.Fatalf("Load: got %d events, want 10", len(events))
		}
	}
}

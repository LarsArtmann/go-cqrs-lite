package pebble

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func makeBenchEvents(b *testing.B, n int, ref id.StreamRef) []event.Event {
	events := make([]event.Event, n)

	for i := range n {
		evt, err := event.NewEvent(
			event.Type(fmt.Sprintf("test.event.%d", i)),
			ref.ID,
			ref.Type,
			event.Version(i+1),
			nil,
		)
		if err != nil {
			b.Fatalf("makeBenchEvents: %v", err)
		}

		events[i] = evt
	}

	return events
}

func openBenchStore(b *testing.B) (*EventStore, func()) {
	b.Helper()

	dir := b.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("pebble.Open: %v", err)
	}

	store, err := NewStore(
		database,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	)
	if err != nil {
		b.Fatalf("NewStore: %v", err)
	}

	return store, func() { _ = database.Close() }
}

func BenchmarkPebbleStore_Save100(b *testing.B) {
	store, cleanup := openBenchStore(b)
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		ref := id.NewStreamRef("Bench", id.NewStreamID())
		events := makeBenchEvents(b, 100, ref)

		err := store.Save(ctx, ref, events, event.Version(i*100))
		if err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkPebbleStore_SaveLoad100(b *testing.B) {
	store, cleanup := openBenchStore(b)
	defer cleanup()

	ctx := context.Background()
	ref := id.NewStreamRef("BenchLoad", id.NewStreamID())

	events := makeBenchEvents(b, 100, ref)

	err := store.Save(ctx, ref, events, event.Version(0))
	if err != nil {
		b.Fatalf("pre-save: %v", err)
	}

	b.ResetTimer()

	for range b.N {
		_, err := store.Load(ctx, ref)
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkPebbleStore_Save1(b *testing.B) {
	store, cleanup := openBenchStore(b)
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		ref := id.NewStreamRef("Bench1", id.NewStreamID())

		evt, err := event.NewEvent("test.event", ref.ID, ref.Type, event.Version(1), nil)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}

		err = store.Save(ctx, ref, []event.Event{evt}, event.Version(0))
		if err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkPebbleStore_LoadEmpty(b *testing.B) {
	store, cleanup := openBenchStore(b)
	defer cleanup()

	ctx := context.Background()
	ref := id.NewStreamRef("BenchEmpty", id.NewStreamID())

	b.ResetTimer()

	for range b.N {
		_, _ = store.Load(ctx, ref)
	}
}

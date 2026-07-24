package turso_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func benchTursoEventStore(b *testing.B) (*storage.SQLEventStore, func()) {
	b.Helper()

	conn, err := turso.OpenTemp(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}

	if err := turso.InitSchema(context.Background(), conn); err != nil {
		b.Fatal(err)
	}

	store, err := turso.NewEventStore(conn)
	if err != nil {
		b.Fatal(err)
	}

	return store, func() { _ = store.Close(); _ = conn.Close() }
}

func BenchmarkTursoEventStore_Save(b *testing.B) {
	b.ReportAllocs()

	store, cleanup := benchTursoEventStore(b)
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("Bench", streamID)
		evt, err := event.NewEvent("BenchCreated", streamID, "Bench", 1, []byte(`{"key":"value"}`))
		if err != nil {
			b.Fatal(err)
		}

		err = store.Save(ctx, ref, []event.Event{evt}, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTursoEventStore_Load(b *testing.B) {
	b.ReportAllocs()

	store, cleanup := benchTursoEventStore(b)
	defer cleanup()

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Bench", streamID)

	for i := range 100 {
		evt, _ := event.NewEvent(
			"BenchEvent",
			streamID,
			"Bench",
			event.Version(i+1),
			[]byte(`{"i":0}`),
		)
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := store.Load(ctx, ref)
		if err != nil {
			b.Fatal(err)
		}
	}
}

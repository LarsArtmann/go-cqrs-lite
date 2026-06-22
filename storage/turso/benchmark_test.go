package turso_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
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
		aggID := id.NewAggregateID()
		ref := event.NewAggregateRef("Bench", aggID)
		evt, err := event.NewEvent("BenchCreated", aggID, "Bench", 1, []byte(`{"key":"value"}`))
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
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Bench", aggID)

	for i := range 100 {
		evt, _ := event.NewEvent(
			"BenchEvent",
			aggID,
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

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func BenchmarkSQLiteEventStore_Save(b *testing.B) {
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	payload := []byte(`{"name":"bench-user"}`)

	benchSaveNewAggregate(b, store, "User", "user.created", payload)
}

func BenchmarkSQLiteEventStore_Load(b *testing.B) {
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	payload := []byte(`{"name":"bench-user"}`)

	seedSQLiteEvents(b, store, "User", aggID, "user.updated", payload, 10)

	b.ResetTimer()

	for b.Loop() {
		_, err := store.Load(ctx, "User", aggID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLiteEventStore_LoadAll(b *testing.B) {
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	payload := []byte(`{}`)

	for i := range 100 {
		aggID := id.NewAggregateID()

		evt, _ := event.NewEvent(
			event.Type(fmt.Sprintf("event.%d", i)), aggID, "Bench",
			event.Version(1), payload,
		)

		err := store.AppendBatch(ctx, "Bench", aggID, []event.Event{evt})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := store.LoadAll(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLiteEventStore_LoadToVersion(b *testing.B) {
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	payload := []byte(`{}`)

	seedSQLiteEvents(b, store, "User", aggID, "user.updated", payload, 50)

	b.ResetTimer()

	for b.Loop() {
		_, err := store.LoadToVersion(ctx, "User", aggID, event.Version(25))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTursoEventStore_Save(b *testing.B) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	if err := TursoInitSchema(context.Background(), db); err != nil {
		b.Fatal(err)
	}

	store, err := NewTursoEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	payload := []byte(`{"name":"bench-turso"}`)

	benchSaveNewAggregate(b, store, "Order", "order.placed", payload)
}

func BenchmarkTursoEventStore_Load(b *testing.B) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	if err := TursoInitSchema(context.Background(), db); err != nil {
		b.Fatal(err)
	}

	store, err := NewTursoEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	payload := []byte(`{"name":"bench-turso"}`)

	seedSQLiteEvents(b, store, "Order", aggID, "order.updated", payload, 10)

	b.ResetTimer()

	for b.Loop() {
		_, err := store.Load(ctx, "Order", aggID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchSaveNewAggregate(
	b *testing.B,
	store *SQLEventStore,
	aggType event.AggregateType,
	eventType event.Type,
	payload []byte,
) {
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		aggID := id.NewAggregateID()

		evt, _ := event.NewEvent(
			eventType, aggID, aggType,
			event.Version(1), payload,
		)

		err := store.Save(ctx, aggType, aggID, []event.Event{evt}, event.Version(0))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func seedSQLiteEvents(
	b *testing.B,
	store *SQLEventStore,
	aggType event.AggregateType,
	aggID id.AggregateID,
	eventType event.Type,
	payload []byte,
	n int,
) {
	b.Helper()

	ctx := context.Background()

	for i := range n {
		evt, _ := event.NewEvent(
			eventType, aggID, aggType,
			event.Version(i+1), payload,
		)

		err := store.Save(ctx, aggType, aggID, []event.Event{evt}, event.Version(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func openSQLiteBenchDB(b *testing.B) (*sql.DB, error) {
	b.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteOutboxSchema()} {
		_, err := db.ExecContext(context.Background(), ddl)
		if err != nil {
			return nil, fmt.Errorf("exec DDL: %w", err)
		}
	}

	return db, nil
}

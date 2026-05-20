package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	_ "modernc.org/sqlite"
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

	ctx := context.Background()
	payload := []byte(`{"name":"bench-user"}`)

	b.ResetTimer()

	for b.Loop() {
		aggID := id.NewAggregateID()

		evt, _ := event.NewEvent(
			"user.created", aggID, "User",
			event.Version(1), payload,
		)

		err := store.Save(ctx, "User", aggID, []event.Event{evt}, event.Version(0))
		if err != nil {
			b.Fatal(err)
		}
	}
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

	for i := range 10 {
		evt, _ := event.NewEvent(
			"user.updated", aggID, "User",
			event.Version(i+1), payload,
		)

		err := store.Save(ctx, "User", aggID, []event.Event{evt}, event.Version(i))
		if err != nil {
			b.Fatal(err)
		}
	}

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

	for i := range 50 {
		evt, _ := event.NewEvent(
			"user.updated", aggID, "User",
			event.Version(i+1), payload,
		)

		err := store.Save(ctx, "User", aggID, []event.Event{evt}, event.Version(i))
		if err != nil {
			b.Fatal(err)
		}
	}

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

	ctx := context.Background()
	payload := []byte(`{"name":"bench-turso"}`)

	b.ResetTimer()

	for b.Loop() {
		aggID := id.NewAggregateID()

		evt, _ := event.NewEvent(
			"order.placed", aggID, "Order",
			event.Version(1), payload,
		)

		err := store.Save(ctx, "Order", aggID, []event.Event{evt}, event.Version(0))
		if err != nil {
			b.Fatal(err)
		}
	}
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

	for i := range 10 {
		evt, _ := event.NewEvent(
			"order.updated", aggID, "Order",
			event.Version(i+1), payload,
		)

		err := store.Save(ctx, "Order", aggID, []event.Event{evt}, event.Version(i))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := store.Load(ctx, "Order", aggID)
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

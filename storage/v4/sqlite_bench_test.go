package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func BenchmarkSQLiteEventStore_Save(b *testing.B) {
	b.ReportAllocs()
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

	benchSaveNewStream(b, store, "User", "user.created", payload)
}

func BenchmarkSQLiteEventStore_Load(b *testing.B) {
	b.ReportAllocs()
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	streamID := id.NewStreamID()
	payload := []byte(`{"name":"bench-user"}`)

	seedSQLiteEvents(b, store, "User", streamID, "user.updated", payload, 10)

	benchLoadStream(b, store, "User", streamID)
}

func benchLoadStream(
	b *testing.B,
	store *SQLEventStore,
	streamType id.StreamType,
	streamID id.StreamID,
) {
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := store.Load(ctx, id.NewStreamRef(streamType, streamID))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func seedSQLiteBenchEvents(b *testing.B, store *SQLEventStore, n int) {
	b.Helper()

	ctx := context.Background()
	payload := []byte(`{}`)

	for i := range n {
		streamID := id.NewStreamID()

		evt, _ := event.NewEvent(
			event.Type(fmt.Sprintf("event.%d", i)), streamID, "Bench",
			event.Version(1), payload,
		)

		err := store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Bench"), streamID),
			[]event.Event{evt},
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLiteEventStore_ReadAll(b *testing.B) {
	b.ReportAllocs()
	db, err := openSQLiteBenchDB(b)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatal(err)
	}

	seedSQLiteBenchEvents(b, store, 100)

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := store.ReadAll(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLiteEventStore_LoadToVersion(b *testing.B) {
	b.ReportAllocs()
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
	streamID := id.NewStreamID()
	payload := []byte(`{}`)

	seedSQLiteEvents(b, store, "User", streamID, "user.updated", payload, 50)

	b.ResetTimer()

	for b.Loop() {
		_, err := store.LoadToVersion(
			ctx,
			id.NewStreamRef(id.StreamType("User"), streamID),
			event.Version(25),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchSaveNewStream(
	b *testing.B,
	store *SQLEventStore,
	streamType id.StreamType,
	eventType event.Type,
	payload []byte,
) {
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()

		evt, _ := event.NewEvent(
			eventType, streamID, streamType,
			event.Version(1), payload,
		)

		err := store.Save(
			ctx,
			id.NewStreamRef(streamType, streamID),
			[]event.Event{evt},
			event.Version(0),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func seedSQLiteEvents(
	b *testing.B,
	store *SQLEventStore,
	streamType id.StreamType,
	streamID id.StreamID,
	eventType event.Type,
	payload []byte,
	n int,
) {
	b.Helper()

	ctx := context.Background()

	for i := range n {
		evt, _ := event.NewEvent(
			eventType, streamID, streamType,
			event.Version(i+1), payload,
		)

		err := store.Save(
			ctx,
			id.NewStreamRef(streamType, streamID),
			[]event.Event{evt},
			event.Version(i),
		)
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

	for _, ddl := range []string{sqlpkg.SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema()} {
		_, err := db.ExecContext(context.Background(), ddl)
		if err != nil {
			return nil, fmt.Errorf("exec DDL: %w", err)
		}
	}

	return db, nil
}

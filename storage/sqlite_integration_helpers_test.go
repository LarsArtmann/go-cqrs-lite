package storage

import (
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	db.SetMaxOpenConns(1)

	return db
}

func initSQLiteSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, ddl := range []string{sqlpkg.SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema()} {
		_, err := db.ExecContext(context.Background(), ddl)
		if err != nil {
			t.Fatalf("exec DDL: %v\nDDL: %s", err, ddl)
		}
	}
}

func newSQLiteTestStore(t *testing.T) *SQLEventStore {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	return store
}

func newTestSnapshot(
	aggID id.AggregateID,
	aggregateType event.AggregateType,
	version event.Version,
	state []byte,
) event.Snapshot {
	return event.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggregateType,
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}
}

func assertSnapshotVersion(t *testing.T, loaded *event.Snapshot, want event.Version) {
	t.Helper()

	if loaded.Version.Int() != int(want) {
		t.Errorf("Version = %d, want %d", loaded.Version.Int(), int(want))
	}
}

func saveAndLoadSnapshot(
	t *testing.T,
	store event.SnapshotStore,
	ctx context.Context,
	snap event.Snapshot,
	want event.Version,
) *event.Snapshot {
	t.Helper()

	err := store.Save(ctx, snap)
	if err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(snap.AggregateType, snap.AggregateID))
	if err != nil {
		t.Fatalf("Load snapshot: %v", err)
	}

	assertSnapshotVersion(t, loaded, want)

	return loaded
}

func TestNewSQLiteEventStore_NilDB(t *testing.T) {
	_, err := NewSQLiteEventStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteSnapshotStore_NilDB(t *testing.T) {
	_, err := NewSQLiteSnapshotStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteCheckpointStore_NilDB(t *testing.T) {
	_, err := NewSQLiteCheckpointStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteOutbox_NilDB(t *testing.T) {
	_, err := NewSQLiteOutbox(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLTransactionalStore_NilStore_SQLite(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	outbox, err := NewSQLiteOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLiteOutbox: %v", err)
	}

	_, err = NewSQLTransactionalStore(nil, outbox)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestNewSQLTransactionalStore_NilOutbox_SQLite(t *testing.T) {
	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	_, err = NewSQLTransactionalStore(store, nil)
	if err == nil {
		t.Fatal("expected error for nil outbox")
	}
}

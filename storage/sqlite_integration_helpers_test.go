package storage

import (
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

	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteSagaSchema()} {
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

func newTestSnapshot(aggID id.AggregateID, version event.Version, state []byte) event.Snapshot {
	return event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}
}

func newSQLiteTestSagaStore(t *testing.T) *SQLSagaStore {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSagaStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSagaStore: %v", err)
	}

	return store
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

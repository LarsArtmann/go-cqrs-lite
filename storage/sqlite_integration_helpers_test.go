package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
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
	aggID id.StreamID,
	streamType id.StreamType,
	version event.Version,
	state []byte,
) snapshot.Snapshot {
	return snapshot.Snapshot{
		StreamID:   aggID,
		StreamType: streamType,
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}
}

func assertSnapshotVersion(t *testing.T, loaded *snapshot.Snapshot, want event.Version) {
	t.Helper()

	if loaded.Version.Int() != int(want) {
		t.Errorf("Version = %d, want %d", loaded.Version.Int(), int(want))
	}
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

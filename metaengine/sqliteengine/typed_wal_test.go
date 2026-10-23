package sqliteengine_test

// Typed-WAL pins (disk-truth plan A11/A14): the journal of record carries
// database-level identity constraints — a duplicate event append CONFLICTS
// instead of silently adding a second row — and a legacy (pre-typed) database
// migrates in place: columns added, identity backfilled, reads unchanged.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

func openTypedWALEngine(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "typed.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng, db
}

const typedEventEnvelope = `{"id":"01TYPED0000000000000000001","type":"t.evt",` +
	`"stream_id":"01TYPEDSTREAM000000000001","stream_type":"Task","version":1,` +
	`"schema_version":0,"payload":"gx9mA2p9bnVt","encoding":"cbor",` +
	`"metadata":"e30=","occurred_at":"2026-10-23T15:00:00Z"}`

func TestTypedWAL_DuplicateEventAppendConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALEngine(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	if err := backend.StreamAppend(ctx, "events", "Task:01TYPEDSTREAM000000000001",
		[]any{typedEventEnvelope}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	err := backend.StreamAppend(ctx, "events", "Task:01TYPEDSTREAM000000000001",
		[]any{typedEventEnvelope})
	if err == nil {
		t.Fatal("duplicate event append must conflict (UNIQUE event_id)")
	}

	if err != metaengine.ErrVersionConflict {
		t.Fatalf("duplicate append error = %v, want metaengine.ErrVersionConflict", err)
	}
}

func TestTypedWAL_StreamVersionDuplicateConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALEngine(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	if err := backend.StreamAppend(ctx, "events", "Task:01TYPEDSTREAM000000000002",
		[]any{typedEventEnvelope}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	// Same (collection, stream, version), DIFFERENT event id: the second
	// identity index must reject the slot collision.
	otherID := `{"id":"01TYPED0000000000000000002","type":"t.evt",` +
		`"stream_id":"01TYPEDSTREAM000000000002","stream_type":"Task","version":1,` +
		`"payload":"gA","encoding":"cbor","metadata":"e30=",` +
		`"occurred_at":"2026-10-23T15:00:01Z"}`

	err := backend.StreamAppend(ctx, "events", "Task:01TYPEDSTREAM000000000002", []any{otherID})
	if err != metaengine.ErrVersionConflict {
		t.Fatalf("(stream,version) duplicate error = %v, want ErrVersionConflict", err)
	}
}

func TestTypedWAL_TypedColumnsFilled(t *testing.T) {
	t.Parallel()

	eng, db := openTypedWALEngine(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	if err := backend.StreamAppend(ctx, "events", "Task:01TYPEDSTREAM000000000001",
		[]any{typedEventEnvelope}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var eventID, streamType, encoding string

	var version int

	var payload []byte

	var occurredAt sql.NullInt64

	err := db.QueryRow(`SELECT event_id, stream_type, version, encoding, payload, occurred_at
		FROM meta_stream_log WHERE collection = 'events'`).
		Scan(&eventID, &streamType, &version, &encoding, &payload, &occurredAt)
	if err != nil {
		t.Fatalf("read typed columns: %v", err)
	}

	if eventID != "01TYPED0000000000000000001" || streamType != "Task" ||
		version != 1 || encoding != "cbor" {
		t.Fatalf("typed identity columns wrong: %q %q %d %q", eventID, streamType, version, encoding)
	}

	if len(payload) == 0 {
		t.Fatal("payload BLOB column not filled")
	}

	if !occurredAt.Valid || occurredAt.Int64 <= 0 {
		t.Fatalf("occurred_at nanos not filled: %+v", occurredAt)
	}
}

func TestTypedWAL_LegacyDatabaseMigratesInPlace(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// A pre-typed WAL: the old three-column shape with envelope rows.
	if _, err := db.Exec(`CREATE TABLE meta_stream_log (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		collection TEXT NOT NULL,
		stream_id TEXT NOT NULL,
		value TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO meta_stream_log (collection, stream_id, value)
		VALUES ('events', 'Task:01TYPEDSTREAM000000000001', ?)`, typedEventEnvelope); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("open engine over legacy DB: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	// Backfill filled the identity columns from the envelope.
	var eventID string

	var version int

	if err := db.QueryRow(`SELECT event_id, version FROM meta_stream_log
		WHERE collection = 'events'`).Scan(&eventID, &version); err != nil {
		t.Fatalf("read backfilled columns: %v", err)
	}

	if eventID != "01TYPED0000000000000000001" || version != 1 {
		t.Fatalf("backfill wrong: event_id=%q version=%d", eventID, version)
	}

	// The duplicate that legacy shape silently accepted now conflicts.
	backend := eng.(metaengine.StreamLogBackend)

	err = backend.StreamAppend(context.Background(), "events",
		"Task:01TYPEDSTREAM000000000001", []any{typedEventEnvelope})
	if err != metaengine.ErrVersionConflict {
		t.Fatalf("post-migration duplicate = %v, want ErrVersionConflict", err)
	}

	// Reads through the engine still work (value envelope untouched).
	vals, err := backend.StreamRead(context.Background(), "events",
		"Task:01TYPEDSTREAM000000000001")
	if err != nil || len(vals) != 1 {
		t.Fatalf("StreamRead after migration: %v, %d values", err, len(vals))
	}

	// By-ID lookup prefers the typed column over the legacy row.
	byID, ok := eng.(metaengine.EventByIDBackend)
	if !ok {
		t.Fatal("engine must implement EventByIDBackend")
	}

	_, err = byID.StreamLoadByEventID(context.Background(), "events",
		"01TYPED0000000000000000001")
	if err != nil {
		t.Fatalf("StreamLoadByEventID after migration: %v", err)
	}
}

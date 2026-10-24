package duckdbengine_test

// Typed-WAL pins (disk-truth plan A13, mirroring the sqliteengine A11
// reference): the journal of record carries database-level identity
// constraints — a duplicate event append CONFLICTS instead of silently adding
// a second row — and a legacy (pre-typed) database migrates in place:
// columns added, identity backfilled, reads unchanged.

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func openTypedWALDuck(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "typed.db")

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// One shared connection: DuckDB locks the database file per instance, so
	// the engine and the raw-row assertions must ride the SAME handle.
	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Fatalf("duckdb engine construction failed: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng, db
}

// typedEnvelope builds a wire envelope the way the system adapter does —
// json.Marshal of the real shape — so the []byte fields carry guaranteed-valid
// base64 (hand-written fixture strings once shipped INVALID base64 and the
// json decode of the payload field failed, silently disabling typed
// extraction for the whole row: the A11 red-pin lesson).
func typedEnvelope(t *testing.T, eventID, streamID string, version int) string {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"id":             eventID,
		"type":           "t.evt",
		"stream_id":      streamID,
		"stream_type":    "Task",
		"version":        version,
		"schema_version": 0,
		"payload":        []byte{0xa1, 0x02, 0x69, 0x64},
		"encoding":       "cbor",
		"metadata":       []byte(`{}`),
		"occurred_at":    "2026-10-23T15:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return string(data)
}

const (
	duckTypedStreamID = "01TYPEDSTREAM000000000001"
	duckTypedEventID1 = "01TYPED0000000000000000001"
	duckTypedEventID2 = "01TYPED0000000000000000002"
)

func TestTypedWAL_DuplicateEventAppendConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALDuck(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	env := typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)

	if err := backend.StreamAppend(ctx, "events", "Task:"+duckTypedStreamID,
		[]any{env}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	err := backend.StreamAppend(ctx, "events", "Task:"+duckTypedStreamID, []any{env})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("duplicate append error = %v, want metaengine.ErrVersionConflict", err)
	}
}

func TestTypedWAL_StreamVersionDuplicateConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALDuck(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	if err := backend.StreamAppend(ctx, "events", "Task:"+duckTypedStreamID,
		[]any{typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	// Same (collection, stream, version), DIFFERENT event id: the second
	// identity index must reject the slot collision.
	err := backend.StreamAppend(ctx, "events", "Task:"+duckTypedStreamID,
		[]any{typedEnvelope(t, duckTypedEventID2, duckTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("(stream,version) duplicate error = %v, want ErrVersionConflict", err)
	}
}

func TestTypedWAL_TypedColumnsFilled(t *testing.T) {
	t.Parallel()

	eng, db := openTypedWALDuck(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	if err := backend.StreamAppend(ctx, "events", "Task:"+duckTypedStreamID,
		[]any{typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var eventID, streamType, encoding string

	var version int

	var payload, metadata []byte

	var occurredAt sql.NullInt64

	err := db.QueryRow(`SELECT event_id, stream_type, version, encoding, payload, metadata, occurred_at
		FROM meta_stream_log WHERE collection = 'events'`).
		Scan(&eventID, &streamType, &version, &encoding, &payload, &metadata, &occurredAt)
	if err != nil {
		t.Fatalf("read typed columns: %v", err)
	}

	if eventID != duckTypedEventID1 || streamType != "Task" ||
		version != 1 || encoding != "cbor" {
		t.Fatalf("typed identity columns wrong: %q %q %d %q", eventID, streamType, version, encoding)
	}

	if len(payload) == 0 {
		t.Fatal("payload BLOB column not filled")
	}

	if len(metadata) == 0 {
		t.Fatal("metadata column not filled")
	}

	if !occurredAt.Valid || occurredAt.Int64 <= 0 {
		t.Fatalf("occurred_at nanos not filled: %+v", occurredAt)
	}
}

func TestTypedWAL_LegacyDatabaseMigratesInPlace(t *testing.T) {
	t.Parallel()

	dsn := filepath.Join(t.TempDir(), "legacy.db")

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// A pre-typed WAL: the old four-column shape with envelope rows (the
	// sequence predates the typed columns too).
	if _, err := db.Exec(`CREATE SEQUENCE seq_stream_log`); err != nil {
		t.Fatalf("create legacy sequence: %v", err)
	}

	if _, err := db.Exec(`CREATE TABLE meta_stream_log (
		seq BIGINT PRIMARY KEY DEFAULT nextval('seq_stream_log'),
		collection VARCHAR NOT NULL,
		stream_id VARCHAR NOT NULL,
		value VARCHAR NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO meta_stream_log (collection, stream_id, value)
		VALUES ('events', 'Task:`+duckTypedStreamID+`', ?)`,
		typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	eng, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Fatalf("open engine over legacy DB: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	// Backfill filled the identity columns from the envelope (payload/
	// metadata stay NULL on legacy rows by contract — writer-filled, readers
	// fall back to the envelope).
	var eventID, encoding string

	var version int

	if err := db.QueryRow(`SELECT event_id, version, encoding
		FROM meta_stream_log WHERE collection = 'events'`).
		Scan(&eventID, &version, &encoding); err != nil {
		t.Fatalf("read backfilled columns: %v", err)
	}

	if eventID != duckTypedEventID1 || version != 1 || encoding != "cbor" {
		t.Fatalf("backfilled identity wrong: %q %d %q", eventID, version, encoding)
	}

	// Post-migration, the unique indexes enforce identity: re-appending the
	// same envelope conflicts instead of silently doubling the journal.
	backend := eng.(metaengine.StreamLogBackend)

	err = backend.StreamAppend(context.Background(), "events", "Task:"+duckTypedStreamID,
		[]any{typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("post-migration duplicate append error = %v, want ErrVersionConflict", err)
	}

	// Reads are unchanged: the envelope stays the read path.
	values, err := backend.StreamRead(context.Background(), "events", "Task:"+duckTypedStreamID)
	if err != nil {
		t.Fatalf("post-migration read: %v", err)
	}

	if len(values) != 1 {
		t.Fatalf("post-migration stream length = %d, want 1", len(values))
	}
}

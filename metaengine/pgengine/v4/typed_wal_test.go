package pgengine_test

// Typed-WAL pins (disk-truth plan A12, mirroring the sqliteengine A11
// reference): the journal of record carries database-level identity
// constraints — a duplicate event append CONFLICTS instead of silently adding
// a second row — and a legacy (pre-typed) database migrates in place: columns
// added, identity backfilled, reads unchanged.

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func openTypedWALPg(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	dsn := pgDSN(t)
	eng, err := pgengine.New(dsn)
	if err != nil {
		if pgSkipClass(err) {
			t.Skipf("Postgres not available: %v", err)
		}

		t.Fatalf("postgres engine construction failed: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open raw pgx conn: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

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
	pgTypedStreamID = "01TYPEDSTREAM000000000001"
	pgTypedEventID1 = "01TYPED0000000000000000001"
	pgTypedEventID2 = "01TYPED0000000000000000002"
)

// runSuffix makes every derived name unique per test-binary run: under the
// shared POSTGRES_TEST_DSN (CI service container) the database PERSISTS
// across runs, so a stable name would make the second run's first append
// conflict with the first run's rows (the adttest rerun-safety lesson —
// deterministic red on run 2).
var runSuffix = fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)

// typedCollection derives a run-unique collection from the test name: with
// POSTGRES_TEST_DSN set there is NO per-test database (the testcontainer path
// isolates per test), so parallel pins must not share the 'events' collection.
func typedCollection(t *testing.T) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "/", "")
	if len(name) > 24 {
		name = name[:24]
	}

	return "events_" + name + "_" + runSuffix
}

func TestTypedWAL_DuplicateEventAppendConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALPg(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()

	env := typedEnvelope(t, pgTypedEventID1, pgTypedStreamID, 1)
	col := typedCollection(t)

	if err := backend.StreamAppend(ctx, col, "Task:"+pgTypedStreamID,
		[]any{env}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	err := backend.StreamAppend(ctx, col, "Task:"+pgTypedStreamID, []any{env})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("duplicate append error = %v, want metaengine.ErrVersionConflict", err)
	}
}

func TestTypedWAL_StreamVersionDuplicateConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALPg(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := typedCollection(t)

	if err := backend.StreamAppend(ctx, col, "Task:"+pgTypedStreamID,
		[]any{typedEnvelope(t, pgTypedEventID1, pgTypedStreamID, 1)}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	// Same (collection, stream, version), DIFFERENT event id: the second
	// identity index must reject the slot collision.
	err := backend.StreamAppend(ctx, col, "Task:"+pgTypedStreamID,
		[]any{typedEnvelope(t, pgTypedEventID2, pgTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("(stream,version) duplicate error = %v, want ErrVersionConflict", err)
	}
}

func TestTypedWAL_TypedColumnsFilled(t *testing.T) {
	t.Parallel()

	eng, db := openTypedWALPg(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := typedCollection(t)

	if err := backend.StreamAppend(ctx, col, "Task:"+pgTypedStreamID,
		[]any{typedEnvelope(t, pgTypedEventID1, pgTypedStreamID, 1)}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var eventID, streamType, encoding string

	var version int

	var payload []byte

	var metadata []byte

	var occurredAt sql.NullInt64

	err := db.QueryRow(`SELECT event_id, stream_type, version, encoding, payload, metadata, occurred_at
		FROM meta_stream_log WHERE collection = $1`, col).
		Scan(&eventID, &streamType, &version, &encoding, &payload, &metadata, &occurredAt)
	if err != nil {
		t.Fatalf("read typed columns: %v", err)
	}

	if eventID != pgTypedEventID1 || streamType != "Task" ||
		version != 1 || encoding != "cbor" {
		t.Fatalf(
			"typed identity columns wrong: %q %q %d %q",
			eventID,
			streamType,
			version,
			encoding,
		)
	}

	if len(payload) == 0 {
		t.Fatal("payload BYTEA column not filled")
	}

	if len(metadata) == 0 {
		t.Fatal("metadata JSONB column not filled")
	}

	if !occurredAt.Valid || occurredAt.Int64 <= 0 {
		t.Fatalf("occurred_at nanos not filled: %+v", occurredAt)
	}
}

// legacyScratchDSN provisions a throwaway database on the test server and
// returns its DSN. The legacy-migration pin needs a server that has NEVER
// seen the typed schema: under the shared POSTGRES_TEST_DSN (CI service
// container) meta_stream_log already exists WITH the typed columns, so the
// ALTER/backfill path would silently degrade to a no-op. A scratch database
// restores the true legacy scenario under every DSN mode; it is dropped on
// test cleanup.
func legacyScratchDSN(t *testing.T) string {
	t.Helper()

	u, err := url.Parse(pgDSN(t))
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}

	admin := *u
	scratch := *u
	scratch.Path = "/twl_" + runSuffix

	adminDB, err := sql.Open("pgx", admin.String())
	if err != nil {
		t.Fatalf("open admin conn: %v", err)
	}

	if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + scratch.Path[1:]); err != nil {
		_ = adminDB.Close()
		t.Fatalf("drop stale scratch database: %v", err)
	}

	if _, err := adminDB.Exec("CREATE DATABASE " + scratch.Path[1:]); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	t.Cleanup(func() {
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS " + scratch.Path[1:])
		_ = adminDB.Close()
	})

	return scratch.String()
}

func TestTypedWAL_LegacyDatabaseMigratesInPlace(t *testing.T) {
	t.Parallel()

	dsn := legacyScratchDSN(t)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open pgx: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// A pre-typed WAL: the old four-column shape with envelope rows.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta_stream_log (
		seq BIGSERIAL PRIMARY KEY,
		collection TEXT NOT NULL,
		stream_id TEXT NOT NULL,
		value TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	col := typedCollection(t)

	if _, err := db.Exec(`INSERT INTO meta_stream_log (collection, stream_id, value)
		VALUES ($1, 'Task:`+pgTypedStreamID+`', $2)`,
		col, typedEnvelope(t, pgTypedEventID1, pgTypedStreamID, 1)); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	eng, err := pgengine.NewFromDB(db)
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
		FROM meta_stream_log WHERE collection = $1`, col).
		Scan(&eventID, &version, &encoding); err != nil {
		t.Fatalf("read backfilled columns: %v", err)
	}

	if eventID != pgTypedEventID1 || version != 1 || encoding != "cbor" {
		t.Fatalf("backfilled identity wrong: %q %d %q", eventID, version, encoding)
	}

	// Post-migration, the unique indexes enforce identity: re-appending the
	// same envelope conflicts instead of silently doubling the journal.
	backend := eng.(metaengine.StreamLogBackend)

	err = backend.StreamAppend(context.Background(), col, "Task:"+pgTypedStreamID,
		[]any{typedEnvelope(t, pgTypedEventID1, pgTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("post-migration duplicate append error = %v, want ErrVersionConflict", err)
	}

	// Reads are unchanged: the envelope stays the read path.
	values, err := backend.StreamRead(context.Background(), col, "Task:"+pgTypedStreamID)
	if err != nil {
		t.Fatalf("post-migration read: %v", err)
	}

	if len(values) != 1 {
		t.Fatalf("post-migration stream length = %d, want 1", len(values))
	}
}

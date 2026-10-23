package mysqlengine_test

// Typed-WAL pins (disk-truth plan A12b, mirroring the sqliteengine A11
// reference): the journal of record carries database-level identity
// constraints — a duplicate event append CONFLICTS instead of silently adding
// a second row — and a legacy (pre-typed) database migrates in place with
// INSTANT ALTERs only: the table is never recreated, never locked.

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func openTypedWALMySQL(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	dsn := mysqlTestDSN()
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	eng, err := mysqlengine.New(dsn)
	if err != nil {
		if mysqlSkipClass(err) {
			t.Skipf("MySQL not available: %v", err)
		}

		t.Fatalf("mysql engine construction failed: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open raw mysql conn: %v", err)
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
	myTypedStreamID = "01TYPEDSTREAM000000000001"
	myTypedEventID1 = "01TYPED0000000000000000001"
	myTypedEventID2 = "01TYPED0000000000000000002"
)

// runSuffix makes every derived name unique per test-binary run: the shared
// MYSQL_TEST_DSN database PERSISTS across runs, so a stable name would make
// the second run's first append conflict with the first run's rows (the
// adttest rerun-safety lesson — deterministic red on run 2).
var runSuffix = fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)

// typedCollection derives a run-unique collection from the test name,
// trimmed to keep the full identifier inside MySQL's 64-char limit.
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

	eng, _ := openTypedWALMySQL(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := typedCollection(t)

	env := typedEnvelope(t, myTypedEventID1, myTypedStreamID, 1)

	if err := backend.StreamAppend(ctx, col, "Task:"+myTypedStreamID,
		[]any{env}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	err := backend.StreamAppend(ctx, col, "Task:"+myTypedStreamID, []any{env})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("duplicate append error = %v, want metaengine.ErrVersionConflict", err)
	}
}

func TestTypedWAL_StreamVersionDuplicateConflicts(t *testing.T) {
	t.Parallel()

	eng, _ := openTypedWALMySQL(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := typedCollection(t)

	if err := backend.StreamAppend(ctx, col, "Task:"+myTypedStreamID,
		[]any{typedEnvelope(t, myTypedEventID1, myTypedStreamID, 1)}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	// Same (collection, stream, version), DIFFERENT event id: the second
	// identity index must reject the slot collision.
	err := backend.StreamAppend(ctx, col, "Task:"+myTypedStreamID,
		[]any{typedEnvelope(t, myTypedEventID2, myTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("(stream,version) duplicate error = %v, want ErrVersionConflict", err)
	}
}

func TestTypedWAL_TypedColumnsFilled(t *testing.T) {
	t.Parallel()

	eng, db := openTypedWALMySQL(t)

	backend := eng.(metaengine.StreamLogBackend)
	ctx := context.Background()
	col := typedCollection(t)

	if err := backend.StreamAppend(ctx, col, "Task:"+myTypedStreamID,
		[]any{typedEnvelope(t, myTypedEventID1, myTypedStreamID, 1)}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var eventID, streamType, encoding string

	var version int

	var payload []byte

	var metadata []byte

	var occurredAt sql.NullInt64

	err := db.QueryRow(`SELECT event_id, stream_type, version, encoding, payload, metadata, occurred_at
		FROM meta_stream_log WHERE collection = ?`, col).
		Scan(&eventID, &streamType, &version, &encoding, &payload, &metadata, &occurredAt)
	if err != nil {
		t.Fatalf("read typed columns: %v", err)
	}

	if eventID != myTypedEventID1 || streamType != "Task" ||
		version != 1 || encoding != "cbor" {
		t.Fatalf("typed identity columns wrong: %q %q %d %q", eventID, streamType, version, encoding)
	}

	if len(payload) == 0 {
		t.Fatal("payload BLOB column not filled")
	}

	if len(metadata) == 0 {
		t.Fatal("metadata JSON column not filled")
	}

	if !occurredAt.Valid || occurredAt.Int64 <= 0 {
		t.Fatalf("occurred_at nanos not filled: %+v", occurredAt)
	}
}

func TestTypedWAL_LegacyDatabaseMigratesInPlace(t *testing.T) {
	t.Parallel()

	// The legacy pin needs a database that has NEVER seen the typed schema.
	// MySQL/MariaDB cannot CREATE DATABASE inside this test's connection to a
	// different schema cheaply — it can; do it on the same server.
	dsn := legacyScratchDSN(t)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// A pre-typed WAL: the old four-column shape with envelope rows.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta_stream_log (
		seq BIGINT AUTO_INCREMENT PRIMARY KEY,
		collection VARCHAR(255) NOT NULL,
		stream_id VARCHAR(255) NOT NULL,
		value TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	col := typedCollection(t)

	if _, err := db.Exec(`INSERT INTO meta_stream_log (collection, stream_id, value)
		VALUES (?, 'Task:`+myTypedStreamID+`', ?)`,
		col, typedEnvelope(t, myTypedEventID1, myTypedStreamID, 1)); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// The no-recreate baseline: the seeded row's identity before migration.
	var beforeSeq int64

	var beforeValue string

	if err := db.QueryRow(`SELECT seq, value FROM meta_stream_log WHERE collection = ?`, col).
		Scan(&beforeSeq, &beforeValue); err != nil {
		t.Fatalf("baseline read: %v", err)
	}

	eng, err := mysqlengine.NewFromDB(db)
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
		FROM meta_stream_log WHERE collection = ?`, col).
		Scan(&eventID, &version, &encoding); err != nil {
		t.Fatalf("read backfilled columns: %v", err)
	}

	if eventID != myTypedEventID1 || version != 1 || encoding != "cbor" {
		t.Fatalf("backfilled identity wrong: %q %d %q", eventID, version, encoding)
	}

	// NO-RECREATE PROOF: the migration's ALTERs are pinned to
	// ALGORITHM=INSTANT — the server refuses a rebuild-shaped ALTER — and the
	// seeded row's physical identity (same seq, same value bytes) survived.
	var afterSeq int64

	var afterValue string

	if err := db.QueryRow(`SELECT seq, value FROM meta_stream_log WHERE collection = ?`, col).
		Scan(&afterSeq, &afterValue); err != nil {
		t.Fatalf("post-migration read: %v", err)
	}

	if afterSeq != beforeSeq || afterValue != beforeValue {
		t.Fatalf("journal row changed across migration: seq %d->%d", beforeSeq, afterSeq)
	}

	// Post-migration, the unique indexes enforce identity: re-appending the
	// same envelope conflicts instead of silently doubling the journal.
	backend := eng.(metaengine.StreamLogBackend)

	err = backend.StreamAppend(context.Background(), col, "Task:"+myTypedStreamID,
		[]any{typedEnvelope(t, myTypedEventID1, myTypedStreamID, 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("post-migration duplicate append error = %v, want ErrVersionConflict", err)
	}

	// Reads are unchanged: the envelope stays the read path.
	values, err := backend.StreamRead(context.Background(), col, "Task:"+myTypedStreamID)
	if err != nil {
		t.Fatalf("post-migration read: %v", err)
	}

	if len(values) != 1 {
		t.Fatalf("post-migration stream length = %d, want 1", len(values))
	}
}

// legacyScratchDSN provisions a throwaway database on the test server and
// returns its DSN (dropped on cleanup). The shared cqrs_test database already
// carries the typed schema, which would silently degrade the ALTER/backfill
// pin to a no-op.
func legacyScratchDSN(t *testing.T) string {
	t.Helper()

	base := os.Getenv("MYSQL_TEST_DSN")
	if base == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	name := "twl_" + runSuffix

	admin, err := sql.Open("mysql", base)
	if err != nil {
		t.Fatalf("open admin conn: %v", err)
	}

	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + name); err != nil {
		_ = admin.Close()
		t.Fatalf("drop stale scratch database: %v", err)
	}

	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		_ = admin.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name)
		_ = admin.Close()
	})

	// Swap the database segment, preserving any ?params suffix.
	idx := strings.LastIndex(base, "/")
	tail := base[idx+1:]
	params := ""
	if q := strings.Index(tail, "?"); q >= 0 {
		params = tail[q:]
	}

	return base[:idx+1] + name + params
}

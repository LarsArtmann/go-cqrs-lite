package sqliteengine_test

// Stored-bytes pins for the wave-4 value-encoding change (T1).
//
// Round-trip tests cannot see a PERSISTED-FORM regression: a writer that
// switches a plain string to a quoted JSON string, re-marshals a value through
// a different codec, or encodes a batch per row instead of up front still
// reads back the same Go value. These tests read the RAW columns a real SQLite
// engine wrote and assert the exact bytes, so "the change did not alter what
// lands on disk" is a fact rather than an argument.

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type pinValue struct {
	Title string `json:"title"`
	N     int    `json:"n"`
}

// Literal expectations, deliberately not derived from the encoder: the point
// is to catch an encoder that agrees with itself.
const (
	pinValueJSON  = `{"title":"pin","n":7}`
	pinStringJSON = `raw-string`
	pinIntJSON    = `42`
)

func openPinDB(t *testing.T) *sql.DB {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())

	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// A single connection keeps the shared-cache in-memory database alive for
	// the engine handle and the raw reader below.
	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func openPinEngine(t *testing.T, opts ...sqliteengine.EngineOption) (*sql.DB, metaengine.Engine) {
	t.Helper()

	db := openPinDB(t)

	eng, err := sqliteengine.NewSQLiteEngine(db, opts...)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return db, eng
}

// TestStoredBytes_StreamAppend pins the stream-log column bytes, including the
// batch shape: each value is stored verbatim (string values unquoted) in its
// own row, in caller order.
func TestStoredBytes_StreamAppend(t *testing.T) {
	ctx := context.Background()

	db, eng := openPinEngine(t)

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.StreamLogBackend")
	}

	values := []any{pinValue{Title: "pin", N: 7}, "raw-string", 42}
	if err := backend.StreamAppend(ctx, "pin-stream", "s-1", values); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	rows, err := db.QueryContext(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = ? AND stream_id = ? ORDER BY seq`,
		"pin-stream", "s-1")
	if err != nil {
		t.Fatalf("query stream rows: %v", err)
	}

	defer func() { _ = rows.Close() }()

	want := []string{pinValueJSON, pinStringJSON, pinIntJSON}

	var got []string

	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan stream row: %v", err)
		}

		got = append(got, v)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate stream rows: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("stored %d rows, want %d: %q", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stream row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestStoredBytes_MapSet pins the meta_map value column for a JSON value,
// including the key form: keys stay plain strings, values stay JSON text.
func TestStoredBytes_MapSet(t *testing.T) {
	ctx := context.Background()

	db, eng := openPinEngine(t)

	base, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.MapBackend")
	}

	if err := base.MapSet(ctx, "pin-map", "k1", pinValue{Title: "pin", N: 7}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	var key, value string

	err := db.QueryRowContext(ctx,
		`SELECT key, value FROM meta_map WHERE collection = ?`, "pin-map").Scan(&key, &value)
	if err != nil {
		t.Fatalf("query map row: %v", err)
	}

	if key != "k1" {
		t.Errorf("stored key = %q, want %q", key, "k1")
	}

	if value != pinValueJSON {
		t.Errorf("stored value = %q, want %q", value, pinValueJSON)
	}
}

// TestStoredBytes_MapSetAt pins the version row: the latest view and the
// versioned cell must carry the SAME bytes, so a time-travel read cannot
// disagree with the current read.
func TestStoredBytes_MapSetAt(t *testing.T) {
	ctx := context.Background()

	ts := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)

	db, eng := openPinEngine(t, sqliteengine.WithCellVersioning())

	versioned, ok := eng.(metaengine.VersionedWriter)
	if !ok {
		t.Fatal("sqlite engine with cell versioning must implement metaengine.VersionedWriter")
	}

	if err := versioned.MapSetAt(
		ctx,
		"pin-versioned",
		"k1",
		pinValue{Title: "pin", N: 7},
		ts,
	); err != nil {
		t.Fatalf("MapSetAt: %v", err)
	}

	var latest string

	err := db.QueryRowContext(
		ctx,
		`SELECT value FROM meta_map WHERE collection = ? AND key = ?`,
		"pin-versioned",
		"k1",
	).Scan(&latest)
	if err != nil {
		t.Fatalf("query latest row: %v", err)
	}

	var versionedValue string

	err = db.QueryRowContext(ctx,
		`SELECT value FROM meta_cell_versions WHERE collection = ? AND key = ? AND ts = ?`,
		"pin-versioned", "k1", ts.UnixNano()).Scan(&versionedValue)
	if err != nil {
		t.Fatalf("query version row: %v", err)
	}

	if latest != pinValueJSON {
		t.Errorf("latest value = %q, want %q", latest, pinValueJSON)
	}

	if versionedValue != latest {
		t.Errorf("version value = %q, want the latest bytes %q", versionedValue, latest)
	}
}

package sqliteengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// sweepRow inserts one synthetic typed journal row with a fully controlled
// occurred_at (unix nanos), bypassing the envelope pipeline so the sweep
// boundary, the NULL-timestamp rule, and the snapshot co-trim can be pinned
// precisely.
func sweepRow(t *testing.T, db *sql.DB, col, sid string, occurred sql.NullInt64) {
	t.Helper()

	_, err := db.Exec(`INSERT INTO meta_stream_log
		(collection, stream_id, value, occurred_at) VALUES (?, ?, ?, ?)`,
		col, sid, fmt.Sprintf(`{"sid":%q}`, sid), occurred)
	if err != nil {
		t.Fatalf("insert sweep row %s: %v", sid, err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()

	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}

	return n
}

// TestSQLiteTrimJournalBefore pins the engine's age-sweep contract (#118):
// rows older than the cutoff are deleted, newer rows stay, rows without a
// typed occurred_at (legacy, NULL) are conservatively KEPT, and snapshots of
// streams the sweep removed rows from are co-deleted while untouched
// streams' snapshots survive.
func TestSQLiteTrimJournalBefore(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_sweep_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	sweeper, ok := eng.(metaengine.JournalSweeper)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.JournalSweeper")
	}

	ctx := context.Background()

	now := time.Now()
	old := now.Add(-48 * time.Hour).UnixNano()
	fresh := now.Add(-1 * time.Minute).UnixNano()

	sweepRow(t, db, "events", "old-stream", sql.NullInt64{Int64: old, Valid: true})
	sweepRow(t, db, "events", "mixed-stream", sql.NullInt64{Int64: old, Valid: true})
	sweepRow(t, db, "events", "mixed-stream", sql.NullInt64{Int64: fresh, Valid: true})
	sweepRow(t, db, "events", "legacy-stream", sql.NullInt64{}) // NULL occurred_at
	sweepRow(t, db, "events", "fresh-stream", sql.NullInt64{Int64: fresh, Valid: true})

	for _, sid := range []string{"old-stream", "mixed-stream", "fresh-stream"} {
		if _, err := db.Exec(
			`INSERT INTO meta_snapshot (collection, stream_id, version, data) VALUES (?, ?, ?, ?)`,
			"snapshots", sid, 1, []byte("{}"),
		); err != nil {
			t.Fatalf("insert snapshot %s: %v", sid, err)
		}
	}

	removed, err := sweeper.TrimJournalBefore(ctx, "events", "snapshots", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("TrimJournalBefore: %v", err)
	}

	if removed != 2 {
		t.Fatalf("removed %d rows, want 2 (old-stream's row + mixed-stream's old row)", removed)
	}

	if got := countRows(t, db, `SELECT COUNT(*) FROM meta_stream_log WHERE collection = 'events'`); got != 3 {
		t.Fatalf("%d journal rows remain, want 3 (mixed-new + legacy + fresh)", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = 'events' AND stream_id = 'old-stream'`); got != 0 {
		t.Fatalf("old-stream kept %d rows, want 0", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = 'events' AND stream_id = 'legacy-stream'`); got != 1 {
		t.Fatalf("legacy NULL-timestamp row kept %d, want 1 (conservatively kept)", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_snapshot WHERE collection = 'snapshots' AND stream_id = 'old-stream'`); got != 0 {
		t.Fatalf("swept stream's snapshot kept %d, want 0 (co-trimmed)", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_snapshot WHERE collection = 'snapshots' AND stream_id = 'mixed-stream'`); got != 0 {
		t.Fatalf("partially swept stream's snapshot kept %d, want 0 (co-trimmed)", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_snapshot WHERE collection = 'snapshots' AND stream_id = 'fresh-stream'`); got != 1 {
		t.Fatalf("untouched stream's snapshot kept %d, want 1", got)
	}
}

// TestSQLiteTrimStream pins the per-stream erasure contract (#118): every
// journal row of the stream plus its snapshot go, other streams' rows stay.
func TestSQLiteTrimStream(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_trimstream_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	trimmer, ok := eng.(metaengine.StreamTrimmer)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.StreamTrimmer")
	}

	ctx := context.Background()
	ts := sql.NullInt64{Int64: time.Now().UnixNano(), Valid: true}

	sweepRow(t, db, "events", "gone", ts)
	sweepRow(t, db, "events", "gone", ts)
	sweepRow(t, db, "events", "stays", ts)

	if _, err := db.Exec(
		`INSERT INTO meta_snapshot (collection, stream_id, version, data) VALUES (?, ?, ?, ?)`,
		"snapshots", "gone", 2, []byte("{}"),
	); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	removed, err := trimmer.TrimStream(ctx, "events", "snapshots", "gone")
	if err != nil {
		t.Fatalf("TrimStream: %v", err)
	}

	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = 'events' AND stream_id = 'gone'`); got != 0 {
		t.Fatalf("trimmed stream kept %d rows, want 0", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_snapshot WHERE collection = 'snapshots' AND stream_id = 'gone'`); got != 0 {
		t.Fatalf("trimmed stream's snapshot kept %d, want 0", got)
	}

	if got := countRows(t, db,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = 'events' AND stream_id = 'stays'`); got != 1 {
		t.Fatalf("untouched stream kept %d rows, want 1", got)
	}
}

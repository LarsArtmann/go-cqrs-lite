package sqliteengine_test

// Planned-table sweep tests (dnsblockd metaflood MF3.2): the bulk by-time
// planned-table trim (TrimPlannedBefore) must delete exactly the rows whose
// extracted INTEGER column is older than the cutoff, refuse non-INTEGER
// columns and unknown fields/collections, and flush buffered planned writes
// before sweeping (the delete-after-set contract, batch form).

import (
	"context"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestTrimPlannedBefore_RemovesOnlyOlderRows(t *testing.T) {
	t.Parallel()

	store, _, _ := newBatchTestStore(t, 0)
	ctx := context.Background()

	if err := store.ApplyBatch(ctx, batchRows(100)); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	// batchRows seeds createdUnix = 1_700_000_000 + i; cutoff at +50 keeps
	// rows 0..49 removable and 50..99 alive.
	removed, err := store.TrimPlannedBefore(ctx, "batch_tracks", "createdUnix", 1_700_000_050)
	if err != nil {
		t.Fatalf("TrimPlannedBefore: %v", err)
	}

	if removed != 50 {
		t.Fatalf("removed = %d, want 50", removed)
	}

	var alive int
	if err := queryCount(t, store, "batch_tracks", &alive); err != nil {
		t.Fatalf("count after trim: %v", err)
	}

	if alive != 50 {
		t.Fatalf("rows alive after trim = %d, want 50", alive)
	}

	// Idempotent: a second sweep at the same cutoff removes nothing.
	removed, err = store.TrimPlannedBefore(ctx, "batch_tracks", "createdUnix", 1_700_000_050)
	if err != nil {
		t.Fatalf("second TrimPlannedBefore: %v", err)
	}

	if removed != 0 {
		t.Fatalf("second sweep removed = %d, want 0", removed)
	}
}

func TestTrimPlannedBefore_RejectsNonIntegerColumn(t *testing.T) {
	t.Parallel()

	store, _, _ := newBatchTestStore(t, 0)
	ctx := context.Background()

	// "domain" IS an extracted column, but TEXT-typed: the numeric cutoff
	// comparison would be dishonest, so the sweep must refuse it.
	if _, err := store.TrimPlannedBefore(ctx, "batch_tracks", "domain", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject TEXT-typed columns")
	}
}

func TestTrimPlannedBefore_RejectsUnknownFieldAndCollection(t *testing.T) {
	t.Parallel()

	store, _, _ := newBatchTestStore(t, 0)
	ctx := context.Background()

	if _, err := store.TrimPlannedBefore(ctx, "batch_tracks", "clientIp", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject a field with no extracted column")
	}

	if _, err := store.TrimPlannedBefore(ctx, "no_such_collection", "createdUnix", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject an unknown collection")
	}
}

func TestTrimPlannedBefore_FlushesBufferedWritesFirst(t *testing.T) {
	t.Parallel()

	db := openTrimDB(t)
	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	q := trimOnlyQuery()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	// One buffered batch (not yet flushed — chunk cap 500 not reached), then
	// an immediate sweep: the buffered rows must be flushed BEFORE the sweep
	// or they would resurrect after it committed.
	if err := store.ApplyBatch(ctx, batchRows(30)); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	removed, err := store.TrimPlannedBefore(ctx, "batch_tracks", "createdUnix", 1_700_000_010)
	if err != nil {
		t.Fatalf("TrimPlannedBefore: %v", err)
	}

	if removed != 10 {
		t.Fatalf("removed = %d, want 10 (buffered rows must be swept, not resurrected)", removed)
	}
}

// --- helpers -------------------------------------------------------------

func openTrimDB(t *testing.T) interface {
	Exec(string, ...any) (interface{ RowsAffected() (int64, error) }, error)
} {
	t.Helper()

	db, err := openTrimSQL(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	return db
}

func queryCount(t *testing.T, store *metaengine.Store, table string, out *int) error {
	t.Helper()

	row, err := store.TrimPlannedBefore(t.Context(), table, "__count_probe__", 0)
	_ = row

	if err == nil {
		t.Fatal("probe should not succeed")
	}

	return countViaRawDB(t, table, out)
}

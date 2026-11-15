package sqliteengine_test

// Planned-table sweep tests (dnsblockd metaflood MF3.2): the bulk by-time
// planned-table trim (TrimPlannedBefore) must delete exactly the rows whose
// extracted INTEGER column is older than the cutoff, refuse non-INTEGER
// columns and unknown fields/collections, and flush buffered planned writes
// before sweeping (the delete-after-set contract, batch form).

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

func countPlannedRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var n int

	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}

func TestTrimPlannedBefore_RemovesOnlyOlderRows(t *testing.T) {
	t.Parallel()

	store, _, db := newBatchTestStore(t, -1)
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

	if alive := countPlannedRows(t, db, "meta_planned_batch_tracks"); alive != 50 {
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

	store, _, _ := newBatchTestStore(t, -1)

	// "domain" IS an extracted column, but TEXT-typed: the numeric cutoff
	// comparison would be dishonest, so the sweep must refuse it.
	if _, err := store.TrimPlannedBefore(t.Context(), "batch_tracks", "domain", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject TEXT-typed columns")
	}
}

func TestTrimPlannedBefore_RejectsUnknownFieldAndCollection(t *testing.T) {
	t.Parallel()

	store, _, _ := newBatchTestStore(t, -1)

	// The fixture query filters domain/blocked but never extracts clientIp.
	if _, err := store.TrimPlannedBefore(t.Context(), "batch_tracks", "clientIp", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject a field with no extracted column")
	}

	if _, err := store.TrimPlannedBefore(t.Context(), "no_such_collection", "createdUnix", 42); err == nil {
		t.Fatal("TrimPlannedBefore must reject an unknown collection")
	}
}

func TestTrimPlannedBefore_FlushesBufferedWritesFirst(t *testing.T) {
	t.Parallel()

	store, _, db := newBatchTestStore(t, -1)
	ctx := context.Background()

	// The delete-after-set contract, sweep form: inside ONE ambient tx,
	// fold writes sit in the planned buffer (chunk cap 500 not reached);
	// the sweep must flush them BEFORE its DELETE executes, or they would
	// resurrect after the sweep's rows-affected was counted.
	var removed int64

	err := store.InTransaction(ctx, func(ctx context.Context) error {
		if err := store.ApplyBatch(ctx, batchRows(30)); err != nil {
			return err
		}

		if got := countPlannedRows(t, db, "meta_planned_batch_tracks"); got != 0 {
			t.Fatalf("precondition: expected 0 flushed rows mid-tx, got %d (fixture no longer buffers)", got)
		}

		var trimErr error

		removed, trimErr = store.TrimPlannedBefore(ctx, "batch_tracks", "createdUnix", 1_700_000_010)

		return trimErr
	})
	if err != nil {
		t.Fatalf("InTransaction sweep: %v", err)
	}

	if removed != 10 {
		t.Fatalf("removed = %d, want 10 (buffered rows must be swept, not resurrected)", removed)
	}

	if alive := countPlannedRows(t, db, "meta_planned_batch_tracks"); alive != 20 {
		t.Fatalf("rows alive after trim = %d, want 20", alive)
	}
}

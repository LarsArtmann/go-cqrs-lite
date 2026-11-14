package sqliteengine_test

// MF1 batch-write contract tests (dnsblockd metaflood, MF1.4): the
// one-tx-per-batch + buffered multi-row planned writes must be
// indistinguishable from sequential single-row execution, except faster.
//
// Pins (design memo: dnsblockd docs/research/2026-11-14_metaflood-mf1-bulk-write-contract.md):
//   - batch atomicity: a mid-batch fold failure rolls back the WHOLE batch
//   - chunked landing: 1200 rows (chunk cap 500) all land with correct values
//   - duplicate keys in one batch: last write wins, same as sequential
//   - read-your-writes: MapGet/scan inside the tx observes buffered rows
//   - delete-after-set: a delete following a buffered set wins
//   - InTransaction+Apply works (pre-MF1.2 it hit the nested-tx rejection)

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// newBatchTestStore builds a file-backed sqliteengine + Store with one
// planned-table query ("batch_tracks") whose fold fails on demand for the
// configured fail ID (folds signal failure by panicking an error).
func newBatchTestStore(t *testing.T, failOnID int64) (*metaengine.Store, metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "batch.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	errFailFold := errors.New("batch test: intentional fold failure")

	q := metaengine.Query[spikeMF1List, spikeMF1Row]("batch_tracks",
		metaengine.Volume(1_000_000),
		metaengine.FilterOnField[spikeMF1Row]("domain", metaengine.FilterEq),
		metaengine.FilterOnField[spikeMF1Row]("blocked", metaengine.FilterEq),
		metaengine.SortOnField[spikeMF1Row]("createdUnix", true),
		metaengine.OnRecordTyped("track.recorded", spikeMF1Row{},
			func(_ record.Record, e spikeMF1Row) (int64, spikeMF1Row) {
				if e.ID == failOnID {
					panic(errFailFold)
				}

				return e.ID, e
			}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store, eng, db
}

func batchRows(n int) []metaengine.EventInput {
	events := make([]metaengine.EventInput, 0, n)
	for i := range n {
		events = append(events, metaengine.EventInput{
			Type: "track.recorded",
			Payload: spikeMF1Row{
				ID:          int64(i),
				Domain:      fmt.Sprintf("host%d.example.net", i%97),
				ClientIP:    fmt.Sprintf("10.0.0.%d", i%251),
				Blocked:     i%3 == 0,
				CreatedUnix: 1_700_000_000 + int64(i),
			},
		})
	}

	return events
}

func TestPlannedBatch_MidBatchFailureRollsBackWholeBatch(t *testing.T) {
	t.Parallel()

	store, _, db := newBatchTestStore(t, 25)
	ctx := context.Background()

	err := store.ApplyBatch(ctx, batchRows(50))
	if err == nil {
		t.Fatal("ApplyBatch must fail when a fold panics mid-batch")
	}

	var table string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'meta_planned_batch%'`,
	).Scan(&table); err != nil {
		// Table may exist (created at registration) even with zero rows.
		t.Fatalf("planned table missing: %v", err)
	}

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}

	if n != 0 {
		t.Fatalf("batch atomicity violated: %d rows committed despite mid-batch failure (want 0)", n)
	}
}

func TestPlannedBatch_ChunkedRowsAllLand(t *testing.T) {
	t.Parallel()

	store, _, db := newBatchTestStore(t, -1)
	ctx := context.Background()

	// 1200 rows = 500 + 500 + 200 chunks; append a duplicate key at the end
	// with a different value to pin last-write-wins.
	events := batchRows(1200)
	events = append(events, metaengine.EventInput{
		Type: "track.recorded",
		Payload: spikeMF1Row{
			ID:          600,
			Domain:      "overwritten.example.net",
			CreatedUnix: 42,
		},
	})

	if err := store.ApplyBatch(ctx, events); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	var table string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'meta_planned_batch%'`,
	).Scan(&table); err != nil {
		t.Fatalf("planned table missing: %v", err)
	}

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}

	if n != 1200 {
		t.Fatalf("chunked landing broken: %d rows, want 1200 (duplicate key must replace, not add)", n)
	}

	var domain string
	if err := db.QueryRow(
		fmt.Sprintf("SELECT domain FROM %s WHERE key = ?", table), "600",
	).Scan(&domain); err != nil {
		t.Fatalf("dup-key readback: %v", err)
	}

	if domain != "overwritten.example.net" {
		t.Fatalf("duplicate key in one batch must resolve last-write-wins, got domain %q", domain)
	}

	var first, last string
	if err := db.QueryRow(
		fmt.Sprintf("SELECT domain FROM %s WHERE key = ?", table), "0",
	).Scan(&first); err != nil {
		t.Fatalf("first row readback: %v", err)
	}

	if err := db.QueryRow(
		fmt.Sprintf("SELECT domain FROM %s WHERE key = ?", table), "1199",
	).Scan(&last); err != nil {
		t.Fatalf("last row readback: %v", err)
	}

	if first != "host0.example.net" || last != "host35.example.net" { // 1199 % 97 == 35
		t.Fatalf("row corruption across chunks: first=%q last=%q", first, last)
	}
}

func TestPlannedBatch_ReadYourWritesInsideTransaction(t *testing.T) {
	t.Parallel()

	store, eng, _ := newBatchTestStore(t, -1)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	err := store.InTransaction(ctx, func(ctx context.Context) error {
		if err := store.Apply(ctx, "track.recorded", spikeMF1Row{
			ID: 7, Domain: "ryw.example.net", CreatedUnix: 1,
		}); err != nil {
			return err
		}

		v, found, err := mb.MapGet(ctx, "batch_tracks", int64(7))
		if err != nil {
			return err
		}

		if !found {
			return errors.New("planned MapGet inside tx must observe the buffered row (flush-before-read)")
		}

		row, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("planned MapGet returned %T, want map[string]any", v)
		}

		if row["domain"] != "ryw.example.net" {
			return fmt.Errorf("buffered row value mismatch: %+v", row)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("InTransaction read-your-writes: %v", err)
	}

	v, found, err := mb.MapGet(ctx, "batch_tracks", int64(7))
	if err != nil || !found {
		t.Fatalf("post-commit read: found=%v err=%v", found, err)
	}

	if v.(map[string]any)["domain"] != "ryw.example.net" { //nolint:forcetypeassert
		t.Fatalf("post-commit value mismatch: %+v", v)
	}
}

func TestPlannedBatch_DeleteAfterBufferedSetWins(t *testing.T) {
	t.Parallel()

	store, eng, _ := newBatchTestStore(t, -1)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	err := store.InTransaction(ctx, func(ctx context.Context) error {
		if err := store.Apply(ctx, "track.recorded", spikeMF1Row{
			ID: 9, Domain: "doomed.example.net", CreatedUnix: 1,
		}); err != nil {
			return err
		}

		return mb.MapDelete(ctx, "batch_tracks", int64(9))
	})
	if err != nil {
		t.Fatalf("InTransaction delete-after-set: %v", err)
	}

	_, found, err := mb.MapGet(ctx, "batch_tracks", int64(9))
	if err != nil {
		t.Fatalf("post-commit read: %v", err)
	}

	if found {
		t.Fatal("delete-after-set ordering violated: the buffered set resurrected the deleted row")
	}
}

// TestPlannedBatch_InTransactionApplyRegression pins the pre-MF1.2 bug: the
// documented Store.InTransaction{Apply...} pattern returned
// "sqliteengine.RunInTx: nested transactions are not supported" because
// dispatch's per-event tx wrap re-entered RunInTx.
func TestPlannedBatch_InTransactionApplyRegression(t *testing.T) {
	t.Parallel()

	store, _, _ := newBatchTestStore(t, -1)
	ctx := context.Background()

	err := store.InTransaction(ctx, func(ctx context.Context) error {
		return store.Apply(ctx, "track.recorded", spikeMF1Row{
			ID: 11, Domain: "regression.example.net", CreatedUnix: 1,
		})
	})
	if err != nil {
		t.Fatalf("InTransaction+Apply must succeed post-MF1.2 (ambient-tx join): %v", err)
	}
}

// TestPlannedBatch_ExtractedColumnsMatchSequential pins that multi-row
// buffered writes populate extracted columns identically to the pre-batch
// single-row path: same values, same column set, filterable.
func TestPlannedBatch_ExtractedColumnsMatchSequential(t *testing.T) {
	t.Parallel()

	store, _, db := newBatchTestStore(t, -1)
	ctx := context.Background()

	if err := store.ApplyBatch(ctx, batchRows(30)); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	var table string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'meta_planned_batch%'`,
	).Scan(&table); err != nil {
		t.Fatalf("planned table missing: %v", err)
	}

	var blockedCount int
	if err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE blocked = 1", table),
	).Scan(&blockedCount); err != nil {
		t.Fatalf("extracted-column filter: %v", err)
	}

	// batchRows marks every third row blocked (0,3,...,27) = 10 of 30.
	if blockedCount != 10 {
		t.Fatalf("extracted column blocked: %d rows, want 10", blockedCount)
	}

	var created int64
	if err := db.QueryRow(
		fmt.Sprintf("SELECT createdUnix FROM %s WHERE key = ?", table), "17",
	).Scan(&created); err != nil {
		t.Fatalf("extracted int column read: %v", err)
	}

	if created != 1_700_000_017 {
		t.Fatalf("createdUnix for key 17: %d, want 1700000017", created)
	}
}

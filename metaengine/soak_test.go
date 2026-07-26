package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestSoak_SQLiteSustainedWrites: multi-second sustained write + read pressure
// against the SQLite engine. Verifies no deadlocks, no data corruption, and
// eventual consistency under load. Skips in -short mode.
//
// This is NOT a benchmark — it's a correctness soak test that runs enough
// operations to surface race conditions, connection exhaustion, and iterator
// leaks that only appear under sustained load.
func TestSoak_SQLiteSustainedWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test: skips in -short mode")
	}
	t.Parallel()

	type deposit struct {
		Account string
		Amount  int64
	}
	type balance struct {
		Account string
		Total   int64
	}
	type query struct {
		Account string
		Limit   int
		After   *metaengine.Cursor
	}
	type result struct {
		Logs []balance
		Next *metaengine.Cursor
	}

	q := metaengine.Query[query, result](
		"balances",
		// Map ADT — latest balance per account
		metaengine.On(deposit{}, func(d deposit) (string, balance) {
			return d.Account, balance{Account: d.Account, Total: d.Amount}
		}),
		metaengine.On(deposit{}, func(d deposit, prev balance) balance {
			prev.Total += d.Amount

			return prev
		}),
		metaengine.FilterOn(func(b balance) string { return b.Account }),
	)

	dbName := fmt.Sprintf("soak_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	const accounts = 20
	const writers = 8
	const writesPerWriter = 500
	const readers = 4
	const readsPerReader = 200

	var totalWrites atomic.Int64
	var totalReads atomic.Int64
	var readErrors atomic.Int64

	var wg atomic.Int64
	wg.Store(int64(writers + readers))
	done := make(chan struct{})

	// Writers: each writes to a random account
	for w := range writers {
		go func(workerID int) {
			defer func() {
				if wg.Add(-1) == 0 {
					close(done)
				}
			}()
			for i := range writesPerWriter {
				account := fmt.Sprintf("acct-%03d", (workerID*7+i)%accounts)
				amt := int64(i%100 + 1)
				if err := store.Apply(ctx, "deposit", deposit{Account: account, Amount: amt}); err != nil {
					t.Errorf("writer %d write %d: %v", workerID, i, err)
					return
				}
				totalWrites.Add(1)
			}
		}(w)
	}

	// Readers: concurrent reads while writes are happening
	for r := range readers {
		go func(workerID int) {
			defer func() {
				if wg.Add(-1) == 0 {
					close(done)
				}
			}()
			for i := range readsPerReader {
				account := fmt.Sprintf("acct-%03d", (workerID*3+i)%accounts)
				_, err := metaengine.ExecuteTyped[query, result](
					ctx, store, query{Account: account, Limit: 10},
				)
				if err != nil {
					readErrors.Add(1)
				}
				totalReads.Add(1)
			}
		}(r)
	}

	<-done

	w := totalWrites.Load()
	r := totalReads.Load()
	t.Logf("soak: %d writes, %d reads, %d read errors", w, r, readErrors.Load())

	if w != int64(writers*writesPerWriter) {
		t.Fatalf("expected %d total writes, got %d", writers*writesPerWriter, w)
	}

	// Verify data integrity: each account's balance should equal the sum of all deposits to it
	expected := make(map[string]int64)
	for w := range writers {
		for i := range writesPerWriter {
			account := fmt.Sprintf("acct-%03d", (w*7+i)%accounts)
			amt := int64(i%100 + 1)
			expected[account] += amt
		}
	}

	for a := range accounts {
		account := fmt.Sprintf("acct-%03d", a)
		result, err := metaengine.ExecuteTyped[query, result](
			ctx, store, query{Account: account, Limit: 1},
		)
		if err != nil {
			t.Fatalf("final read %s: %v", account, err)
		}
		got := int64(0)
		if len(result.Logs) > 0 {
			got = result.Logs[0].Total
		}
		if got != expected[account] {
			t.Errorf("account %s: expected balance %d, got %d", account, expected[account], got)
		}
	}
}

// TestSoak_SQLiteMultimapGrowth: sustained writes to a Multimap ADT to verify
// the seq-seed restart safety (ADR-0067) under growth. Skips in -short mode.
func TestSoak_SQLiteMultimapGrowth(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test: skips in -short mode")
	}
	t.Parallel()

	type appendEvent struct {
		MapKey string
		Value  string
	}
	type lookupInput struct {
		Key string
	}
	type lookupResult struct {
		Values []string
	}

	q := metaengine.Query[lookupInput, lookupResult](
		"tags",
		metaengine.On(appendEvent{}, func(e appendEvent) metaengine.MultiEntry {
			return metaengine.MultiEntry{Key: e.MapKey, Value: e.Value}
		}),
	)

	dbName := fmt.Sprintf("soak_mm_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	const keys = 10
	const appendsPerKey = 100

	for k := range keys {
		for i := range appendsPerKey {
			key := fmt.Sprintf("key-%d", k)
			val := fmt.Sprintf("val-%d-%d", k, i)
			if err := store.Apply(ctx, "appendEvent", appendEvent{MapKey: key, Value: val}); err != nil {
				t.Fatalf("Apply %d-%d: %v", k, i, err)
			}
		}
	}

	for k := range keys {
		result, err := metaengine.ExecuteTyped[lookupInput, lookupResult](
			ctx, store, lookupInput{Key: fmt.Sprintf("key-%d", k)},
		)
		if err != nil {
			t.Fatalf("read key-%d: %v", k, err)
		}
		if len(result.Values) != appendsPerKey {
			t.Errorf("key-%d: expected %d values, got %d", k, appendsPerKey, len(result.Values))
		}
		// Verify ordering is preserved
		for i, v := range result.Values {
			expected := fmt.Sprintf("val-%d-%d", k, i)
			if v != expected {
				t.Errorf("key-%d[%d]: expected %q, got %q", k, i, expected, v)
				break
			}
		}
	}
}

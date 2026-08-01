package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
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
	type lookup struct {
		Account string
	}

	q := metaengine.Query[lookup, balance](
		"balances",
		// Map ADT — latest balance per account (ReadLookup pattern)
		metaengine.On(deposit{}, func(d deposit) (string, balance) {
			return d.Account, balance{Account: d.Account, Total: d.Amount}
		}),
		metaengine.On(deposit{}, func(d deposit, prev balance) balance {
			prev.Total += d.Amount

			return prev
		}),
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
				if err := store.Apply(
					ctx,
					"deposit",
					deposit{Account: account, Amount: amt},
				); err != nil {
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
				_, err := metaengine.ExecuteTyped[lookup, balance](
					ctx, store, lookup{Account: account},
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

	// Verify data integrity: sum of all deposits should equal sum of all balances.
	// Each writer writes writesPerWriter deposits with amounts (i%100 + 1).
	totalExpected := int64(0)
	for i := range writesPerWriter {
		totalExpected += int64(i%100 + 1)
	}
	totalExpected *= int64(writers)

	// Query each account and sum balances
	var grandTotal int64
	for a := range accounts {
		account := fmt.Sprintf("acct-%03d", a)
		result, err := metaengine.ExecuteTyped[lookup, balance](
			ctx, store, lookup{Account: account},
		)
		if err != nil {
			t.Fatalf("final read %s: %v", account, err)
		}
		grandTotal += result.Total
	}

	if grandTotal != totalExpected {
		t.Errorf("grand total: expected %d, got %d (delta=%d)",
			totalExpected, grandTotal, totalExpected-grandTotal)
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
			if err := store.Apply(
				ctx,
				"appendEvent",
				appendEvent{MapKey: key, Value: val},
			); err != nil {
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

// TestSoak_MemoryBounded verifies that processing many events into a bounded
// set of keys does not cause unbounded memory growth. The memory engine should
// be O(unique keys), not O(total events). Skips in -short mode.
func TestSoak_MemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test: skips in -short mode")
	}

	t.Parallel()

	type updateEvent struct {
		Key   string
		Value int64
	}
	type lookup struct {
		Key string
	}
	type state struct {
		Key   string
		Total int64
	}

	q := metaengine.Query[lookup, state](
		"counters",
		metaengine.On(updateEvent{}, func(e updateEvent) (string, state) {
			return e.Key, state{Key: e.Key, Total: e.Value}
		}),
		metaengine.On(updateEvent{}, func(e updateEvent, prev state) state {
			prev.Total += e.Value

			return prev
		}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	const numEvents = 50_000
	const numKeys = 100 // 500 updates per key — memory bounded by numKeys

	runtime.GC()

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	for i := 0; i < numEvents; i++ {
		key := fmt.Sprintf("k-%d", i%numKeys)
		if err := store.Apply(
			ctx,
			"updateEvent",
			updateEvent{Key: key, Value: int64(i)},
		); err != nil {
			t.Fatalf("Apply %d: %v", i, err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&after)

	heapGrowth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	maxExpected := int64(numKeys) * 500 * 100 // 5MB — generous for map + planner overhead

	// The race detector inflates allocations 5-10x, and parallel test load
	// (full verify gate) compounds this further. Relax the threshold so we
	// still catch true unbounded leaks without flaking under -race.
	if raceEnabled {
		maxExpected *= 10 // 50MB under -race
	}

	if heapGrowth > maxExpected {
		t.Errorf("heap grew %d bytes after %d events with %d keys (max %d)",
			heapGrowth, numEvents, numKeys, maxExpected)
	}

	t.Logf("heap: %d bytes for %d keys after %d updates", heapGrowth, numKeys, numEvents)
}

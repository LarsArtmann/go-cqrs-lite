package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"testing"


	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Cross-engine parity tests for Counter and Set ADTs.
// The existing cross_engine_meta_test.go covers Map/Multimap/Log/struct
// results; concurrent_gaps_test.go covers LogTail and (partially) Graph.
// These tests fill the Counter and Set gaps (TODO_LIST "Cross-engine parity
// tests for metaengine ADTs").

// TestCrossEngineCounterParity — CounterIncrement + CounterGet produce
// identical results across memory and SQLite engines.
func TestCrossEngineCounterParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deltas := []metaengine.Delta{
		{"alpha": 1, "beta": 5},
		{"alpha": 2, "gamma": 3},
		{"beta": -3, "gamma": 1},
		{"alpha": 10},
	}

	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
		"sqlite": mustSQLiteEngine(t),
	}

	results := make(map[string]map[string]int64, len(engines))
	var resultsMu sync.Mutex

	for name, eng := range engines {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cb, ok := eng.(metaengine.CounterBackend)
			if !ok {
				t.Fatalf("%s engine does not implement CounterBackend", name)
			}

			for i, d := range deltas {
				if err := cb.CounterIncrement(ctx, "counters", d); err != nil {
					t.Fatalf("CounterIncrement[%d]: %v", i, err)
				}
			}

			got, err := cb.CounterGet(ctx, "counters")
			if err != nil {
				t.Fatalf("CounterGet: %v", err)
			}

			resultsMu.Lock()
			results[name] = got
			resultsMu.Unlock()

			// Expected: alpha=1+2+10=13, beta=5-3=2, gamma=3+1=4
			want := map[string]int64{"alpha": 13, "beta": 2, "gamma": 4}
			assertCounterEq(t, name, got, want)
		})
	}

	// Cross-engine deep-equal (both must agree on every key+value).
	if len(results) == len(engines) {
		assertCrossEngineCountersEq(t, results)
	}
}

// TestCrossEngineSetParity — SetAdd + SetContains produce identical membership
// results across memory and SQLite engines.
func TestCrossEngineSetParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	add := []string{"apple", "banana", "cherry", "date"}
	probe := []string{"apple", "banana", "cherry", "date", "elderberry", "fig"}

	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
		"sqlite": mustSQLiteEngine(t),
	}

	results := make(map[string]map[string]bool, len(engines))
	var setResultsMu sync.Mutex

	for name, eng := range engines {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sb, ok := eng.(metaengine.SetBackend)
			if !ok {
				t.Fatalf("%s engine does not implement SetBackend", name)
			}

			for _, k := range add {
				if err := sb.SetAdd(ctx, "fruits", k); err != nil {
					t.Fatalf("SetAdd(%s): %v", k, err)
				}
			}

			// Idempotency: re-adding an element must not error.
			if err := sb.SetAdd(ctx, "fruits", "apple"); err != nil {
				t.Fatalf("SetAdd(apple) re-add not idempotent: %v", err)
			}

			setResultsMu.Lock()
			results[name] = make(map[string]bool, len(probe))
			setResultsMu.Unlock()

			for _, k := range probe {
				got, err := sb.SetContains(ctx, "fruits", k)
				if err != nil {
					t.Fatalf("SetContains(%s): %v", k, err)
				}

				setResultsMu.Lock()
				results[name][k] = got
				setResultsMu.Unlock()
			}
		})
	}

	if len(results) == len(engines) {
		assertCrossEngineSetEq(t, results)
	}
}

func assertCounterEq(
	t *testing.T,
	engine string,
	got, want map[string]int64,
) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: CounterGet returned %d keys, want %d (got=%v want=%v)",
			engine, len(got), len(want), got, want)
	}

	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("%s: CounterGet missing key %q", engine, k)
		}

		if gv != wv {
			t.Fatalf("%s: counter[%s] = %d, want %d", engine, k, gv, wv)
		}
	}
}

func assertCrossEngineCountersEq(
	t *testing.T,
	results map[string]map[string]int64,
) {
	t.Helper()

	if len(results) < 2 {
		return
	}

	// Collect all keys for a deterministic comparison order.
	keys := make(map[string]struct{})
	for _, r := range results {
		for k := range r {
			keys[k] = struct{}{}
		}
	}

	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Strings(sortedKeys)

	var memVals, sqliteVals []string

	for _, k := range sortedKeys {
		memVals = append(memVals, fmt.Sprintf("%s=%d", k, results["memory"][k]))
		sqliteVals = append(sqliteVals, fmt.Sprintf("%s=%d", k, results["sqlite"][k]))
	}

	if fmt.Sprint(memVals) != fmt.Sprint(sqliteVals) {
		t.Fatalf("cross-engine counter divergence:\n  memory=%v\n  sqlite=%v", memVals, sqliteVals)
	}
}

func assertCrossEngineSetEq(
	t *testing.T,
	results map[string]map[string]bool,
) {
	t.Helper()

	if len(results) < 2 {
		return
	}

	for k, memHit := range results["memory"] {
		sqlHit := results["sqlite"][k]
		if memHit != sqlHit {
			t.Fatalf("cross-engine set divergence on key %q: memory=%v sqlite=%v",
				k, memHit, sqlHit)
		}
	}
}

// TestCrossEngineSortedMapParity — FilterOn + SortOn scan (the ADTSortedMap
// path via ScanBackend.MapScan) produces identical ordered results across
// memory and SQLite engines, including limit truncation.
func TestCrossEngineSortedMapParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tasks := []TaskCreated{
		{ID: "s1", Title: "low", Assignee: "a", Status: "open", Priority: 5},
		{ID: "s2", Title: "high", Assignee: "b", Status: "open", Priority: 1},
		{ID: "s3", Title: "mid", Assignee: "c", Status: "open", Priority: 3},
		{ID: "s4", Title: "done", Assignee: "d", Status: "done", Priority: 2},
		{ID: "s5", Title: "urgent", Assignee: "e", Status: "open", Priority: 0},
	}

	// Expected open tasks sorted by priority ascending: s5(0), s2(1), s3(3), s1(5).
	wantIDs := []TaskID{"s5", "s2", "s3", "s1"}

	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
		"sqlite": newIsolatedSQLiteEngine(t),
	}

	scanResults := make(map[string][]TaskID, len(engines))
	limitResults := make(map[string][]TaskID, len(engines))
	var resultsMu sync.Mutex

	for name, eng := range engines {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, err := metaengine.Plan(
				[]metaengine.Engine{eng},
				listTasksByStatusQuery(),
			)
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			defer store.Close()

			for _, tc := range tasks {
				if err := store.Apply(ctx, "TaskCreated", tc); err != nil {
					t.Fatalf("Apply(%s): %v", tc.ID, err)
				}
			}

			// Full scan — exercises MapScan (SortedMap ADT).
			result, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 10},
			)
			if err != nil {
				t.Fatalf("ExecuteTyped: %v", err)
			}

			if len(result.Tasks) != len(wantIDs) {
				t.Fatalf("got %d tasks, want %d", len(result.Tasks), len(wantIDs))
			}

			ids := make([]TaskID, len(result.Tasks))
			for i, task := range result.Tasks {
				ids[i] = task.ID
				if task.ID != wantIDs[i] {
					t.Fatalf("task[%d].ID = %s, want %s (sorted by priority ascending)",
						i, task.ID, wantIDs[i])
				}
			}
			resultsMu.Lock()
			scanResults[name] = ids
			resultsMu.Unlock()

			// Limit truncation — only first 2 tasks.
			limited, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 2},
			)
			if err != nil {
				t.Fatalf("ExecuteTyped(limit=2): %v", err)
			}

			if len(limited.Tasks) != 2 {
				t.Fatalf("limited scan: got %d tasks, want 2", len(limited.Tasks))
			}

			limitIDs := make([]TaskID, len(limited.Tasks))
			for i, task := range limited.Tasks {
				limitIDs[i] = task.ID
			}

			if limitIDs[0] != "s5" || limitIDs[1] != "s2" {
				t.Fatalf("limited scan order: got %s,%s want s5,s2",
					limitIDs[0], limitIDs[1])
			}
			resultsMu.Lock()
			limitResults[name] = limitIDs
			resultsMu.Unlock()
		})
	}

	// Cross-engine deep-equal: both engines must produce identical ordered results.
	if len(scanResults) == len(engines) {
		assertCrossEngineScanEq(t, scanResults, "full scan")
		assertCrossEngineScanEq(t, limitResults, "limit=2 scan")
	}
}

func assertCrossEngineScanEq(t *testing.T, results map[string][]TaskID, label string) {
	t.Helper()

	mem, ok := results["memory"]
	if !ok {
		t.Fatalf("missing memory engine results for %s", label)
	}

	sqlite, ok := results["sqlite"]
	if !ok {
		t.Fatalf("missing sqlite engine results for %s", label)
	}

	if len(mem) != len(sqlite) {
		t.Fatalf("%s: memory has %d items, sqlite has %d", label, len(mem), len(sqlite))
	}

	for i := range mem {
		if mem[i] != sqlite[i] {
			t.Fatalf("%s: position %d differs: memory=%s sqlite=%s",
				label, i, mem[i], sqlite[i])
		}
	}
}

// newIsolatedSQLiteEngine creates a SQLite engine backed by a private
// in-memory database (NOT shared-cache), ensuring no cross-test interference.
func newIsolatedSQLiteEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng
}

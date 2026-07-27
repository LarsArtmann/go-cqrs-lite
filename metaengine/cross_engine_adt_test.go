package metaengine_test

import (
	"context"
	"fmt"
	"sort"
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

			results[name] = got

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

			results[name] = make(map[string]bool, len(probe))

			for _, k := range probe {
				got, err := sb.SetContains(ctx, "fruits", k)
				if err != nil {
					t.Fatalf("SetContains(%s): %v", k, err)
				}

				results[name][k] = got
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

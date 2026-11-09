package metaengine

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type soakTick struct{ N int }

func soakCounterQuery(name string) any {
	return Query[struct{}, map[string]int64](
		name,
		OnRecord(soakTick{}, func(_ record.Record, _ soakTick) Delta {
			return Delta{"n": 1}
		}),
	)
}

// TestFoldLocks_ConcurrentApplySoak hammers one event type that feeds counter
// folds in TWO queries concurrently. Any lost update or fold-dispatch race
// corrupts the final counts.
// Run with -race -count=3 (METAENGINE-LAYOUT-ROLES.md §7).
func TestFoldLocks_ConcurrentApplySoak(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer DeferClose(eng)

	store, err := Plan([]Engine{eng}, soakCounterQuery("soak_a"), soakCounterQuery("soak_b"))
	if err != nil {
		t.Fatal(err)
	}
	defer DeferClose(store)

	ctx := context.Background()

	const goroutines = 16
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			for j := range iterations {
				if err := store.Apply(ctx, "soakTick", soakTick{N: j}); err != nil {
					t.Errorf("apply: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	cb, ok := eng.(CounterBackend)
	if !ok {
		t.Fatal("memory engine does not implement CounterBackend")
	}

	want := int64(goroutines * iterations)

	for _, col := range []string{"soak_a", "soak_b"} {
		counts, err := cb.CounterGet(ctx, col)
		if err != nil {
			t.Fatalf("counter get %s: %v", col, err)
		}

		if got := counts["n"]; got != want {
			t.Fatalf("%s: lost updates — want %d, got %d", col, want, got)
		}
	}
}

// TestFoldLocks_PerQueryIsolation verifies that per-query locks are actually
// per query: the same lock instance is returned for one name, different
// instances for different names.
func TestFoldLocks_PerQueryIsolation(t *testing.T) {
	t.Parallel()

	fl := newFoldLocks()

	a1 := fl.get("alpha")
	a2 := fl.get("alpha")
	if a1 != a2 {
		t.Fatal("same query name must return the same lock instance")
	}

	if fl.get("beta") == a1 {
		t.Fatal("different query names must return different lock instances")
	}
}

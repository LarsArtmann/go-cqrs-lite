package metaengine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ════════════ REGRESSION: SORTED PAGINATION PAST LIMIT BOUNDARY ════════════
// Bug #1: MapScan sorted before full scan, losing correct items past limit.
// Fix: collect ALL matching pairs, sort, THEN truncate.

type ScoreItem struct {
	ID    string
	Score int
	When  time.Time
}

type ScoreInput struct {
	Category string
	Limit    int
}

type ScoreResult struct {
	Items []ScoreItem
	Next  *Cursor
}

func TestRegression_SortedPagination(t *testing.T) {
	query := Query[ScoreInput, ScoreResult](
		"scores",
		On(ScoreItem{}, func(e ScoreItem) (string, ScoreItem) {
			return e.ID, e
		}),
		SortOn(func(r ScoreItem) time.Time { return r.When }),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// Insert 10 items with random When timestamps.
	base := time.Now()
	delays := []time.Duration{5, 1, 10, 3, 8, 2, 7, 4, 9, 6}
	for i, d := range delays {
		err := store.Apply("ScoreItem", ScoreItem{
			ID:    fmt.Sprintf("item%d", i),
			Score: int(d),
			When:  base.Add(d * time.Hour),
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	ctx := context.Background()

	// Request limit=5 — should get the 5 earliest items by When.
	page, err := ExecuteTyped[ScoreInput, ScoreResult](ctx, store,
		ScoreInput{Limit: 5})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(page.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(page.Items))
	}

	// Verify items are sorted by When ascending.
	for i := 1; i < len(page.Items); i++ {
		if page.Items[i-1].When.After(page.Items[i].When) {
			t.Errorf("sort violation at position %d: %v after %v",
				i, page.Items[i-1].When, page.Items[i].When)
		}
	}

	// The earliest item should be the one with delay=1 (item1).
	if page.Items[0].ID != "item1" {
		t.Errorf("expected first item to be item1 (earliest), got %s", page.Items[0].ID)
	}
}

// ════════════ REGRESSION: CONCURRENT FoldUpdate ATOMICITY ════════════
// Bug #2: FoldUpdate read-modify-write was not atomic.
// Fix: MapUpdater interface with atomic MapUpdate.

type CounterEvent struct {
	ID     string
	Amount int
}

type CounterValue struct {
	ID    string
	Total int
}

type CounterInput struct {
	ID string
}

func TestRegression_ConcurrentFoldUpdate(t *testing.T) {
	query := Query[CounterInput, CounterValue](
		"counters",
		On(CounterEvent{}, func(e CounterEvent) (string, CounterValue) {
			return e.ID, CounterValue{ID: e.ID, Total: e.Amount}
		}),
		On(CounterEvent{}, func(e CounterEvent, prev CounterValue) CounterValue {
			prev.Total += e.Amount
			return prev
		}),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// Fire 100 concurrent updates to the same key.
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.Apply("CounterEvent", CounterEvent{ID: "counter1", Amount: 1})
		}()
	}
	wg.Wait()

	ctx := context.Background()
	result, err := ExecuteTyped[CounterInput, CounterValue](
		ctx,
		store,
		CounterInput{ID: "counter1"},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// All 100 increments should be counted (atomic RMW).
	if result.Total != 100 {
		t.Errorf("expected Total=100 after 100 concurrent +1 updates, got %d", result.Total)
	}
}

// ════════════ REGRESSION: DETERMINISTIC SORT ORDER ════════════
// Bug #3: Go map iteration order is random, causing nondeterministic scan results.
// Fix: secondary sort by map key as string for stable tiebreaker.

type StableItem struct {
	ID    string
	Score int
}

type StableInput struct {
	Limit int
}

type StableResult struct {
	Items []StableItem
	Next  *Cursor
}

func TestRegression_StableOrder(t *testing.T) {
	query := Query[StableInput, StableResult](
		"stable_items",
		On(StableItem{}, func(e StableItem) (string, StableItem) {
			return e.ID, e
		}),
		SortOn(func(r StableItem) int { return r.Score }),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// Insert items with equal scores — tiebreaker must be deterministic.
	for i := range 10 {
		err := store.Apply("StableItem", StableItem{
			ID:    fmt.Sprintf("item%02d", i),
			Score: 42, // all same score — tests tiebreaker
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	ctx := context.Background()

	// Run the scan 5 times — all must produce identical order.
	var firstIDs []string
	for run := range 5 {
		page, err := ExecuteTyped[StableInput, StableResult](ctx, store, StableInput{Limit: 100})
		if err != nil {
			t.Fatalf("Execute run %d: %v", run, err)
		}

		if len(page.Items) != 10 {
			t.Fatalf("run %d: expected 10 items, got %d", run, len(page.Items))
		}

		if run == 0 {
			firstIDs = make([]string, len(page.Items))
			for i, item := range page.Items {
				firstIDs[i] = item.ID
			}
		} else {
			for i, item := range page.Items {
				if item.ID != firstIDs[i] {
					t.Errorf(
						"run %d: item %d ID mismatch: got %s, expected %s (nondeterministic order)",
						run,
						i,
						item.ID,
						firstIDs[i],
					)
				}
			}
		}
	}
}

// ════════════ APPLY ENCODED (JSON payload) ════════════

func TestApplyEncoded(t *testing.T) {
	store, err := Plan([]Engine{NewMemoryEngine()}, findUserQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	payload := `{"ID":"u1","Email":"alice@test.com","Name":"Alice","Country":"SE","At":"2026-01-01T00:00:00Z"}`
	err = store.ApplyEncoded("UserCreated", []byte(payload))
	if err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	ctx := context.Background()
	result, err := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Name != "Alice" {
		t.Errorf("expected Name=Alice, got %q", result.Name)
	}
}

// ════════════ PLAN DIAGNOSTICS ════════════

func TestPlanDiagnostics_GraphDegradation(t *testing.T) {
	// Graph query on memory engine → O(degree^depth) which is NOT degraded.
	// But if we add a scan-only engine profile, it should warn.
	query := friendsOfQuery()

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	plan := store.Plan()
	if len(plan.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(plan.Queries))
	}

	// Memory engine supports graph at O(degree^depth) — not degraded.
	q := plan.Queries[0]
	if q.Complexity != ComplexityODegree {
		t.Errorf("expected O(degree^depth), got %s", q.Complexity)
	}

	t.Log(plan.Report())
}

// ════════════ ON CLASSIFICATION ════════════

func TestOnClassification(t *testing.T) {
	type Event struct {
		ID   string
		Name string
		Prev string
	}
	type Result struct{ Name string }

	sample := Event{}

	tests := []struct {
		name    string
		handler any
		kind    FoldKind
	}{
		{"insert (K,V)", On(sample, func(e Event) (string, Result) {
			return e.ID, Result{Name: e.Name}
		}), FoldInsert},
		{"update (e,prev)→V", On(sample, func(e Event, prev Result) Result {
			prev.Name = e.Name
			return prev
		}), FoldUpdate},
		{"set K", On(sample, func(e Event) string { return e.ID }), FoldSet},
		{"count Delta", On(sample, func(e Event) Delta {
			return Delta{"count": +1}
		}), FoldCount},
		{"edge", On(sample, func(e Event) Edge {
			return Edge{From: e.ID, To: e.Name}
		}), FoldEdge},
		{"remove", On(sample, Remove[Result]()), FoldRemove},
		{"skip", On(sample, func(e Event) Skip { return Skip{} }), FoldSkip},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.handler.(Fold).Kind != tc.kind {
				t.Errorf("expected %s, got %s", tc.kind, tc.handler.(Fold).Kind)
			}
		})
	}
}

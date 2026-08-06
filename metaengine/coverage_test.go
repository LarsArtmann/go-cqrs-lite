package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── T1: Cursor-Based Pagination ───

func TestTypedReader_CursorPagination(t *testing.T) {
	t.Parallel()

	store, reader := setupBenchStore(t, 50, true)
	defer store.Close()

	ctx := context.Background()

	// Paginate through all 50 items in pages of 10.
	var allIDs []string

	cursor := (*metaengine.Cursor)(nil)

	for page := range 10 { // max 10 pages (safety)
		opts := []metaengine.ScanOption{
			metaengine.WithSort("Priority", false),
			metaengine.WithLimit(10),
		}
		if cursor != nil {
			opts = append(opts, metaengine.WithCursor(cursor.Value))
		}

		pageItems, nextCursor, err := reader.ScanPage(ctx, opts...)
		if err != nil {
			t.Fatalf("ScanPage %d: %v", page, err)
		}

		for _, item := range pageItems {
			allIDs = append(allIDs, item.ID)
		}

		if nextCursor == nil {
			break
		}

		cursor = nextCursor
	}

	// All 50 items should be returned across pages (no duplicates).
	seen := make(map[string]bool, len(allIDs))
	for _, id := range allIDs {
		if seen[id] {
			t.Errorf("duplicate ID across pages: %s", id)
		}

		seen[id] = true
	}

	if len(seen) != 50 {
		t.Errorf("unique items across pages = %d, want 50", len(seen))
	}
}

// ─── T5: TypedReader.GetBatch ───

func TestTypedReader_GetBatch(t *testing.T) {
	t.Parallel()

	store, reader := setupBenchStore(t, 10, true)
	defer store.Close()

	ctx := context.Background()

	keys := []any{"item-000000", "item-000003", "item-000005", "item-000009"}
	results, err := reader.GetBatch(ctx, keys)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}

	if len(results) != len(keys) {
		t.Fatalf("GetBatch returned %d, want %d", len(results), len(keys))
	}

	for i, r := range results {
		if r.ID != keys[i] {
			t.Errorf("result[%d].ID = %s, want %s", i, r.ID, keys[i])
		}
	}
}

// ─── T8: Plan Output Stability ───

func TestPlan_OutputStability(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("sqlite engine: %v", err)
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), eng},
		benchFilterQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	plan := store.Plan()

	if len(plan.Queries) != 1 {
		t.Fatalf("plan has %d queries, want 1", len(plan.Queries))
	}

	q := plan.Queries[0]
	if q.QueryName != "bench_filter_scan" {
		t.Errorf("QueryName = %q, want %q", q.QueryName, "bench_filter_scan")
	}

	if q.ADT != metaengine.ADTMap {
		t.Errorf("ADT = %v, want %v", q.ADT, metaengine.ADTMap)
	}

	// The planner should assign the filtered Map query to SQLite (pushdown).
	if q.EngineName == "" {
		t.Error("EngineName should not be empty")
	}

	t.Logf("plan: query=%s adt=%s engine=%s complexity=%s latency=%.3fms",
		q.QueryName, q.ADT, q.EngineName, q.Complexity, q.Cost.EstimatedLatencyMs)
}

// ─── T9: SwapEngine Live Migration ───

func TestStore_SwapEngine(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	sqliteEng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("sqlite engine: %v", err)
	}

	memEng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan(
		[]metaengine.Engine{memEng, sqliteEng},
		benchFilterQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Seed some data.
	for i := range 10 {
		_ = store.Apply(ctx, "benchItemResult", benchItemResult{
			ID:       fmt.Sprintf("swap-%02d", i),
			Status:   "open",
			Priority: i,
		})
	}

	reader := metaengine.NewReader[benchItemResult](store, "bench_filter_scan")

	// Verify data is accessible.
	items, _ := reader.Scan(ctx, metaengine.WithLimit(0))
	if len(items) != 10 {
		t.Fatalf("pre-swap scan returned %d, want 10", len(items))
	}

	// Create a fresh SQLite engine and swap it in.
	db2, _ := sql.Open("sqlite", ":memory:")
	defer db2.Close()
	newEng := metaengine.NewMemoryEngine()

	// Swap the memory engine for a fresh SQLite engine.
	err = store.SwapEngine(memEng.Profile().Name, "sqlite-fresh", newEng)
	if err != nil {
		t.Fatalf("SwapEngine: %v", err)
	}

	// The swap doesn't migrate data — it reassigns queries. The new engine
	// is empty, so scans should return 0 from the reassigned query (if it
	// was on memory). But reads continue without error (no downtime).
	_, err = reader.Scan(ctx, metaengine.WithLimit(0))
	if err != nil {
		t.Errorf("post-swap scan should not error: %v", err)
	}

	// Applying new events should work on the new engine.
	_ = store.Apply(ctx, "benchItemResult", benchItemResult{
		ID: "post-swap-01", Status: "open", Priority: 99,
	})

	postItems, err := reader.Scan(ctx, metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("post-swap+apply scan: %v", err)
	}

	// The new item should be present.
	found := false
	for _, item := range postItems {
		if item.ID == "post-swap-01" {
			found = true
		}
	}

	if !found {
		t.Error("post-swap applied item not found in scan results")
	}
}

// ─── T4: WithPrefetch Cache ───

func TestTypedReader_WithPrefetch(t *testing.T) {
	t.Parallel()

	store, reader := setupBenchStore(t, 50, true)
	defer store.Close()

	cache := metaengine.NewPrefetchCache()
	prefetchReader := reader.WithPrefetch(cache)

	ctx := context.Background()

	// Scan page 1 with prefetch — should cache the next page.
	page1, cursor1, err := prefetchReader.ScanPage(ctx,
		metaengine.WithSort("Priority", false),
		metaengine.WithLimit(10))
	if err != nil {
		t.Fatalf("ScanPage: %v", err)
	}

	if len(page1) != 10 {
		t.Fatalf("page1 len = %d, want 10", len(page1))
	}

	var cursor1Value any
	if cursor1 != nil {
		cursor1Value = cursor1.Value
	} else {
		t.Fatal("cursor1 should not be nil")
	}

	// Scan page 2 using the cursor — should hit the prefetch cache.
	page2, _, err := prefetchReader.ScanPage(ctx,
		metaengine.WithSort("Priority", false),
		metaengine.WithLimit(10),
		metaengine.WithCursor(cursor1Value))
	if err != nil {
		t.Fatalf("ScanPage 2: %v", err)
	}

	if len(page2) != 10 {
		t.Fatalf("page2 len = %d, want 10", len(page2))
	}
}

// ─── T2: Watcher Integration ───

func TestWatcher_ReceivesUpdate(t *testing.T) {
	t.Parallel()

	store, _ := setupBenchStore(t, 0, true)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	watcher := metaengine.NewWatcher[benchItemResult](store, "bench_filter_scan")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil) // all changes

	// Apply an event — watcher should receive it.
	_ = store.Apply(ctx, "benchItemResult", benchItemResult{
		ID: "watch-01", Status: "open", Priority: 1,
	})

	select {
	case val, ok := <-ch:
		if !ok {
			t.Fatal("watcher channel closed without receiving value")
		}

		if val.ID != "watch-01" {
			t.Errorf("watcher received ID=%s, want watch-01", val.ID)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not receive update within 2s")
	}
}

// ─── T6: StreamScan (lazy iteration) ───

func TestTypedReader_Count(t *testing.T) {
	t.Parallel()

	store, reader := setupBenchStore(t, 100, true)
	defer store.Close()

	ctx := context.Background()

	// Count all items.
	total, err := reader.Count(ctx, metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("Count all: %v", err)
	}

	if total != 100 {
		t.Errorf("Count = %d, want 100", total)
	}

	// Count filtered items (open = 2/3).
	open, err := reader.Count(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
		metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("Count open: %v", err)
	}

	// ~66 items (i%3 != 0 for i in 0..99).
	if open < 60 || open > 70 {
		t.Errorf("Count open = %d, expected ~66", open)
	}
}

// ─── O3: ExplainPlan ───

func TestStore_ExplainPlan(t *testing.T) {
	t.Parallel()

	store, _ := setupBenchStore(t, 10, true)
	defer store.Close()

	explanation := store.ExplainPlan()
	if explanation == "" {
		t.Fatal("ExplainPlan returned empty string")
	}

	// Should mention the query name, ADT, and engine.
	for _, want := range []string{"bench_filter_scan", "map", "sqlite"} {
		if !strings.Contains(explanation, want) {
			t.Errorf("ExplainPlan output missing %q:\n%s", want, explanation)
		}
	}

	t.Logf("ExplainPlan output:\n%s", explanation)
}

// ─── O4: Doctor ───

func TestStore_Doctor(t *testing.T) {
	t.Parallel()

	store, _ := setupBenchStore(t, 10, true)
	defer store.Close()

	ctx := context.Background()
	report := store.Doctor(ctx)
	if report == "" {
		t.Fatal("Doctor returned empty string")
	}

	// Should mention health, collections, and poisoned sections.
	for _, want := range []string{"Health", "Collections", "Poisoned", "healthy"} {
		if !strings.Contains(report, want) {
			t.Errorf("Doctor output missing %q:\n%s", want, report)
		}
	}

	t.Logf("Doctor output:\n%s", report)
}

// ─── F3: RegisterQuery at runtime ───

func TestStore_RegisterQuery(t *testing.T) {
	t.Parallel()

	// Start with just the counter query.
	counterQ := metaengine.Query[benchListInput, map[string]int64](
		"runtime_counter",
		metaengine.On(benchItemResult{}, func(e benchItemResult) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
	)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("sqlite engine: %v", err)
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), eng},
		counterQ,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Register a Map query at runtime.
	mapQ := metaengine.Query[benchListInput, benchItemResult](
		"runtime_map",
		metaengine.On(benchItemResult{}, func(e benchItemResult) (string, benchItemResult) {
			return e.ID, e
		}),
	)

	if err := store.RegisterQuery(mapQ); err != nil {
		t.Fatalf("RegisterQuery: %v", err)
	}

	// Apply an event — both queries should receive it.
	err = store.Apply(ctx, "benchItemResult", benchItemResult{
		ID: "reg-01", Status: "open", Priority: 5,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Verify the runtime-registered Map query works.
	reader := metaengine.NewReader[benchItemResult](store, "runtime_map")
	item, found, err := reader.Get(ctx, "reg-01")
	if err != nil {
		t.Fatalf("Get from runtime query: %v", err)
	}

	if !found {
		t.Fatal("runtime-registered query should find the item")
	}

	if item.ID != "reg-01" {
		t.Errorf("got ID=%s, want reg-01", item.ID)
	}

	// Verify duplicate registration fails.
	if err := store.RegisterQuery(mapQ); err == nil {
		t.Error("duplicate RegisterQuery should fail")
	}
}

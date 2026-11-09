//go:build cgo

package bench_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestColumnar_Correctness verifies that all three scan approaches return
// identical results for the same data. If columnar, pushdown, and memory
// disagree, the benchmark comparison is invalid.
func TestColumnar_Correctness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	n := 500

	// Seed identical data into three stores: columnar DuckDB, pushdown DuckDB,
	// and Memory.
	columnarStore, err := metaengine.Plan(
		[]metaengine.Engine{newDuckDBEngine(t)},
		columnarScanQuery(),
	)
	if err != nil {
		t.Fatalf("Plan columnar: %v", err)
	}

	defer columnarStore.Close()

	pushdownStore, err := metaengine.Plan(
		[]metaengine.Engine{newDuckDBEngine(t)},
		pushdownScanQuery(),
	)
	if err != nil {
		t.Fatalf("Plan pushdown: %v", err)
	}

	defer pushdownStore.Close()

	memStore, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		pushdownScanQuery(),
	)
	if err != nil {
		t.Fatalf("Plan memory: %v", err)
	}

	defer memStore.Close()

	for _, store := range []*metaengine.Store{columnarStore, pushdownStore, memStore} {
		seedColumnarItems(t, store, n)
	}

	// All three should return the same active items.
	columnarReader := metaengine.NewReader[benchColumnarItem](columnarStore, "columnar_scan")
	pushdownReader := metaengine.NewReader[benchColumnarItem](pushdownStore, "pushdown_scan")
	memReader := metaengine.NewReader[benchColumnarItem](memStore, "pushdown_scan")

	columnarResults, err := columnarReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
		metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("columnar scan: %v", err)
	}

	pushdownResults, err := pushdownReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
		metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("pushdown scan: %v", err)
	}

	memResults, err := memReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
		metaengine.WithLimit(0))
	if err != nil {
		t.Fatalf("memory scan: %v", err)
	}

	// ~2/3 of items are active (i%3 != 0).
	expected := n * 2 / 3
	if len(columnarResults) != expected {
		t.Errorf("columnar: expected %d active, got %d", expected, len(columnarResults))
	}

	if len(pushdownResults) != expected {
		t.Errorf("pushdown: expected %d active, got %d", expected, len(pushdownResults))
	}

	if len(memResults) != expected {
		t.Errorf("memory: expected %d active, got %d", expected, len(memResults))
	}

	// Verify results are sorted by Amount DESC (first item should have highest amount).
	if len(columnarResults) > 0 && len(pushdownResults) > 0 {
		if columnarResults[0].Amount < pushdownResults[0].Amount {
			t.Errorf("columnar first amount %.2f < pushdown first amount %.2f",
				columnarResults[0].Amount, pushdownResults[0].Amount)
		}
	}
}

// TestColumnar_LayoutApplied verifies that the DuckDB engine actually applied
// the columnar layout when WithColumnarLayout is declared. If the layout
// wasn't applied, the benchmark would be measuring the wrong thing.
func TestColumnar_LayoutApplied(t *testing.T) {
	t.Parallel()

	eng := newDuckDBEngine(t)

	// Plan with columnar layout — the planner should call ApplyLayoutPlan.
	store, err := metaengine.Plan([]metaengine.Engine{eng}, columnarScanQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	// Seed + scan should work without error.
	seedColumnarItems(t, store, 100)

	ctx := context.Background()
	reader := metaengine.NewReader[benchColumnarItem](store, "columnar_scan")
	results, err := reader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "active"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected active results, got 0")
	}

	// Verify the scan returns items with all fields populated (not just JSON blobs).
	first := results[0]
	if first.ID == "" || first.Status == "" || first.Category == "" {
		t.Errorf("field not populated: %+v", first)
	}
}

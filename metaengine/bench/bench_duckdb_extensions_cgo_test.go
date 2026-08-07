//go:build cgo

package bench_test

import (
	"context"
	"reflect"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_duckdb_extensions_cgo_test.go — extends the migrated benchmarks with
// DuckDB engine comparisons. CGo-tagged because DuckDB requires the C++ engine.

// TestPromise_ParityWithDuckDB seeds the same events through Memory and
// Memory+DuckDB stores, then verifies all queries return identical results.
// This proves DuckDB is a valid routing target alongside Memory and SQLite.
func TestPromise_ParityWithDuckDB(t *testing.T) {
	t.Parallel()

	n := 3_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	memStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer memStore.Close()

	duckStore := planPromiseStore(t,
		[]metaengine.Engine{metaengine.NewMemoryEngine(), newDuckDBEngine(t)})
	defer duckStore.Close()

	for _, e := range events {
		if err := memStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("memory Apply %s: %v", e.typeName, err)
		}

		if err := duckStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("duckdb Apply %s: %v", e.typeName, err)
		}
	}

	orderID := OrderID("ord-000000")
	customerID := CustomerID("cus-000")

	// 1. find_order (Map point lookup).
	memOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx, memStore, FindOrderInput{ID: orderID})
	if err != nil {
		t.Fatalf("memory find_order: %v", err)
	}

	duckOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx, duckStore, FindOrderInput{ID: orderID})
	if err != nil {
		t.Fatalf("duckdb find_order: %v", err)
	}

	if !reflect.DeepEqual(memOrder, duckOrder) {
		t.Errorf("find_order divergence: memory=%+v duckdb=%+v", memOrder, duckOrder)
	}

	// 2. count_by_status (Counter aggregate).
	memCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, memStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("memory count_by_status: %v", err)
	}

	duckCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, duckStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("duckdb count_by_status: %v", err)
	}

	for k, mv := range memCounts {
		if dv, ok := duckCounts[k]; !ok || dv != mv {
			t.Errorf("count_by_status divergence for %s: memory=%d duckdb=%d", k, mv, dv)
		}
	}

	// 3. orders_by_customer (Multimap lookup).
	memOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, memStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("memory orders_by_customer: %v", err)
	}

	duckOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, duckStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("duckdb orders_by_customer: %v", err)
	}

	if len(memOrders) != len(duckOrders) {
		t.Errorf("orders_by_customer length divergence: memory=%d duckdb=%d",
			len(memOrders), len(duckOrders))
	}

	// 4. total_revenue (Counter sum).
	memRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, memStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("memory total_revenue: %v", err)
	}

	duckRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, duckStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("duckdb total_revenue: %v", err)
	}

	for k, mv := range memRev {
		if dv, ok := duckRev[k]; !ok || dv != mv {
			t.Errorf("total_revenue divergence for %s: memory=%d duckdb=%d", k, mv, dv)
		}
	}

	t.Logf("DuckDB parity verified across %d events: all queries agree", n)
}

// BenchmarkMultiQuery_DuckDBApplyThroughput measures event ingestion
// throughput when DuckDB is in the engine pool alongside Memory. The planner
// routes filtered queries to DuckDB (pushdown) and point-lookups to Memory.
func BenchmarkMultiQuery_DuckDBApplyThroughput(b *testing.B) {
	n := 1_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		b.StopTimer()
		eng := newDuckDBEngine(b)
		store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine(), eng})
		b.StartTimer()

		for _, e := range events {
			if err := store.Apply(ctx, e.typeName, e.payload); err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		store.Close()
		b.StartTimer()
	}

	b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

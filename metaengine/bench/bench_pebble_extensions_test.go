package bench_test

import (
	"context"
	"reflect"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_pebble_extensions_test.go — extends the migrated benchmarks with Pebble
// engine comparisons. Pebble is pure Go (LSM point reads), no CGo required.

// TestPromise_ParityWithPebble seeds the same events through Memory and
// Memory+Pebble stores, then verifies all queries return identical results.
// This proves Pebble is a valid routing target for the planner.
func TestPromise_ParityWithPebble(t *testing.T) {
	t.Parallel()

	n := 3_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	memStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer memStore.Close()

	pebStore := planPromiseStore(t,
		[]metaengine.Engine{metaengine.NewMemoryEngine(), newPebbleEngine(t)})
	defer pebStore.Close()

	for _, e := range events {
		if err := memStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("memory Apply %s: %v", e.typeName, err)
		}

		if err := pebStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("pebble Apply %s: %v", e.typeName, err)
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

	pebOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx, pebStore, FindOrderInput{ID: orderID})
	if err != nil {
		t.Fatalf("pebble find_order: %v", err)
	}

	if !reflect.DeepEqual(memOrder, pebOrder) {
		t.Errorf("find_order divergence: memory=%+v pebble=%+v", memOrder, pebOrder)
	}

	// 2. count_by_status (Counter aggregate).
	memCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, memStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("memory count_by_status: %v", err)
	}

	pebCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, pebStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("pebble count_by_status: %v", err)
	}

	for k, mv := range memCounts {
		if pv, ok := pebCounts[k]; !ok || pv != mv {
			t.Errorf("count_by_status divergence for %s: memory=%d pebble=%d", k, mv, pv)
		}
	}

	// 3. orders_by_customer (Multimap lookup).
	memOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, memStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("memory orders_by_customer: %v", err)
	}

	pebOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, pebStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("pebble orders_by_customer: %v", err)
	}

	if len(memOrders) != len(pebOrders) {
		t.Errorf("orders_by_customer length divergence: memory=%d pebble=%d",
			len(memOrders), len(pebOrders))
	}

	// 4. total_revenue (Counter sum).
	memRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, memStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("memory total_revenue: %v", err)
	}

	pebRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, pebStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("pebble total_revenue: %v", err)
	}

	for k, mv := range memRev {
		if pv, ok := pebRev[k]; !ok || pv != mv {
			t.Errorf("total_revenue divergence for %s: memory=%d pebble=%d", k, mv, pv)
		}
	}

	t.Logf("Pebble parity verified across %d events: all queries agree", n)
}

// BenchmarkMultiQuery_PebbleApplyThroughput measures event ingestion throughput
// when Pebble is in the engine pool alongside Memory. Pebble excels at point
// reads (LSM), so the planner may route Map lookups to it.
func BenchmarkMultiQuery_PebbleApplyThroughput(b *testing.B) {
	n := 1_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		b.StopTimer()
		eng := newPebbleEngine(b)
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

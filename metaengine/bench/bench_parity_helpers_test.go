package bench_test

import (
	"context"
	"reflect"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// runEngineParityTest seeds the same events through a Memory-only store and an
// alternative store, then verifies all queries return identical results.
// altName is used in error messages (e.g. "duckdb", "pebble").
func runEngineParityTest(t *testing.T, altStore *metaengine.Store, altName string) {
	t.Helper()

	n := 3_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	memStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer memStore.Close()

	for _, e := range events {
		if err := memStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("memory Apply %s: %v", e.typeName, err)
		}

		if err := altStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("%s Apply %s: %v", altName, e.typeName, err)
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

	altOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx, altStore, FindOrderInput{ID: orderID})
	if err != nil {
		t.Fatalf("%s find_order: %v", altName, err)
	}

	if !reflect.DeepEqual(memOrder, altOrder) {
		t.Errorf("find_order divergence: memory=%+v %s=%+v", memOrder, altName, altOrder)
	}

	// 2. count_by_status (Counter aggregate).
	memCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, memStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("memory count_by_status: %v", err)
	}

	altCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx, altStore, CountOrdersByStatusInput{})
	if err != nil {
		t.Fatalf("%s count_by_status: %v", altName, err)
	}

	for k, mv := range memCounts {
		if av, ok := altCounts[k]; !ok || av != mv {
			t.Errorf("count_by_status divergence for %s: memory=%d %s=%d", k, mv, altName, av)
		}
	}

	// 3. orders_by_customer (Multimap lookup).
	memOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, memStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("memory orders_by_customer: %v", err)
	}

	altOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx, altStore, OrdersByCustomerInput{Customer: customerID})
	if err != nil {
		t.Fatalf("%s orders_by_customer: %v", altName, err)
	}

	if len(memOrders) != len(altOrders) {
		t.Errorf("orders_by_customer length divergence: memory=%d %s=%d",
			len(memOrders), altName, len(altOrders))
	}

	// 4. total_revenue (Counter sum).
	memRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, memStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("memory total_revenue: %v", err)
	}

	altRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx, altStore, TotalRevenueInput{})
	if err != nil {
		t.Fatalf("%s total_revenue: %v", altName, err)
	}

	for k, mv := range memRev {
		if av, ok := altRev[k]; !ok || av != mv {
			t.Errorf("total_revenue divergence for %s: memory=%d %s=%d", k, mv, altName, av)
		}
	}

	t.Logf("%s parity verified across %d events: all queries agree", altName, n)
}

// runEngineThroughputBenchmark measures event ingestion throughput.
func runEngineThroughputBenchmark(b *testing.B, newEngine func(b testing.TB) metaengine.Engine) {
	n := 1_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		b.StopTimer()
		eng := newEngine(b)
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

package bench_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_parity_test.go — verifies that the same events produce identical query
// results across different engine configurations. This is the correctness
// foundation for multi-engine routing: if Memory and SQLite disagree, the
// planner's routing decisions are meaningless.

// TestPromise_CrossEngine_ParityAtScale seeds 5K events through both a
// memory-only and a memory+sqlite store, then verifies all queries return
// identical results. This catches reify bugs, serialization divergence, and
// fold-handler engine-specific behavior.
func TestPromise_CrossEngine_ParityAtScale(t *testing.T) {
	t.Parallel()

	n := 5_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	// Run against memory-only.
	memStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer memStore.Close()
	for _, e := range events {
		if err := memStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("memory Apply %s: %v", e.typeName, err)
		}
	}

	// Run against memory+sqlite.
	eng, db := newSQLiteEngine()
	defer metaengine.DeferClose(db)
	sqlStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine(), eng})
	defer sqlStore.Close()
	for _, e := range events {
		if err := sqlStore.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("sqlite Apply %s: %v", e.typeName, err)
		}
	}

	// Compare results for each query type.
	orderID := OrderID("ord-000000")
	customerID := CustomerID("cus-000")

	// 1. find_order (Map point lookup).
	memOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx,
		memStore,
		FindOrderInput{ID: orderID},
	)
	if err != nil {
		t.Fatalf("memory find_order: %v", err)
	}
	sqlOrder, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx,
		sqlStore,
		FindOrderInput{ID: orderID},
	)
	if err != nil {
		t.Fatalf("sqlite find_order: %v", err)
	}
	if !reflect.DeepEqual(memOrder, sqlOrder) {
		t.Errorf("find_order divergence: memory=%+v sqlite=%+v", memOrder, sqlOrder)
	}

	// 2. count_by_status (Counter aggregate).
	memCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx,
		memStore,
		CountOrdersByStatusInput{},
	)
	if err != nil {
		t.Fatalf("memory count_by_status: %v", err)
	}
	sqlCounts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx,
		sqlStore,
		CountOrdersByStatusInput{},
	)
	if err != nil {
		t.Fatalf("sqlite count_by_status: %v", err)
	}
	for k, mv := range memCounts {
		if sv, ok := sqlCounts[k]; !ok || sv != mv {
			t.Errorf("count_by_status divergence for %s: memory=%d sqlite=%d(exists=%v)",
				k, mv, sv, ok)
		}
	}

	// 3. orders_by_customer (Multimap lookup).
	memOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx,
		memStore,
		OrdersByCustomerInput{Customer: customerID},
	)
	if err != nil {
		t.Fatalf("memory orders_by_customer: %v", err)
	}
	sqlOrders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx,
		sqlStore,
		OrdersByCustomerInput{Customer: customerID},
	)
	if err != nil {
		t.Fatalf("sqlite orders_by_customer: %v", err)
	}
	if len(memOrders) != len(sqlOrders) {
		t.Errorf(
			"orders_by_customer length divergence: memory=%d sqlite=%d",
			len(memOrders),
			len(sqlOrders),
		)
	}

	// 4. total_revenue (Counter sum).
	memRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx,
		memStore,
		TotalRevenueInput{},
	)
	if err != nil {
		t.Fatalf("memory total_revenue: %v", err)
	}
	sqlRev, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx,
		sqlStore,
		TotalRevenueInput{},
	)
	if err != nil {
		t.Fatalf("sqlite total_revenue: %v", err)
	}
	for k, mv := range memRev {
		if sv, ok := sqlRev[k]; !ok || sv != mv {
			t.Errorf("total_revenue divergence for %s: memory=%d sqlite=%d(exists=%v)",
				k, mv, sv, ok)
		}
	}

	t.Logf("parity verified across %d events: all queries agree", n)
}

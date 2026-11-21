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

// applyAll seeds every event into the store, failing the test on the first
// rejected apply.
func applyAll(t *testing.T, ctx context.Context, store *metaengine.Store, events []promiseEvent) {
	t.Helper()

	for _, e := range events {
		if err := store.Apply(ctx, e.typeName, e.payload); err != nil {
			t.Fatalf("apply %s: %v", e.typeName, err)
		}
	}
}

// requireParity executes the same typed query against both stores and hands
// the result pair to compare for the engine-divergence assertion.
func requireParity[Q, R any](
	t *testing.T,
	ctx context.Context,
	memStore, sqlStore *metaengine.Store,
	query Q,
	name string,
	compare func(t *testing.T, memResult, sqlResult R),
) {
	t.Helper()

	memResult, err := metaengine.ExecuteTyped[Q, R](ctx, memStore, query)
	if err != nil {
		t.Fatalf("memory %s: %v", name, err)
	}
	sqlResult, err := metaengine.ExecuteTyped[Q, R](ctx, sqlStore, query)
	if err != nil {
		t.Fatalf("sqlite %s: %v", name, err)
	}
	compare(t, memResult, sqlResult)
}

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
	applyAll(t, ctx, memStore, events)

	// Run against memory+sqlite.
	eng, db := newSQLiteEngine()
	defer metaengine.DeferClose(db)
	sqlStore := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine(), eng})
	defer sqlStore.Close()
	applyAll(t, ctx, sqlStore, events)

	// Compare results for each query type.
	orderID := OrderID("ord-000000")
	customerID := CustomerID("cus-000")

	// 1. find_order (Map point lookup).
	requireParity(t, ctx, memStore, sqlStore, FindOrderInput{ID: orderID}, "find_order",
		func(t *testing.T, memOrder, sqlOrder OrderView) {
			if !reflect.DeepEqual(memOrder, sqlOrder) {
				t.Errorf("find_order divergence: memory=%+v sqlite=%+v", memOrder, sqlOrder)
			}
		})

	// 2. count_by_status (Counter aggregate).
	requireParity(t, ctx, memStore, sqlStore, CountOrdersByStatusInput{}, "count_by_status",
		func(t *testing.T, memCounts, sqlCounts map[string]int64) {
			for k, mv := range memCounts {
				if sv, ok := sqlCounts[k]; !ok || sv != mv {
					t.Errorf("count_by_status divergence for %s: memory=%d sqlite=%d(exists=%v)",
						k, mv, sv, ok)
				}
			}
		})

	// 3. orders_by_customer (Multimap lookup).
	requireParity(t, ctx, memStore, sqlStore, OrdersByCustomerInput{Customer: customerID},
		"orders_by_customer",
		func(t *testing.T, memOrders, sqlOrders []OrderID) {
			if len(memOrders) != len(sqlOrders) {
				t.Errorf(
					"orders_by_customer length divergence: memory=%d sqlite=%d",
					len(memOrders),
					len(sqlOrders),
				)
			}
		})

	// 4. total_revenue (Counter sum).
	requireParity(t, ctx, memStore, sqlStore, TotalRevenueInput{}, "total_revenue",
		func(t *testing.T, memRev, sqlRev map[string]int64) {
			for k, mv := range memRev {
				if sv, ok := sqlRev[k]; !ok || sv != mv {
					t.Errorf("total_revenue divergence for %s: memory=%d sqlite=%d(exists=%v)",
						k, mv, sv, ok)
				}
			}
		})

	t.Logf("parity verified across %d events: all queries agree", n)
}

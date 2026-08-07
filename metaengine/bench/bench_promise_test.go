package bench_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── E-COMMERCE DOMAIN (for promise benchmarks) ───
// These types exercise the metaengine's core value: declare events + queries,
// the system routes each query to the optimal engine, events fan out to all
// projections, and every query returns correct results.

type (
	OrderID    string
	CustomerID string
)

// Event types.
type OrderCreated struct {
	ID          OrderID
	CustomerID  CustomerID
	Status      string
	AmountCents int64
	ProductID   string
	CreatedAt   time.Time
}

type ItemAdded struct {
	OrderID    OrderID
	ProductID  string
	PriceCents int64
	At         time.Time
}

type OrderShipped struct {
	ID       OrderID
	Tracking string
	At       time.Time
}

type OrderCancelled struct {
	ID OrderID
	At time.Time
}

type OrderPaid struct {
	ID          OrderID
	AmountCents int64
	At          time.Time
}

// ─── Query 1: find_order (Map ADT — point lookup) ───

type FindOrderInput struct {
	ID OrderID
}

type OrderView struct {
	ID         OrderID
	CustomerID CustomerID
	Status     string
	TotalCents int64
	Items      []string
}

func findOrderQuery() metaengine.QueryDecl[FindOrderInput, OrderView] {
	return metaengine.Query[FindOrderInput, OrderView](
		"find_order",
		metaengine.On(OrderCreated{}, func(e OrderCreated) (OrderID, OrderView) {
			return e.ID, OrderView{
				ID: e.ID, CustomerID: e.CustomerID, Status: e.Status,
				TotalCents: e.AmountCents, Items: []string{e.ProductID},
			}
		}),
		metaengine.On(ItemAdded{}, func(e ItemAdded, prev OrderView) OrderView {
			prev.Items = append(prev.Items, e.ProductID)
			prev.TotalCents += e.PriceCents
			return prev
		}),
		metaengine.On(OrderShipped{}, func(e OrderShipped, prev OrderView) OrderView {
			prev.Status = "shipped"
			return prev
		}),
		metaengine.On(OrderCancelled{}, func(e OrderCancelled, prev OrderView) OrderView {
			prev.Status = "cancelled"
			return prev
		}),
	)
}

// ─── Query 2: list_orders_by_status (FilteredMap — filtered scan) ───

type ListOrdersByStatusInput struct {
	Status string
	Limit  int
}

func listOrdersByStatusQuery() metaengine.QueryDecl[ListOrdersByStatusInput, OrderView] {
	return metaengine.Query[ListOrdersByStatusInput, OrderView](
		"list_orders_by_status",
		metaengine.On(OrderCreated{}, func(e OrderCreated) (OrderID, OrderView) {
			return e.ID, OrderView{
				ID: e.ID, CustomerID: e.CustomerID, Status: e.Status,
				TotalCents: e.AmountCents, Items: []string{e.ProductID},
			}
		}),
		metaengine.On(OrderShipped{}, func(e OrderShipped, prev OrderView) OrderView {
			prev.Status = "shipped"
			return prev
		}),
		metaengine.On(OrderCancelled{}, func(e OrderCancelled, prev OrderView) OrderView {
			prev.Status = "cancelled"
			return prev
		}),
		metaengine.FilterOnField[OrderView]("Status", metaengine.FilterEq),
		metaengine.SortOnField[OrderView]("TotalCents", true),
	)
}

// ─── Query 3: count_by_status (Counter ADT — aggregate) ───

type CountOrdersByStatusInput struct{}

func countOrdersByStatusQuery() metaengine.QueryDecl[CountOrdersByStatusInput, map[string]int64] {
	return metaengine.Query[CountOrdersByStatusInput, map[string]int64](
		"count_orders_by_status",
		metaengine.On(OrderCreated{}, func(e OrderCreated) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
		metaengine.On(OrderShipped{}, func(e OrderShipped) metaengine.Delta {
			return metaengine.Delta{"pending": -1, "shipped": +1}
		}),
		metaengine.On(OrderCancelled{}, func(e OrderCancelled) metaengine.Delta {
			return metaengine.Delta{"pending": -1, "cancelled": +1}
		}),
	)
}

// ─── Query 4: orders_by_customer (Multimap ADT — secondary index) ───

type OrdersByCustomerInput struct {
	Customer CustomerID
}

func ordersByCustomerQuery() metaengine.QueryDecl[OrdersByCustomerInput, []OrderID] {
	return metaengine.Query[OrdersByCustomerInput, []OrderID](
		"orders_by_customer",
		metaengine.On(OrderCreated{}, func(e OrderCreated) metaengine.MultiEntry {
			return metaengine.MultiEntry{Key: e.CustomerID, Value: e.ID}
		}),
	)
}

// ─── Query 5: recent_orders (Log ADT — time-ordered) ───

type RecentOrdersInput struct {
	Limit int
}

func recentOrdersQuery() metaengine.QueryDecl[RecentOrdersInput, []OrderID] {
	return metaengine.Query[RecentOrdersInput, []OrderID](
		"recent_orders",
		metaengine.On(OrderCreated{}, func(e OrderCreated) metaengine.Append {
			return metaengine.Append{Value: e.ID}
		}),
	)
}

// ─── Query 6: total_revenue (Counter ADT — sum) ───

type TotalRevenueInput struct{}

func totalRevenueQuery() metaengine.QueryDecl[TotalRevenueInput, map[string]int64] {
	return metaengine.Query[TotalRevenueInput, map[string]int64](
		"total_revenue",
		metaengine.On(OrderPaid{}, func(e OrderPaid) metaengine.Delta {
			return metaengine.Delta{"revenue_cents": e.AmountCents}
		}),
	)
}

// allPromiseQueries returns all 6 query declarations spanning 4 ADTs:
// Map (find_order, list_by_status), Counter (count_by_status, total_revenue),
// Multimap (orders_by_customer), Log (recent_orders).
func allPromiseQueries() []any {
	return []any{
		findOrderQuery(),
		listOrdersByStatusQuery(),
		countOrdersByStatusQuery(),
		ordersByCustomerQuery(),
		recentOrdersQuery(),
		totalRevenueQuery(),
	}
}

// ─── Event Generator ───

// promiseEvent represents a generated event with its type name and payload.
type promiseEvent struct {
	typeName string
	payload  any
}

// generatePromiseEvents produces a realistic mixed event stream:
// 60% OrderCreated, 20% ItemAdded, 10% OrderShipped, 5% OrderCancelled, 5% OrderPaid.
// Uses deterministic IDs for reproducibility.
func generatePromiseEvents(n int) []promiseEvent {
	events := make([]promiseEvent, 0, n)
	base := time.Now()

	for i := range n {
		orderID := OrderID(fmt.Sprintf("ord-%06d", i))
		customerID := CustomerID(fmt.Sprintf("cus-%06d", i%100)) // 100 unique customers
		pct := i % 20

		switch {
		case pct < 12: // 60% OrderCreated
			status := "pending"
			events = append(events, promiseEvent{
				"OrderCreated", OrderCreated{
					ID: orderID, CustomerID: customerID, Status: status,
					AmountCents: int64(1000 + (i%10)*500),
					ProductID:   fmt.Sprintf("prod-%04d", i%200),
					CreatedAt:   base.Add(time.Duration(i) * time.Millisecond),
				},
			})
		case pct < 16: // 20% ItemAdded
			events = append(events, promiseEvent{
				"ItemAdded", ItemAdded{
					OrderID:    OrderID(fmt.Sprintf("ord-%06d", i%100)),
					ProductID:  fmt.Sprintf("prod-%04d", i%200),
					PriceCents: int64(500 + (i%5)*200),
					At:         base.Add(time.Duration(i) * time.Millisecond),
				},
			})
		case pct < 18: // 10% OrderShipped
			events = append(events, promiseEvent{
				"OrderShipped", OrderShipped{
					ID:       OrderID(fmt.Sprintf("ord-%06d", i%100)),
					Tracking: fmt.Sprintf("TRK%d", i),
					At:       base.Add(time.Duration(i) * time.Millisecond),
				},
			})
		case pct < 19: // 5% OrderCancelled
			events = append(events, promiseEvent{
				"OrderCancelled", OrderCancelled{
					ID: OrderID(fmt.Sprintf("ord-%06d", i%100)),
					At: base.Add(time.Duration(i) * time.Millisecond),
				},
			})
		default: // 5% OrderPaid
			events = append(events, promiseEvent{
				"OrderPaid", OrderPaid{
					ID:          OrderID(fmt.Sprintf("ord-%06d", i%100)),
					AmountCents: int64(1000 + (i%10)*500),
					At:          base.Add(time.Duration(i) * time.Millisecond),
				},
			})
		}
	}

	return events
}

// seedPromiseStore applies n events to the store and returns the count actually applied.
func seedPromiseStore(tb testing.TB, store *metaengine.Store, n int) {
	tb.Helper()
	ctx := context.Background()
	events := generatePromiseEvents(n)

	for _, e := range events {
		if err := store.Apply(ctx, e.typeName, e.payload); err != nil {
			tb.Fatalf("seed Apply %s: %v", e.typeName, err)
		}
	}
}

// ─── Multi-Engine Factory ───

// promiseEnginePool describes a named engine pool configuration.
type promiseEnginePool struct {
	name    string
	engines func() []metaengine.Engine
	cleanup func()
}

func memoryOnlyPool() promiseEnginePool {
	return promiseEnginePool{
		name: "memory-only",
		engines: func() []metaengine.Engine {
			return []metaengine.Engine{metaengine.NewMemoryEngine()}
		},
		cleanup: func() {},
	}
}

func memorySQLitePool() promiseEnginePool {
	var dbs []func()

	return promiseEnginePool{
		name: "memory+sqlite",
		engines: func() []metaengine.Engine {
			eng, db := newSQLiteEngine()
			dbs = append(dbs, func() { _ = db.Close() })
			return []metaengine.Engine{metaengine.NewMemoryEngine(), eng}
		},
		cleanup: func() {
			for _, c := range dbs {
				c()
			}
		},
	}
}

// promiseEnginePools returns all available engine pool configurations for
// comparison benchmarks. Each pool creates fresh engines on each call.
func promiseEnginePools() []promiseEnginePool {
	return []promiseEnginePool{
		memoryOnlyPool(),
		memorySQLitePool(),
	}
}

// planPromiseStore creates a Store from the given engines with all 6 promise queries.
func planPromiseStore(tb testing.TB, engines []metaengine.Engine) *metaengine.Store {
	tb.Helper()
	store, err := metaengine.Plan(engines, allPromiseQueries()...)
	if err != nil {
		tb.Fatalf("Plan: %v", err)
	}
	return store
}

// ─── Domain Sanity Test ───

// TestPromise_DomainModel verifies the 6-query domain works correctly on the
// memory engine. This catches any declaration bugs before benchmarks run.
func TestPromise_DomainModel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer store.Close()

	// Apply a few events.
	mustApply(t, ctx, store, "OrderCreated", OrderCreated{
		ID: "ord-001", CustomerID: "cus-001", Status: "pending",
		AmountCents: 5000, ProductID: "prod-001", CreatedAt: time.Now(),
	})
	mustApply(t, ctx, store, "OrderCreated", OrderCreated{
		ID: "ord-002", CustomerID: "cus-001", Status: "pending",
		AmountCents: 3000, ProductID: "prod-002", CreatedAt: time.Now(),
	})
	mustApply(
		t,
		ctx,
		store,
		"OrderPaid",
		OrderPaid{ID: "ord-001", AmountCents: 5000, At: time.Now()},
	)

	// Query 1: find_order (Map point lookup).
	order, err := metaengine.ExecuteTyped[FindOrderInput, OrderView](
		ctx,
		store,
		FindOrderInput{ID: "ord-001"},
	)
	if err != nil {
		t.Fatalf("find_order: %v", err)
	}
	if order.Status != "pending" || order.TotalCents != 5000 {
		t.Fatalf("find_order result mismatch: %+v", order)
	}

	// Query 3: count_by_status (Counter aggregate).
	counts, err := metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
		ctx,
		store,
		CountOrdersByStatusInput{},
	)
	if err != nil {
		t.Fatalf("count_by_status: %v", err)
	}
	if counts["pending"] != 2 {
		t.Fatalf("expected 2 pending, got %d", counts["pending"])
	}

	// Query 4: orders_by_customer (Multimap).
	orders, err := metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
		ctx,
		store,
		OrdersByCustomerInput{Customer: "cus-001"},
	)
	if err != nil {
		t.Fatalf("orders_by_customer: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders for cus-001, got %d", len(orders))
	}

	// Query 5: recent_orders (Log tail).
	recent, err := metaengine.ExecuteTyped[RecentOrdersInput, []OrderID](
		ctx,
		store,
		RecentOrdersInput{Limit: 10},
	)
	if err != nil {
		t.Fatalf("recent_orders: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent orders, got %d", len(recent))
	}

	// Query 6: total_revenue (Counter sum).
	revenue, err := metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
		ctx,
		store,
		TotalRevenueInput{},
	)
	if err != nil {
		t.Fatalf("total_revenue: %v", err)
	}
	if revenue["revenue_cents"] != 5000 {
		t.Fatalf("expected 5000 revenue, got %d", revenue["revenue_cents"])
	}
}

func mustApply(
	t *testing.T,
	ctx context.Context,
	store *metaengine.Store,
	eventType string,
	payload any,
) {
	t.Helper()
	if err := store.Apply(ctx, eventType, payload); err != nil {
		t.Fatalf("Apply %s: %v", eventType, err)
	}
}

// ─── Micro-benchmark: Apply throughput at scale ───

// BenchmarkPromise_ApplyThroughput measures raw event ingestion throughput
// as N increases. This reveals whether the fan-out cost scales linearly.
func BenchmarkPromise_ApplyThroughput(b *testing.B) {
	for _, n := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("events=%d", n), func(b *testing.B) {
			events := generatePromiseEvents(n)
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
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

			b.ReportMetric(float64(n)/b.Elapsed().Seconds()*float64(b.N), "events/sec")
		})
	}
}

// BenchmarkPromise_ConcurrentApply measures concurrent Apply throughput with
// 8 goroutines pushing events into a single store. Reveals contention behavior.
func BenchmarkPromise_ConcurrentApply(b *testing.B) {
	concurrency := 8
	n := 1_000
	events := generatePromiseEvents(n)

	b.ResetTimer()

	for range b.N {
		store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
		ctx := context.Background()
		var errCount atomic.Int64

		var wg sync.WaitGroup
		wg.Add(concurrency)

		for w := range concurrency {
			go func(workerID int) {
				defer wg.Done()
				start := (n / concurrency) * workerID
				end := start + n/concurrency
				if end > n {
					end = n
				}
				for i := start; i < end; i++ {
					if err := store.Apply(ctx, events[i].typeName, events[i].payload); err != nil {
						errCount.Add(1)
						return
					}
				}
			}(w)
		}
		wg.Wait()
		store.Close()

		if errCount.Load() > 0 {
			b.Fatalf("%d workers failed", errCount.Load())
		}
	}

	b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

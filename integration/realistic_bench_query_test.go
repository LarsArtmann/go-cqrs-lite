//go:build scale

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3/querytest"
)

// ---------------------------------------------------------------------------
// 7. Query dispatch — 1K queries with paginated results
// ---------------------------------------------------------------------------

func BenchmarkRealistic_QueryDispatch(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	items := make([]OrderCreated, 1000)
	for i := range items {
		items[i] = OrderCreated{
			OrderID:   fmt.Sprintf("ORD-%04d", i),
			Customer:  fmt.Sprintf("customer-%d", i),
			Total:     float64(i) * 10.0,
			Items:     i % 20,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	if err := dispatcher.Register(
		"list.orders",
		func(_ context.Context, _ query.Query) (any, error) {
			return query.NewPaginatedResult(
				items[:50],
				uint(len(items)),
				query.NewPagination(1, 50),
			), nil
		},
	); err != nil {
		b.Fatalf("register: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for range 1000 {
			q := querytest.New(t, "list.orders")
			if _, err := dispatcher.Dispatch(ctx, q); err != nil {
				b.Fatalf("Dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*1000)/b.Elapsed().Seconds(), "queries/sec")
}

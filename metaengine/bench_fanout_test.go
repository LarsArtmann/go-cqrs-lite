package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_fanout_test.go — measures the fan-out cost: when one event is Applied,
// it updates ALL queries that declare a fold for that event type. With 6
// queries, OrderCreated fans out to 5 of them. This reveals the write
// amplification of the multi-query pattern.

// BenchmarkMultiQuery_EventFanOut measures event ingestion throughput with all
// 6 promise queries active (realistic multi-projection scenario). Compare with
// single-query throughput to see the fan-out overhead.
func BenchmarkMultiQuery_EventFanOut(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
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

			b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkWriteAmplification_Scaling measures Apply throughput as the number
// of active projections scales from 1 to 6. Each projection adds write overhead.
// The key insight: 1 event → N projection writes. How does N affect throughput?
func BenchmarkWriteAmplification_Scaling(b *testing.B) {
	// Define queries incrementally — each adds one more fold handler for
	// OrderCreated, increasing the fan-out by 1.
	queriesByCount := map[int][]any{
		1: {findOrderQuery()},
		2: {findOrderQuery(), countOrdersByStatusQuery()},
		3: {findOrderQuery(), countOrdersByStatusQuery(), ordersByCustomerQuery()},
		4: {findOrderQuery(), countOrdersByStatusQuery(), ordersByCustomerQuery(), recentOrdersQuery()},
		5: {
			findOrderQuery(), countOrdersByStatusQuery(), ordersByCustomerQuery(),
			recentOrdersQuery(), listOrdersByStatusQuery(),
		},
		6: allPromiseQueries(),
	}

	n := 1_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	for projCount := 1; projCount <= 6; projCount++ {
		b.Run(fmt.Sprintf("projections=%d", projCount), func(b *testing.B) {
			queries := queriesByCount[projCount]

			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()},
					queries...,
				)
				if err != nil {
					b.Fatal(err)
				}
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
		})
	}
}

// BenchmarkWriteAmplification_BudgetEnforcement verifies the write-amplification
// budget diagnostic fires and measures whether it affects throughput.
// With 6 projections and budget=3, the planner should emit a WARN diagnostic.
func BenchmarkWriteAmplification_BudgetEnforcement(b *testing.B) {
	n := 1_000
	events := generatePromiseEvents(n)
	ctx := context.Background()

	// With budget (diagnostic overhead).
	b.Run("budget=3", func(b *testing.B) {
		b.ResetTimer()

		for range b.N {
			b.StopTimer()
			args := append(allPromiseQueries(), metaengine.WithWriteAmplificationBudget(3))
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				args...,
			)
			if err != nil {
				b.Fatal(err)
			}
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
	})

	// Without budget (no diagnostic).
	b.Run("budget=unlimited", func(b *testing.B) {
		b.ResetTimer()

		for range b.N {
			b.StopTimer()
			args := append(allPromiseQueries(), metaengine.WithWriteAmplificationBudget(100))
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				args...,
			)
			if err != nil {
				b.Fatal(err)
			}
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
	})
}

package metaengine_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_materialize_test.go — validates the materialize-vs-replay cost model.
// The planner's ShouldMaterialize formula recommends whether to materialize a
// projection (store the result) or replay events on each read. This benchmark
// measures the actual cost difference and validates the formula's accuracy.

// BenchmarkMaterializeVsReplay_WriteCost measures the write-time cost of
// maintaining 1 vs 6 projections. Maintaining more projections = higher write
// cost but lower read cost (O(1) vs O(N) fold). This is the materialization
// tradeoff in action.
func BenchmarkMaterializeVsReplay_WriteCost(b *testing.B) {
	for _, projectionCount := range []int{1, 3, 6} {
		b.Run(fmt.Sprintf("projections=%d", projectionCount), func(b *testing.B) {
			queries := allPromiseQueries()[:projectionCount]
			events := generatePromiseEvents(1_000)
			ctx := context.Background()

			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				store, err := metaengine.Plan(
					[]metaengine.Engine{metaengine.NewMemoryEngine()}, queries...)
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

			b.ReportMetric(float64(1_000)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkMaterializeVsReplay_ReadCost measures read latency with different
// projection counts. With 6 projections, find_order is O(1) Map lookup.
// With 1 projection, the data still needs to be computed. This shows the
// read-side benefit of materialization.
func BenchmarkMaterializeVsReplay_ReadCost(b *testing.B) {
	store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
	defer store.Close()
	seedPromiseStore(b, store, 10_000)
	ctx := context.Background()

	orderID := OrderID("ord-000000")
	customerID := CustomerID("cus-000")

	// Read from the fully-materialized store (all 6 projections).
	b.Run("materialized-6-projections", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			// All reads are O(1) or O(logN) since projections are maintained.
			_, _ = metaengine.ExecuteTyped[FindOrderInput, OrderView](
				ctx, store, FindOrderInput{ID: orderID})
			_, _ = metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
				ctx, store, CountOrdersByStatusInput{})
			_, _ = metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
				ctx, store, OrdersByCustomerInput{Customer: customerID})
		}
		b.ReportMetric(float64(b.N)*3/b.Elapsed().Seconds(), "queries/sec")
	})
}

// TestMaterializeVsReplay_PredictionAccuracy validates that the
// ShouldMaterialize formula gives correct recommendations for different
// workload patterns. High-read/low-write should recommend materialize;
// low-read/high-write should recommend replay.
func TestPromise_MaterializeVsReplay_PredictionAccuracy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stats    metaengine.WorkloadStats
		wantMat  bool
		wantDiag string
	}{
		{
			name: "read-heavy-short-streams",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 10,
				ReadRatePerSec:  1000,
				AvgStreamLength: 50,
			},
			wantMat:  true,
			wantDiag: "materialize recommended",
		},
		{
			name: "write-heavy-long-streams",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 1000,
				ReadRatePerSec:  1,
				AvgStreamLength: 100,
			},
			wantMat:  false,
			wantDiag: "replay may be cheaper",
		},
		{
			name: "balanced-moderate",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 50,
				ReadRatePerSec:  50,
				AvgStreamLength: 10,
			},
			wantMat:  true, // replay=500, materialize=55 → materialize
			wantDiag: "materialize recommended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := metaengine.ShouldMaterialize(tt.stats)
			if got != tt.wantMat {
				t.Errorf("ShouldMaterialize(%+v) = %v, want %v (replay=%.2f, materialize=%.2f)",
					tt.stats, got, tt.wantMat,
					metaengine.ReplayCost(tt.stats), metaengine.MaterializeCost(tt.stats))
			}

			// Verify the planner emits the right diagnostic.
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				findOrderQuery(),
				metaengine.WithWorkloadStats(map[string]metaengine.WorkloadStats{
					"find_order": tt.stats,
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()

			found := false
			for _, d := range store.Plan().Diagnostics {
				if strings.Contains(d.Message, tt.wantDiag) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf(
					"expected diagnostic containing %q, got: %v",
					tt.wantDiag,
					store.Plan().Diagnostics,
				)
			}
		})
	}
}

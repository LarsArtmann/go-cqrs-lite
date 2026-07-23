//go:build scale

package integration_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// ---------------------------------------------------------------------------
// 5. Aggregate Listing — 10K aggregates, cursor-paginated
// ---------------------------------------------------------------------------

func BenchmarkRealistic_Listing(b *testing.B) {
	b.ReportAllocs()

	aggCount := 10_000

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	_ = seedOrders(b, store, aggCount, 3)

	reader := listing.NewInMemoryStreamReader(store)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		var after id.StreamID
		items := 0

		for {
			page, err := listing.NewListBuilder(reader).
				OfType("Order").
				PageSize(100).
				After(after).
				List(ctx)
			if err != nil {
				b.Fatalf("List: %v", err)
			}

			items += len(page.Items)

			if !page.HasMore || len(page.Items) == 0 {
				break
			}
			after = page.Items[len(page.Items)-1].ID
		}

		if items != aggCount {
			b.Fatalf("expected %d items, got %d", aggCount, items)
		}
	}

	b.ReportMetric(float64(aggCount), "aggregates")
	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "items-iterated/sec")
}

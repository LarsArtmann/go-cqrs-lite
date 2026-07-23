package integration_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// ---------------------------------------------------------------------------
// 6. Thousand Materialized Views — listing aggregates at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_Listing_10KAggregates(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	aggCount := 10_000

	aggIDs := make([]id.StreamID, aggCount)
	for i := range aggCount {
		aggIDs[i] = id.NewAggregateID()
		payload, err := json.Marshal(map[string]string{"name": fmt.Sprintf("item-%d", i)})
		if err != nil {
			b.Fatalf("json.Marshal: %v", err)
		}

		evt, err := event.NewEvent("ItemCreated", aggIDs[i], "Item", 1, payload)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}

		err = store.AppendBatch(
			ctx,
			id.NewAggregateRef("Item", aggIDs[i]),
			[]event.Event{evt},
		)
		if err != nil {
			b.Fatalf("AppendBatch: %v", err)
		}
	}

	reader := listing.NewInMemoryStreamReader(store)

	b.ResetTimer()

	for b.Loop() {
		_, err := listing.NewListBuilder(reader).
			OfType("Item").
			PageSize(100).
			List(ctx)
		if err != nil {
			b.Fatalf("List: %v", err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "list-ops/sec")
}

func BenchmarkScale_Listing_PaginateThrough10K(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	aggCount := 10_000

	for i := range aggCount {
		aggID := id.NewAggregateID()
		payload, err := json.Marshal(map[string]string{"name": fmt.Sprintf("item-%d", i)})
		if err != nil {
			b.Fatalf("json.Marshal: %v", err)
		}

		evt, err := event.NewEvent("ItemCreated", aggID, "Item", 1, payload)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
		_ = store.AppendBatch(ctx, id.NewAggregateRef("Item", aggID), []event.Event{evt})
	}

	reader := listing.NewInMemoryStreamReader(store)

	b.ResetTimer()

	for b.Loop() {
		var after id.StreamID

		for {
			page, err := listing.NewListBuilder(reader).
				OfType("Item").
				PageSize(50).
				After(after).
				List(ctx)
			if err != nil {
				b.Fatalf("List: %v", err)
			}

			if !page.HasMore || len(page.Items) == 0 {
				break
			}

			after = page.Items[len(page.Items)-1].ID
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "items-iterated/sec")
}

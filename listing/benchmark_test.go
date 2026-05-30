package listing_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/listing"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func seedBenchAggregates(
	b *testing.B,
	aggType string,
	evtType string,
	payloadKey string,
	payloadVal string,
	n int,
) *listing.InMemoryAggregateReader {
	b.Helper()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range n {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal( //nolint:errchkjson
			map[string]string{payloadKey: payloadVal},
		)
		evt, _ := event.NewEvent(
			event.Type(evtType),
			aggID,
			event.AggregateType(aggType),
			1,
			payload,
		)
		_ = store.AppendBatch(
			ctx,
			event.NewAggregateRef(event.AggregateType(aggType), aggID),
			[]event.Event{evt},
		)
	}

	return listing.NewInMemoryAggregateReader(store)
}

func BenchmarkInMemoryList(b *testing.B) {
	tests := []struct {
		name     string
		aggType  event.AggregateType
		evtType  event.Type
		key      string
		val      string
		count    int
		pageSize uint
	}{
		{"1000Aggregates", "User", "UserCreated", "name", "user", 1000, 50},
		{"100Aggregates", "Order", "OrderCreated", "item", "widget", 100, 10},
		{"SmallPages", "Cart", "ItemAdded", "sku", "ABC", 500, 5},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			reader := seedBenchAggregates(
				b,
				string(tc.aggType),
				string(tc.evtType),
				tc.key,
				tc.val,
				tc.count,
			)
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				_, err := listing.NewListBuilder(reader).
					OfType(tc.aggType).
					PageSize(tc.pageSize).
					List(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkInMemoryList_TombstoneFilter(b *testing.B) {
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 500 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal( //nolint:errchkjson
			map[string]string{"name": "doc"},
		)
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			event.NewAggregateRef(event.AggregateType("Doc"), aggID),
			[]event.Event{evt},
		)
	}

	for range 200 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal( //nolint:errchkjson
			map[string]string{"name": "deleted"},
		)
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			event.NewAggregateRef(event.AggregateType("Doc"), aggID),
			[]event.Event{evt},
		)
		marked, _ := event.MarkTombstone(evt)
		_ = store.AppendBatch(
			ctx,
			event.NewAggregateRef(event.AggregateType("Doc"), aggID),
			[]event.Event{marked},
		)
	}

	reader := listing.NewInMemoryAggregateReader(store)

	b.ResetTimer()

	for b.Loop() {
		_, err := listing.NewListBuilder(reader).
			OfType("Doc").
			PageSize(50).
			List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

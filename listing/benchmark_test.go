package listing_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func seedBenchAggregates(
	b *testing.B,
	aggType string,
	evtType string,
	payloadKey string,
	payloadVal string,
	n int,
) *listing.InMemoryStreamReader {
	b.Helper()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range n {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(
			map[string]string{payloadKey: payloadVal},
		)
		evt, _ := event.NewEvent(
			event.Type(evtType),
			aggID,
			id.StreamType(aggType),
			1,
			payload,
		)
		_ = store.AppendBatch(
			ctx,
			id.NewAggregateRef(id.StreamType(aggType), aggID),
			[]event.Event{evt},
		)
	}

	return listing.NewInMemoryStreamReader(store)
}

func BenchmarkInMemoryList(b *testing.B) {
	b.ReportAllocs()
	tests := []struct {
		name     string
		aggType  id.StreamType
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
	b.ReportAllocs()
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 500 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(
			map[string]string{"name": "doc"},
		)
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			id.NewAggregateRef(id.StreamType("Doc"), aggID),
			[]event.Event{evt},
		)
	}

	for range 200 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(
			map[string]string{"name": "deleted"},
		)
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			id.NewAggregateRef(id.StreamType("Doc"), aggID),
			[]event.Event{evt},
		)
		marked, _ := event.MarkTombstone(evt)
		_ = store.AppendBatch(
			ctx,
			id.NewAggregateRef(id.StreamType("Doc"), aggID),
			[]event.Event{marked},
		)
	}

	reader := listing.NewInMemoryStreamReader(store)

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

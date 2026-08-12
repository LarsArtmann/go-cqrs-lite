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

func seedBenchStreams(
	b *testing.B,
	streamType string,
	evtType string,
	payloadKey string,
	payloadVal string,
	n int,
) *listing.InMemoryStreamReader {
	b.Helper()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range n {
		streamID := id.NewStreamID()
		payload, _ := json.Marshal(
			map[string]string{payloadKey: payloadVal},
		)
		evt, _ := event.NewEvent(
			event.Type(evtType),
			streamID,
			id.StreamType(streamType),
			1,
			payload,
		)
		_ = store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType(streamType), streamID),
			[]event.Event{evt},
		)
	}

	return listing.NewInMemoryStreamReader(store)
}

func BenchmarkInMemoryList(b *testing.B) {
	b.ReportAllocs()
	tests := []struct {
		name       string
		streamType id.StreamType
		evtType    event.Type
		key        string
		val        string
		count      int
		pageSize   uint
	}{
		{"1000Streams", "User", "UserCreated", "name", "user", 1000, 50},
		{"100Streams", "Order", "OrderCreated", "item", "widget", 100, 10},
		{"SmallPages", "Cart", "ItemAdded", "sku", "ABC", 500, 5},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			reader := seedBenchStreams(
				b,
				string(tc.streamType),
				string(tc.evtType),
				tc.key,
				tc.val,
				tc.count,
			)
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				_, err := listing.NewListBuilder(reader).
					OfType(tc.streamType).
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
		streamID := id.NewStreamID()
		payload, _ := json.Marshal(
			map[string]string{"name": "doc"},
		)
		evt, _ := event.NewEvent("DocCreated", streamID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Doc"), streamID),
			[]event.Event{evt},
		)
	}

	for range 200 {
		streamID := id.NewStreamID()
		payload, _ := json.Marshal(
			map[string]string{"name": "deleted"},
		)
		evt, _ := event.NewEvent("DocCreated", streamID, "Doc", 1, payload)
		_ = store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Doc"), streamID),
			[]event.Event{evt},
		)
		marked, _ := event.MarkTombstone(evt)
		_ = store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Doc"), streamID),
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

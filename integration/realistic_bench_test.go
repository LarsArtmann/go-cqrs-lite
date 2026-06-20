//go:build scale

package integration_test

// Realistic large-scale benchmarks for go-cqrs-lite.
//
// Run with:
//
//	go test ./integration/... -tags=scale -bench=BenchmarkRealistic -benchmem -run=^$ -benchtime=1x -timeout=10m
//
// Each benchmark runs exactly once (benchtime=1x). They exercise the full
// CQRS pipeline with realistic JSON payloads, projection replay, signing,
// snapshots, concurrent writes, and aggregate listing.
//
// Gated by the "scale" build tag — excluded from normal builds and CI.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func mustEveryN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		panic(err)
	}

	return s
}

func benchNewOrderRepo(
	b *testing.B,
	store *memory.MemoryStore,
	bus *eventtest.FakeBus,
	snapEvery int,
) *decider.Repository[OrderState] {
	b.Helper()

	memSnap := memory.NewMemorySnapshotStore()
	b.Cleanup(func() { _ = memSnap.Close() })

	d := decider.Decider[OrderState]{Initial: OrderState{}, Fold: foldOrder}
	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[OrderState](memSnap),
		decider.WithSnapshotStrategy[OrderState](mustEveryN(snapEvery)),
		decider.WithCodec[OrderState](codec.JSONCodec{}),
	)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	return repo
}

// ---------------------------------------------------------------------------
// Domain types — realistic e-commerce order model
// ---------------------------------------------------------------------------

type OrderCreated struct {
	OrderID   string  `json:"orderId"`
	Customer  string  `json:"customer"`
	Total     float64 `json:"total"`
	Items     int     `json:"items"`
	Timestamp string  `json:"timestamp"`
}

type ItemAdded struct {
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type OrderShipped struct {
	Carrier    string `json:"carrier"`
	TrackingID string `json:"trackingId"`
	ShippedAt  string `json:"shippedAt"`
}

type OrderCancelled struct {
	Reason string `json:"reason"`
}

type OrderState struct {
	Total     float64
	Items     int
	Shipped   bool
	Cancelled bool
}

func foldOrder(_ OrderState, evt event.Event) (OrderState, error) {
	switch evt.Type() {
	case "OrderCreated":
		var p OrderCreated
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return OrderState{}, err
		}
		return OrderState{Total: p.Total, Items: p.Items}, nil
	case "ItemAdded":
		var p ItemAdded
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return OrderState{}, err
		}
		return OrderState{Items: p.Quantity}, nil
	case "OrderShipped":
		return OrderState{Shipped: true}, nil
	case "OrderCancelled":
		return OrderState{Cancelled: true}, nil
	}

	return OrderState{}, nil
}

// ---------------------------------------------------------------------------
// Shared infrastructure
// ---------------------------------------------------------------------------

func newRealisticEvent(
	tb testing.TB,
	eventType string,
	aggID id.AggregateID,
	v event.Version,
	payload any,
) event.Event {
	tb.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		tb.Fatalf("marshal payload: %v", err)
	}

	evt, err := event.New(event.Type(eventType), aggID, "Order", event.Version(v), data)
	if err != nil {
		tb.Fatalf("New: %v", err)
	}

	return evt
}

func seedOrders(
	b *testing.B,
	store *memory.MemoryStore,
	aggCount, eventsPerAgg int,
) []id.AggregateID {
	b.Helper()

	ctx := context.Background()
	aggIDs := make([]id.AggregateID, aggCount)

	for i := range aggCount {
		aggIDs[i] = id.NewAggregateID()
		ref := event.NewAggregateRef("Order", aggIDs[i])
		events := make([]event.Event, eventsPerAgg)

		for v := range eventsPerAgg {
			switch v % 4 {
			case 0:
				events[v] = newRealisticEvent(
					b,
					"OrderCreated",
					aggIDs[i],
					event.Version(v+1),
					OrderCreated{
						OrderID:   aggIDs[i].String(),
						Customer:  fmt.Sprintf("customer-%d", i),
						Total:     99.99,
						Items:     3,
						Timestamp: time.Now().Format(time.RFC3339),
					},
				)
			case 1:
				events[v] = newRealisticEvent(
					b,
					"ItemAdded",
					aggIDs[i],
					event.Version(v+1),
					ItemAdded{
						SKU:      fmt.Sprintf("SKU-%04d", i),
						Name:     fmt.Sprintf("Widget %d", i),
						Price:    19.99,
						Quantity: 2,
					},
				)
			case 2:
				events[v] = newRealisticEvent(
					b,
					"OrderShipped",
					aggIDs[i],
					event.Version(v+1),
					OrderShipped{
						Carrier:    "FedEx",
						TrackingID: fmt.Sprintf("TRACK-%d-%d", i, v),
						ShippedAt:  time.Now().Format(time.RFC3339),
					},
				)
			case 3:
				events[v] = newRealisticEvent(
					b,
					"ItemAdded",
					aggIDs[i],
					event.Version(v+1),
					ItemAdded{
						SKU:      fmt.Sprintf("SKU-%04d-extra", i),
						Name:     fmt.Sprintf("Extra %d", v),
						Price:    5.99,
						Quantity: 1,
					},
				)
			}
		}

		if err := store.AppendBatch(ctx, ref, events); err != nil {
			b.Fatalf("AppendBatch: %v", err)
		}
	}

	return aggIDs
}

// countEvents reports total events in the store via ReadAll.
func countEvents(b *testing.B, store *memory.MemoryStore) int {
	b.Helper()

	all, err := store.ReadAll(context.Background())
	if err != nil {
		b.Fatalf("ReadAll: %v", err)
	}

	return len(all)
}

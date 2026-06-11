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
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
	"github.com/larsartmann/go-cqrs-lite/testutil/v2"
)

func mustEveryN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		panic(err)
	}

	return s
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

// ---------------------------------------------------------------------------
// 1. Full Pipeline — 10K orders × 2 commands = 20K events through full stack
//    command → decider → JSON encode → store → bus → projection (JSON decode)
// ---------------------------------------------------------------------------

func BenchmarkRealistic_FullPipeline(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	aggCount := 10_000

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	d := decider.Decider[OrderState]{Initial: OrderState{}, Fold: foldOrder}
	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	var projected atomic.Int64

	builder := projection.NewBuilder("order-view")
	projection.On(
		builder,
		"OrderCreated",
		jsonCodec,
		func(_ context.Context, _ OrderCreated) error {
			projected.Add(1)
			return nil
		},
	)
	projection.On(builder, "ItemAdded", jsonCodec, func(_ context.Context, _ ItemAdded) error {
		projected.Add(1)
		return nil
	})
	projection.On(
		builder,
		"OrderShipped",
		jsonCodec,
		func(_ context.Context, _ OrderShipped) error {
			projected.Add(1)
			return nil
		},
	)

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	if err := runner.Register(builder.Build()); err != nil {
		b.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(func() { cancel(); _ = runner.Close() })

	go func() { _ = runner.Run(ctx) }()

	cmdDisp := command.NewDispatcher()
	b.Cleanup(func() { _ = cmdDisp.Close() })

	cmdDisp.Register("create.order", func(ctx context.Context, cmd command.Command) error {
		return repo.Execute(ctx, cmd.AggregateID(), "Order",
			func(_ OrderState, v event.Version) ([]event.Event, error) {
				return []event.Event{
					newRealisticEvent(
						b,
						"OrderCreated",
						cmd.AggregateID(),
						v.Increment(),
						OrderCreated{
							OrderID:   cmd.AggregateID().String(),
							Customer:  "test-customer",
							Total:     149.99,
							Items:     5,
							Timestamp: time.Now().Format(time.RFC3339),
						},
					),
				}, nil
			})
	})

	cmdDisp.Register("add.item", func(ctx context.Context, cmd command.Command) error {
		return repo.Execute(ctx, cmd.AggregateID(), "Order",
			func(_ OrderState, v event.Version) ([]event.Event, error) {
				return []event.Event{
					newRealisticEvent(b, "ItemAdded", cmd.AggregateID(), v.Increment(),
						ItemAdded{SKU: "SKU-TEST", Name: "Test Widget", Price: 29.99, Quantity: 1}),
				}, nil
			})
	})

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
	}

	b.ResetTimer()

	for b.Loop() {
		projected.Store(0)

		for _, aggID := range aggIDs {
			cmd := testutil.MustNewCmd("create.order", aggID)
			if err := cmdDisp.Dispatch(context.Background(), cmd); err != nil {
				b.Fatalf("create: %v", err)
			}

			cmd = testutil.MustNewCmd("add.item", aggID)
			if err := cmdDisp.Dispatch(context.Background(), cmd); err != nil {
				b.Fatalf("add: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*aggCount*2)/b.Elapsed().Seconds(), "events/sec")
}

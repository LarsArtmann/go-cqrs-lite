//go:build scale

package integration_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/testutil/v2"
)

// ---------------------------------------------------------------------------
// 1. Full Pipeline — 10K orders × 2 commands = 20K events through full stack
//    command → decider → JSON encode → store → bus → projection (JSON decode)
// ---------------------------------------------------------------------------

func BenchmarkRealistic_FullPipeline(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	aggCount := 10_000

	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
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

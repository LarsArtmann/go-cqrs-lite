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
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func mustEveryN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		panic(err)
	}
	return s
}

func mustNewCmd(commandType command.Type, aggregateID id.AggregateID, opts ...command.Option) *command.BasicCommand {
	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		panic(err)
	}
	return cmd
}

func mustNewQuery(queryType query.Type) *query.BasicQuery {
	q, err := query.New(queryType)
	if err != nil {
		panic(err)
	}
	return q
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

func benchNewOrderRepo(
	b *testing.B,
	store *memory.MemoryStore,
	bus *memory.MemoryBus,
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
			cmd := mustNewCmd("create.order", aggID)
			if err := cmdDisp.Dispatch(context.Background(), cmd); err != nil {
				b.Fatalf("create: %v", err)
			}

			cmd = mustNewCmd("add.item", aggID)
			if err := cmdDisp.Dispatch(context.Background(), cmd); err != nil {
				b.Fatalf("add: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*aggCount*2)/b.Elapsed().Seconds(), "events/sec")
}

// ---------------------------------------------------------------------------
// 2. Projection Replay — 100K pre-stored events replayed through typed handlers
// ---------------------------------------------------------------------------

func BenchmarkRealistic_ProjectionReplay(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	aggCount := 1000
	eventsPerAgg := 100

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	_ = seedOrders(b, store, aggCount, eventsPerAgg)
	totalEvents := countEvents(b, store)

	var processed atomic.Int64

	builder := projection.NewBuilder("replay-view")
	projection.On(
		builder,
		"OrderCreated",
		jsonCodec,
		func(_ context.Context, _ OrderCreated) error {
			processed.Add(1)
			return nil
		},
	)
	projection.On(builder, "ItemAdded", jsonCodec, func(_ context.Context, _ ItemAdded) error {
		processed.Add(1)
		return nil
	})
	projection.On(
		builder,
		"OrderShipped",
		jsonCodec,
		func(_ context.Context, _ OrderShipped) error {
			processed.Add(1)
			return nil
		},
	)

	b.ResetTimer()

	for b.Loop() {
		processed.Store(0)

		cp := memory.NewMemoryCheckpointStore()
		runner, err := projection.NewRunner(store, bus, cp)
		if err != nil {
			b.Fatalf("NewRunner: %v", err)
		}

		if err := runner.Register(builder.Build()); err != nil {
			b.Fatalf("Register: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			_ = runner.Run(ctx)
			close(done)
		}()

		for processed.Load() < int64(totalEvents) {
			runtime.Gosched()
		}

		cancel()
		<-done
		_ = runner.Close()
		_ = cp.Close()
	}

	b.ReportMetric(float64(totalEvents), "events")
	b.ReportMetric(float64(b.N*totalEvents)/b.Elapsed().Seconds(), "events-replayed/sec")
}

// ---------------------------------------------------------------------------
// 3. Concurrent Decider — N CPU goroutines, 100 ops each
// ---------------------------------------------------------------------------

func BenchmarkRealistic_ConcurrentDecider(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	repo := benchNewOrderRepo(b, store, bus, 50)

	ctx := context.Background()
	workers := runtime.NumCPU()
	opsPerWorker := 100

	b.ResetTimer()

	for b.Loop() {
		var wg sync.WaitGroup
		var errCount atomic.Int64

		for w := range workers {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := range opsPerWorker {
					aggID := id.NewAggregateID()
					err := repo.Execute(ctx, aggID, "Order",
						func(_ OrderState, ver event.Version) ([]event.Event, error) {
							return []event.Event{
								newRealisticEvent(
									b,
									"OrderCreated",
									aggID,
									ver.Increment(),
									OrderCreated{
										OrderID:   aggID.String(),
										Customer:  fmt.Sprintf("w%d-op%d", workerID, j),
										Total:     99.99,
										Items:     1,
										Timestamp: time.Now().Format(time.RFC3339),
									},
								),
							}, nil
						})
					if err != nil {
						errCount.Add(1)
					}
				}
			}(w)
		}

		wg.Wait()

		if errCount.Load() > 0 {
			b.Fatalf("%d errors during concurrent execute", errCount.Load())
		}
	}

	totalOps := b.N * workers * opsPerWorker
	b.ReportMetric(float64(workers), "goroutines")
	b.ReportMetric(float64(totalOps)/b.Elapsed().Seconds(), "executes/sec")
}

// ---------------------------------------------------------------------------
// 4. HMAC Signing — sign + verify 10K events
// ---------------------------------------------------------------------------

func BenchmarkRealistic_Signing(b *testing.B) {
	b.ReportAllocs()

	secret := slices.Repeat([]byte("x"), 32)
	signer, err := signing.NewHMAC(secret)
	if err != nil {
		b.Fatalf("NewHMAC: %v", err)
	}

	eventCount := 10_000
	events := make([]event.Event, eventCount)
	for i := range events {
		aggID := id.NewAggregateID()
		events[i] = newRealisticEvent(
			b,
			"OrderCreated",
			aggID,
			1,
			OrderCreated{
				OrderID:   aggID.String(),
				Customer:  "alice",
				Total:     199.99,
				Items:     10,
				Timestamp: time.Now().Format(time.RFC3339),
			},
		)
	}

	sigs := make([]signing.Signature, eventCount)
	for i, evt := range events {
		sigs[i], _ = signer.Sign(evt)
	}

	b.Run("Sign", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for _, evt := range events {
				if _, err := signer.Sign(evt); err != nil {
					b.Fatalf("Sign: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventCount), "events")
		b.ReportMetric(float64(b.N*eventCount)/b.Elapsed().Seconds(), "signs/sec")
	})

	b.Run("Verify", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for i, evt := range events {
				if err := signer.Verify(evt, sigs[i]); err != nil {
					b.Fatalf("Verify: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventCount), "events")
		b.ReportMetric(float64(b.N*eventCount)/b.Elapsed().Seconds(), "verifies/sec")
	})
}

// ---------------------------------------------------------------------------
// 5. Aggregate Listing — 50K aggregates, cursor-paginated
// ---------------------------------------------------------------------------

func BenchmarkRealistic_Listing(b *testing.B) {
	b.ReportAllocs()

	aggCount := 10_000

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	_ = seedOrders(b, store, aggCount, 3)

	reader := listing.NewInMemoryAggregateReader(store)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		var after id.AggregateID
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

// ---------------------------------------------------------------------------
// 6. Snapshot vs Replay — load 100 aggregates × 500 events each
// ---------------------------------------------------------------------------

func BenchmarkRealistic_SnapshotVsReplay(b *testing.B) {
	eventsPerAgg := 500
	aggCount := 100

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	snapRepo := benchNewOrderRepo(b, store, bus, 100)

	ctx := context.Background()

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggCount {
		aggIDs[i] = id.NewAggregateID()

		for range eventsPerAgg {
			if err := snapRepo.Execute(ctx, aggIDs[i], "Order",
				func(_ OrderState, ver event.Version) ([]event.Event, error) {
					return []event.Event{
						newRealisticEvent(
							b,
							"OrderCreated",
							aggIDs[i],
							ver.Increment(),
							OrderCreated{
								OrderID:   aggIDs[i].String(),
								Customer:  "test",
								Total:     10.0,
								Items:     1,
								Timestamp: time.Now().Format(time.RFC3339),
							},
						),
					}, nil
				}); err != nil {
				b.Fatalf("seed: %v", err)
			}
		}
	}

	b.Run("Replay", func(b *testing.B) {
		b.ReportAllocs()

		plainDecider := decider.Decider[OrderState]{Initial: OrderState{}, Fold: foldOrder}
		replayRepo, _ := decider.NewRepository[OrderState](store, bus, plainDecider)

		b.ResetTimer()

		for b.Loop() {
			for _, aggID := range aggIDs {
				if _, _, err := replayRepo.Load(ctx, aggID, "Order"); err != nil {
					b.Fatalf("Load: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventsPerAgg), "events_per_agg")
		b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
	})

	b.Run("Snapshot", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for b.Loop() {
			for _, aggID := range aggIDs {
				if _, _, err := snapRepo.Load(ctx, aggID, "Order"); err != nil {
					b.Fatalf("Load: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventsPerAgg), "events_per_agg")
		b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
	})
}

// ---------------------------------------------------------------------------
// 7. Query dispatch — 1K queries with paginated results
// ---------------------------------------------------------------------------

func BenchmarkRealistic_QueryDispatch(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	items := make([]OrderCreated, 1000)
	for i := range items {
		items[i] = OrderCreated{
			OrderID:   fmt.Sprintf("ORD-%04d", i),
			Customer:  fmt.Sprintf("customer-%d", i),
			Total:     float64(i) * 10.0,
			Items:     i % 20,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	if err := dispatcher.Register(
		"list.orders",
		func(_ context.Context, _ query.Query) (any, error) {
			return query.NewPaginatedResult(
				items[:50],
				uint(len(items)),
				query.NewPagination(1, 50),
			), nil
		},
	); err != nil {
		b.Fatalf("register: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for range 1000 {
			q := mustNewQuery("list.orders")
			if _, err := dispatcher.Dispatch(ctx, q); err != nil {
				b.Fatalf("Dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*1000)/b.Elapsed().Seconds(), "queries/sec")
}

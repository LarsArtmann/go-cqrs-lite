package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func mustNewCmd(commandType command.Type, aggregateID id.AggregateID, opts ...command.Option) *command.BasicCommand {
	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		panic(err)
	}
	return s
}

func mustNewQuery(queryType query.Type) *query.BasicQuery {
	q, err := query.New(queryType)
	if err != nil {
		panic(err)
	}
	return q
}


// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type benchState struct{ Value int }

func benchFold(_ benchState, evt event.Event) (benchState, error) {
	switch evt.Type() {
	case "ItemCreated":
		return benchState{Value: 1}, nil
	case "ItemUpdated":
		return benchState{Value: 2}, nil
	}

	return benchState{}, nil
}

func benchDecider() decider.Decider[benchState] {
	return decider.Decider[benchState]{Initial: benchState{}, Fold: benchFold}
}

func newBenchDeciderRepo(b *testing.B) (*decider.Repository[benchState], context.Context) {
	b.Helper()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	repo, err := decider.NewRepository(store, bus, benchDecider())
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	return repo, context.Background()
}

func newBenchEvent(
	b *testing.B,
	eventType string,
	aggID id.AggregateID,
	v event.Version,
) event.Event {
	b.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Item", v, nil)
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func noopCmdHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error { return nil }
}

func benchNoopQueryHandler(_ context.Context, _ query.Query) (any, error) {
	return nil, nil //nolint:nilnil
}

func noopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error { return nil }
}

func benchCreateItem(
	b *testing.B,
	repo *decider.Repository[benchState],
	ctx context.Context,
	aggID id.AggregateID,
) {
	b.Helper()

	err := repo.Execute(
		ctx, aggID, "Item",
		func(_ benchState, v event.Version) ([]event.Event, error) {
			evt := newBenchEvent(b, "ItemCreated", aggID, v.Increment())

			return []event.Event{evt}, nil
		},
	)
	if err != nil {
		b.Fatalf("Execute: %v", err)
	}
}

func benchCreateItemConcurrent(
	b *testing.B,
	repo *decider.Repository[benchState],
	ctx context.Context,
) id.AggregateID {
	b.Helper()

	aggID := id.NewAggregateID()
	decideFn := func(_ benchState, v event.Version) ([]event.Event, error) {
		return []event.Event{newBenchEvent(b, "ItemCreated", aggID, v.Increment())}, nil
	}

	if err := repo.Execute(ctx, aggID, "Item", decideFn); err != nil {
		b.Errorf("concurrent Execute: %v", err)
	}

	return aggID
}

// ---------------------------------------------------------------------------
// 1. Million Commands — sustained dispatch throughput
// ---------------------------------------------------------------------------

func BenchmarkScale_CommandDispatch(b *testing.B) {
	b.ReportAllocs()

	dispatcher := command.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	for i := range 100 {
		err := dispatcher.Register(
			command.Type(fmt.Sprintf("cmd.%d", i)),
			noopCmdHandler(),
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	aggID := id.NewAggregateID()
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for i := range 100 {
			cmd := mustNewCmd(command.Type(fmt.Sprintf("cmd.%d", i)), aggID)
			err := dispatcher.Dispatch(ctx, cmd)
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "commands/sec")
}

// ---------------------------------------------------------------------------
// 2. Million Events — creation + save + publish at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_EventCreation(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewAggregateID()

	b.ResetTimer()

	for b.Loop() {
		_, _ = event.NewEvent("ItemCreated", aggID, "Item", 1, nil)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkScale_EventCreation_WithPayload(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewAggregateID()
	payload, err := json.Marshal(map[string]string{"name": "test-item", "sku": "ABC-123"})
	if err != nil {
		b.Fatalf("json.Marshal: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, _ = event.NewEvent("ItemCreated", aggID, "Item", 1, payload)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkScale_EventSave_10KAggregates_100EventsEach(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	aggCount := 10_000
	eventsPerAgg := 100

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
	}

	b.ResetTimer()

	for b.Loop() {
		for _, aggID := range aggIDs {
			events := make([]event.Event, eventsPerAgg)

			for v := range eventsPerAgg {
				events[v] = newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))
			}

			err := store.AppendBatch(
				ctx,
				event.NewAggregateRef("Item", aggID),
				events,
			)
			if err != nil {
				b.Fatalf("AppendBatch: %v", err)
			}
		}
	}

	totalEvents := int64(aggCount) * int64(eventsPerAgg)
	b.ReportMetric(float64(b.N*int(totalEvents))/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkScale_EventPublish_MemoryBus_100KEvents(b *testing.B) {
	b.ReportAllocs()

	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	err := bus.SubscribeAll(noopEventHandler())
	if err != nil {
		b.Fatalf("SubscribeAll: %v", err)
	}

	aggID := id.NewAggregateID()
	events := make([]event.Event, 100)
	for i := range events {
		events[i] = newBenchEvent(b, "ItemUpdated", aggID, event.Version(i+1))
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		err := bus.Publish(ctx, events...)
		if err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "events/sec")
}

// ---------------------------------------------------------------------------
// 3. Million Aggregates — decider Execute + Load at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_DeciderExecute_ManyAggregates(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggIDs := make([]id.AggregateID, b.N)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
	}

	b.ResetTimer()

	for i := range aggIDs {
		benchCreateItem(b, repo, ctx, aggIDs[i])
	}
}

func BenchmarkScale_DeciderExecute_1000Aggregates_100UpdatesEach(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggCount := 1_000
	aggIDs := make([]id.AggregateID, aggCount)

	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()

		benchCreateItem(b, repo, ctx, aggIDs[i])
	}

	b.ResetTimer()

	for b.Loop() {
		for i := range aggIDs {
			benchCreateItem(b, repo, ctx, aggIDs[i])
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "executes/sec")
}

func BenchmarkScale_DeciderLoad_10KAggregates(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)

	aggCount := 10_000
	eventsPerAgg := 50
	aggIDs := make([]id.AggregateID, aggCount)

	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()

		for range eventsPerAgg {
			err := repo.Execute(
				ctx, aggIDs[i], "Item",
				func(_ benchState, ver event.Version) ([]event.Event, error) {
					return []event.Event{
						newBenchEvent(b, "ItemUpdated", aggIDs[i], ver.Increment()),
					}, nil
				},
			)
			if err != nil {
				b.Fatalf("seed Execute: %v", err)
			}
		}
	}

	b.ResetTimer()

	for b.Loop() {
		for _, aggID := range aggIDs {
			_, _, err := repo.Load(ctx, aggID, "Item")
			if err != nil {
				b.Fatalf("Load: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "loads/sec")
}

// ---------------------------------------------------------------------------
// 4. Thousand Projections — registration + event processing at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_ProjectionRegistration_1000(b *testing.B) {
	b.ReportAllocs()

	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	b.ResetTimer()

	var counter atomic.Int64

	for b.Loop() {
		n := counter.Add(1)
		p := event.NewProjection(
			fmt.Sprintf("projection-%d", n),
			noopEventHandler(),
			[]event.Type{"ItemCreated"},
		)

		err := runner.Register(p)
		if err != nil {
			b.Fatalf("Register: %v", err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "registrations/sec")
}

func BenchmarkScale_ProjectionProcessing_100Projections_100KEvents(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	projectionCount := 100
	eventCount := 100_000

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	var processed atomic.Int64

	for i := range projectionCount {
		counter := &processed
		p := event.NewProjection(
			fmt.Sprintf("view-%d", i),
			func(_ context.Context, _ event.Event) error {
				counter.Add(1)

				return nil
			},
			[]event.Type{"ItemCreated", "ItemUpdated"},
		)

		err := runner.Register(p)
		if err != nil {
			b.Fatalf("Register: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)

	go func() {
		runErr <- runner.Run(ctx)
	}()

	seedCtx := context.Background()
	aggID := id.NewAggregateID()

	for v := range eventCount {
		evt := newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))

		err := bus.Publish(seedCtx, evt)
		if err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}

	b.ResetTimer()

	start := time.Now()

	for b.Loop() {
		processed.Store(0)
		aggID := id.NewAggregateID()

		for v := range eventCount {
			evt := newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))
			_ = bus.Publish(seedCtx, evt)
		}
	}

	elapsed := time.Since(start)
	b.ReportMetric(
		float64(b.N*eventCount*projectionCount)/elapsed.Seconds(),
		"projection-events/sec",
	)

	cancel()
	_ = runner.Close()
}

// ---------------------------------------------------------------------------
// 5. Thousand Queries — dispatch with many registered handlers
// ---------------------------------------------------------------------------

func BenchmarkScale_QueryDispatch_1000Handlers(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	handlerCount := 1000

	for i := range handlerCount {
		err := dispatcher.Register(
			query.Type(fmt.Sprintf("query.%d", i)),
			benchNoopQueryHandler,
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for i := range handlerCount {
			q := mustNewQuery(query.Type(fmt.Sprintf("query.%d", i)))
			_, err := dispatcher.Dispatch(ctx, q)
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*handlerCount)/b.Elapsed().Seconds(), "queries/sec")
}

func BenchmarkScale_QueryDispatch_PaginatedResults(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	pageSizes := []uint{10, 50, 100, 500}
	resultData := make([]string, 1000)
	for i := range resultData {
		resultData[i] = fmt.Sprintf("item-%d", i)
	}

	for _, ps := range pageSizes {
		err := dispatcher.Register(
			query.Type(fmt.Sprintf("list.page%d", ps)),
			func(_ context.Context, _ query.Query) (any, error) {
				p := query.NewPagination(1, ps)
				end := min(int(ps), len(resultData))

				return query.NewPaginatedResult(
					resultData[:end],
					uint(len(resultData)),
					p,
				), nil
			},
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for _, ps := range pageSizes {
			q := mustNewQuery(query.Type(fmt.Sprintf("list.page%d", ps)))
			_, err := dispatcher.Dispatch(ctx, q)
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*len(pageSizes))/b.Elapsed().Seconds(), "queries/sec")
}

// ---------------------------------------------------------------------------
// 6. Thousand Materialized Views — listing aggregates at scale
// ---------------------------------------------------------------------------

func BenchmarkScale_Listing_10KAggregates(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	aggCount := 10_000

	aggIDs := make([]id.AggregateID, aggCount)
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
			event.NewAggregateRef("Item", aggIDs[i]),
			[]event.Event{evt},
		)
		if err != nil {
			b.Fatalf("AppendBatch: %v", err)
		}
	}

	reader := listing.NewInMemoryAggregateReader(store)

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
		_ = store.AppendBatch(ctx, event.NewAggregateRef("Item", aggID), []event.Event{evt})
	}

	reader := listing.NewInMemoryAggregateReader(store)

	b.ResetTimer()

	for b.Loop() {
		var after id.AggregateID

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

// ---------------------------------------------------------------------------
// 7. Full Pipeline — end-to-end: command → decider → event → projection → query
// ---------------------------------------------------------------------------

func BenchmarkScale_FullPipeline_1KAggregates(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close(); _ = checkpoint.Close() })

	repo, err := decider.NewRepository(store, bus, benchDecider())
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	var projectionEvents atomic.Int64

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	counter := &projectionEvents
	proj := event.NewProjection(
		"full-pipeline",
		func(_ context.Context, _ event.Event) error {
			counter.Add(1)

			return nil
		},
		[]event.Type{"ItemCreated", "ItemUpdated"},
	)

	err = runner.Register(proj)
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() { _ = runner.Run(ctx) }()

	queryDisp := query.NewDispatcher()
	b.Cleanup(func() { _ = queryDisp.Close() })

	err = queryDisp.Register("list.items", func(_ context.Context, _ query.Query) (any, error) {
		return map[string]int{"total": int(projectionEvents.Load())}, nil
	})
	if err != nil {
		b.Fatalf("register query: %v", err)
	}

	aggCount := 1_000

	b.ResetTimer()

	aggIDs := make([]id.AggregateID, aggCount)
	for i := range aggIDs {
		aggIDs[i] = id.NewAggregateID()
	}

	for b.Loop() {
		projectionEvents.Store(0)

		for _, aggID := range aggIDs {
			benchCreateItem(b, repo, context.Background(), aggID)
		}

		q := mustNewQuery("list.items")
		_, err := queryDisp.Dispatch(context.Background(), q)
		if err != nil {
			b.Fatalf("Dispatch: %v", err)
		}
	}

	b.ReportMetric(float64(b.N*aggCount)/b.Elapsed().Seconds(), "aggregates/sec")

	cancel()
	_ = runner.Close()
}

// ---------------------------------------------------------------------------
// 8. Concurrent — parallel command dispatch + event processing
// ---------------------------------------------------------------------------

func BenchmarkScale_Concurrent_10KCommands_8Goroutines(b *testing.B) {
	b.ReportAllocs()

	dispatcher := command.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	err := dispatcher.Register("bench.cmd", noopCmdHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	workers := 8
	opsPerWorker := 10_000

	b.ResetTimer()

	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(workers)

		for range workers {
			go func() {
				defer wg.Done()
				aggID := id.NewAggregateID()

				for range opsPerWorker {
					cmd := mustNewCmd("bench.cmd", aggID)
					_ = dispatcher.Dispatch(ctx, cmd)
				}
			}()
		}

		wg.Wait()
	}

	b.ReportMetric(float64(b.N*workers*opsPerWorker)/b.Elapsed().Seconds(), "commands/sec")
}

func BenchmarkScale_Concurrent_DeciderExecute_4Goroutines(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)
	workers := 4
	opsPerWorker := 1000

	b.ResetTimer()

	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(workers)

		for range workers {
			go func() {
				defer wg.Done()

				for range opsPerWorker {
					benchCreateItemConcurrent(b, repo, ctx)
				}
			}()
		}

		wg.Wait()
	}

	b.ReportMetric(float64(b.N*workers*opsPerWorker)/b.Elapsed().Seconds(), "executes/sec")
}

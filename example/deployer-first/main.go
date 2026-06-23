package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// This is the DEPLOYER-FIRST example. The deployer and consumer are separated:
//
//   - DEPLOYER code picks the infrastructure (memory here; swap to SQLite or
//     Pebble by changing one line — the consumer code below does not change).
//   - CONSUMER code defines the domain (domain.go, view.go) and uses stack
//     accessors to wire it together.
//
// The composable read-model pipeline is:
//
//		Decider → EventBus → CatchUpSubscriber → Materialize
//
//	  - CatchUpSubscriber replays the journal from the last checkpoint (ordered),
//	    then seamlessly switches to live delivery (with EventID-based dedup).
//	  - Materialize turns the ordered event stream into a typed, tombstone-aware,
//	    queryable view.
//
// IMPORTANT — ordering:
// The CatchUpSubscriber's output channel is a FIFO Go channel. Consuming it
// from a SINGLE goroutine guarantees event ordering (create before update
// before delete). Do NOT route through Watermill's Router for ordered
// projections — the Router processes messages in parallel (one goroutine per
// message), which breaks ordering for multi-event aggregates.
//
// Startup pattern: commands execute BEFORE the projection starts. All events
// are in the journal. The CatchUpSubscriber replays them (ordered), then enters
// live mode for any subsequent events.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── DEPLOYER: choose infrastructure ──────────────────────────────────
	//
	// This uses stack.New() with individual With* options to show the MANUAL
	// assembly path — the pattern you'd use to mix engines per-concern, e.g.
	// Pebble for events + SQL for materialized views + Redis for checkpoints.
	// For a homogeneous stack (one engine for everything), use a preset instead:
	//
	//   bundle, err := memory.New()           // or sqlite.New("app.db")
	//                                           // or pebble.New("data")
	//                                           // or postgres.New(dsn)
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
	)
	if err != nil {
		log.Fatalf("deployer: stack.New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// ── CONSUMER: build the materialized view ───────────────────────────
	mat, err := stack.NewMaterialize[TodoView, id.AggregateID](bundle, codec.JSONCodec{}, todoKey)
	if err != nil {
		log.Fatalf("consumer: NewMaterialize: %v", err)
	}

	configureMaterialize(mat)

	// ── CONSUMER: execute commands (events saved to journal) ────────────
	repo, err := stack.Repository[TodoState](bundle, decider.Decider[TodoState]{
		Initial: TodoState{},
		Apply:   applyTodo,
	})
	if err != nil {
		log.Fatalf("consumer: Repository: %v", err)
	}

	todoID := id.NewAggregateID()

	if err := repo.Execute(
		ctx,
		todoID,
		aggregateType,
		decideCreate(todoID, "Buy milk"),
	); err != nil {
		log.Fatalf("create todo: %v", err)
	}

	if err := repo.Execute(ctx, todoID, aggregateType, decideComplete(todoID)); err != nil {
		log.Fatalf("complete todo: %v", err)
	}

	if err := repo.Execute(ctx, todoID, aggregateType, decideDelete(todoID, "done")); err != nil {
		log.Fatalf("delete todo: %v", err)
	}

	// ── CONSUMER: start the projection ──────────────────────────────────
	// CatchUpSubscriber replays the journal (all 3 events in order), then
	// enters live mode for any future events.
	catchUp, err := bundle.CatchUpSubscriber()
	if err != nil {
		log.Fatalf("consumer: CatchUpSubscriber: %v", err)
	}

	msgs, err := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)
	if err != nil {
		log.Fatalf("consumer: Subscribe: %v", err)
	}

	handler := mat.HandlerFunc()

	go func() {
		for msg := range msgs {
			if err := handler(msg); err != nil {
				log.Printf("projection: handle %s: %v", msg.Metadata.Get("event_type"), err)
			}

			msg.Ack()
		}
	}()

	// ── Query the materialized view ─────────────────────────────────────
	view, ok := waitForView(ctx, mat, todoID, func(v *TodoView) bool {
		return v.Title == "Buy milk" && v.Completed && v.Tombstoned
	})
	if !ok {
		log.Fatal("timed out waiting for materialized view")
	}

	fmt.Fprintf(os.Stderr, "todo %s: %q completed=%v tombstoned=%v\n",
		todoID, view.Title, view.Completed, view.Tombstoned)
}

// waitForView polls the materialized view until it matches the predicate or the
// context expires.
func waitForView(
	ctx context.Context,
	mat *stack.Materialize[TodoView, id.AggregateID],
	todoID id.AggregateID,
	matches func(*TodoView) bool,
) (*TodoView, bool) {
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if view, err := mat.View(ctx, todoID); err == nil && matches(view) {
			return view, true
		}

		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(10 * time.Millisecond):
		}
	}

	return nil, false
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// go-cqrs-lite in 80 lines — the simplest useful example.
//
// Pipeline: Decider → Event Store → EventBus → CatchUpSubscriber → Materialize
//
// The deployer picks infrastructure (in-memory here). The consumer defines
// the domain (events, state, decide). Swap memory → sqlite/pebble/postgres
// by changing ONE line — the consumer code doesn't change.

const evtIncremented = event.Type("counter.incremented")

type IncrementedPayload struct {
	Amount int `json:"amount"`
}

type CounterState struct{ Value int }

func applyCounter(s CounterState, evt event.Event) (CounterState, error) {
	if evt.Type() != evtIncremented {
		return s, nil
	}

	p, err := event.DecodePayloadAuto[IncrementedPayload](evt)
	if err != nil {
		return s, err
	}

	s.Value += p.Amount

	return s, nil
}

// increment returns a DecideFunc that the Repository executes against
// the current (replayed) state. Version enables optimistic concurrency.
func increment(aggID id.AggregateID, amount int) decider.DecideFunc[CounterState] {
	return func(_ CounterState, v event.Version) ([]event.Event, error) {
		evt, err := event.New(evtIncremented, aggID, "Counter",
			v.Increment(), IncrementedPayload{Amount: amount})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

type CounterView struct {
	Value int `json:"value"`
}

func main() {
	ctx := context.Background()

	// ── Deployer: choose infrastructure ──────────────────────────────
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = bundle.Close() }()

	// ── Consumer: repository (load → fold → decide → save → publish) ─
	repo, err := stack.Repository(bundle, decider.Decider[CounterState]{
		Initial: CounterState{},
		Apply:   applyCounter,
	})
	if err != nil {
		log.Fatal(err)
	}

	// ── Consumer: materialized view (read model) ─────────────────────
	mat, err := stack.NewMaterialize[CounterView, id.AggregateID](bundle, nil,
		func(evt event.Event) (id.AggregateID, error) { return evt.AggregateID(), nil })
	if err != nil {
		log.Fatal(err)
	}

	mat.OnCreate = func(_ context.Context, evt event.Event) (*CounterView, error) {
		p, _ := event.DecodePayloadAuto[IncrementedPayload](evt)

		return &CounterView{Value: p.Amount}, nil
	}

	mat.OnUpdate = func(_ context.Context, evt event.Event, ex *CounterView) (*CounterView, error) {
		p, _ := event.DecodePayloadAuto[IncrementedPayload](evt)

		return &CounterView{Value: ex.Value + p.Amount}, nil
	}

	// ── Consumer: projection (ordered replay + live delivery) ────────
	// IMPORTANT: start AFTER executing commands so the CatchUpSubscriber
	// replays them from the journal. Events published before the subscriber
	// enters live mode are not lost — they're in the journal.

	// ── Execute commands (events sourced to journal) ─────────────────
	counterID := id.NewAggregateID()

	for _, amt := range []int{5, 3, 2} {
		if err := repo.Execute(ctx, counterID, "Counter", increment(counterID, amt)); err != nil {
			log.Fatal(err)
		}
	}

	// ── Start projection (replays journal, then enters live mode) ────
	catchUp, _ := bundle.CatchUpSubscriber()
	msgs, _ := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)

	go func() {
		handler := mat.HandlerFunc()
		for msg := range msgs {
			_ = handler(msg)
			msg.Ack()
		}
	}()

	// ── Query the materialized view ──────────────────────────────────
	time.Sleep(100 * time.Millisecond) // wait for projection

	view, err := mat.View(ctx, counterID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Counter %s: value=%d (expected 10)\n", counterID, view.Value)

	// Swap in-memory → persistent by changing ONE line:
	//   bundle, err := sqlite.New("counter.db")
	//   bundle, err := pebble.New("./data")
	//   bundle, err := postgres.New(dsn)
	// The domain code above doesn't change.
}

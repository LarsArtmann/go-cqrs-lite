package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type incrementCmd struct {
	Amount int
}

func decideIncrement(state counterState, cmd incrementCmd) ([]event.Event, error) {
	if state.Value == 0 {
		return []event.Event{mustNewCounterEvent("CounterCreated", id.NewAggregateID(), 1)}, nil
	}

	return []event.Event{mustNewCounterEvent("CounterIncremented", id.NewAggregateID(), 1)}, nil
}

func mustNewCounterEvent(eventType string, aggID id.AggregateID, version int) event.Event {
	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", event.Version(version), nil)
	if err != nil {
		panic(err)
	}

	return evt
}

func TestTypedDecider_ExecuteCommand(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := decider.TypedDecider[counterState, incrementCmd]{
		Initial: counterState{Value: 0},
		Decide:  decideIncrement,
		Apply:   applyCounter,
	}

	repo, err := decider.NewTypedRepository[counterState, incrementCmd](store, bus, d)
	if err != nil {
		t.Fatalf("NewTypedRepository: %v", err)
	}

	aggID := id.NewAggregateID()
	ctx := context.Background()

	err = repo.ExecuteCommand(ctx, aggID, "Counter", incrementCmd{Amount: 1})
	if err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef("Counter", aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != "CounterCreated" {
		t.Fatalf("expected CounterCreated, got %s", events[0].Type())
	}
}

func TestTypedDecider_NilPublisher(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()

	d := decider.TypedDecider[counterState, incrementCmd]{
		Initial: counterState{Value: 0},
		Decide:  decideIncrement,
		Apply:   applyCounter,
	}

	repo, err := decider.NewTypedRepository[counterState, incrementCmd](store, nil, d)
	if err != nil {
		t.Fatalf("NewTypedRepository with nil publisher should succeed: %v", err)
	}

	aggID := id.NewAggregateID()
	ctx := context.Background()

	err = repo.ExecuteCommand(ctx, aggID, "Counter", incrementCmd{Amount: 1})
	if err != nil {
		t.Fatalf("ExecuteCommand with nil publisher: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef("Counter", aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event persisted (pure-ES mode), got %d", len(events))
	}
}

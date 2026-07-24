package decider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type incrementCmd struct {
	Amount int
}

func decideIncrement(state counterState, cmd incrementCmd) ([]event.Event, error) {
	eventType := event.Type("CounterIncremented")
	if state.Value == 0 {
		eventType = "CounterCreated"
	}

	evt, err := event.NewEvent(eventType, id.NewStreamID(), "Counter", 1, nil)
	if err != nil {
		return nil, fmt.Errorf("decideIncrement: %w", err)
	}

	return []event.Event{evt}, nil
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

	streamID := id.NewStreamID()
	ctx := context.Background()

	err = repo.ExecuteCommand(ctx, streamID, "Counter", incrementCmd{Amount: 1})
	if err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}

	events, err := store.Load(ctx, id.NewStreamRef("Counter", streamID))
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

	streamID := id.NewStreamID()
	ctx := context.Background()

	err = repo.ExecuteCommand(ctx, streamID, "Counter", incrementCmd{Amount: 1})
	if err != nil {
		t.Fatalf("ExecuteCommand with nil publisher: %v", err)
	}

	events, err := store.Load(ctx, id.NewStreamRef("Counter", streamID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event persisted (pure-ES mode), got %d", len(events))
	}
}

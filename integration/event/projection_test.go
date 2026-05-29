package event_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

var errTestProjection = errors.New("projection failed")

func appendingProjection(name string, types []event.Type, handled *[]string) event.Projection {
	return event.NewProjection(name, func(_ context.Context, evt event.Event) error {
		*handled = append(*handled, string(evt.Type()))
		return nil
	}, types)
}

func TestInMemoryRunner_Handle(t *testing.T) {
	t.Parallel()

	checkpoint := memory.NewMemoryCheckpointStore()

	var handled []string

	runner, err := event.NewInMemoryRunner(checkpoint)
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := appendingProjection("user-stats", []event.Type{"UserCreated"}, &handled)
	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(handled) != 1 {
		t.Fatalf("expected 1 handled event, got %d", len(handled))
	}

	if handled[0] != "UserCreated" {
		t.Errorf("handled[0] = %q, want UserCreated", handled[0])
	}

	checkpointID, err := checkpoint.Load(context.Background(), "user-stats")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if checkpointID != evt.ID() {
		t.Errorf("checkpoint = %v, want %v", checkpointID, evt.ID())
	}
}

func TestInMemoryRunner_FiltersByEventType(t *testing.T) {
	t.Parallel()

	checkpoint := memory.NewMemoryCheckpointStore()

	var handled []string

	runner, err := event.NewInMemoryRunner(checkpoint)
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := appendingProjection("filtered", []event.Type{"UserCreated"}, &handled)
	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	created, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	deleted, _ := event.NewEvent("UserDeleted", aggID, "User", 2, nil)

	err = runner.Handle(context.Background(), created)
	if err != nil {
		t.Fatalf("Handle created: %v", err)
	}

	err = runner.Handle(context.Background(), deleted)
	if err != nil {
		t.Fatalf("Handle deleted: %v", err)
	}

	if len(handled) != 1 {
		t.Fatalf("expected 1 handled (UserCreated only), got %d", len(handled))
	}
}

func TestInMemoryRunner_SubscribesToAll(t *testing.T) {
	t.Parallel()

	checkpoint := memory.NewMemoryCheckpointStore()

	var count int

	proj := event.NewProjection(
		"all-events",
		func(_ context.Context, _ event.Event) error {
			count++

			return nil
		},
		nil,
	)

	runner, err := event.NewInMemoryRunner(checkpoint)
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	created, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	updated, _ := event.NewEvent("UserUpdated", aggID, "User", 2, nil)

	err = runner.Handle(context.Background(), created)
	if err != nil {
		t.Fatalf("Handle created: %v", err)
	}

	err = runner.Handle(context.Background(), updated)
	if err != nil {
		t.Fatalf("Handle updated: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 events handled, got %d", count)
	}
}

func TestInMemoryRunner_ProjectionError(t *testing.T) {
	t.Parallel()

	checkpoint := memory.NewMemoryCheckpointStore()

	proj := event.NewProjection(
		"failing",
		func(_ context.Context, _ event.Event) error {
			return errTestProjection
		},
		nil,
	)

	runner, err := event.NewInMemoryRunner(checkpoint)
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, nil)

	err = runner.Handle(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from failing projection")
	}
}

func TestMemoryCheckpointStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCheckpointStore()
	ctx := context.Background()

	initialCP, err := store.Load(ctx, "my-projection")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if !initialCP.IsZero() {
		t.Errorf("expected zero checkpoint, got %v", initialCP)
	}

	eventID := id.NewEventID()

	err = store.Save(ctx, "my-projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	savedCP, err := store.Load(ctx, "my-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if savedCP != eventID {
		t.Errorf("checkpoint = %v, want %v", savedCP, eventID)
	}
}

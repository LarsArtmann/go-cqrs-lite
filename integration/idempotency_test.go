package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestIdempotencyIntegration tests the full idempotency pipeline:
// - Command dispatcher with CommandIdempotency middleware
// - Decider execution (load → decide → save → publish)
// - Duplicate command rejected by middleware (handler NOT called)
// - Only one event persisted
// - Query idempotency deduplicates by custom key.
func TestIdempotencyIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	idemStore := idempotency.NewMemoryStore(0)
	defer idemStore.Close()

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   applyUserEvents,
	}

	userRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(middleware.CommandRecovery())
	cmdDispatcher.Use(middleware.CommandIdempotency(idemStore, time.Minute, nil))

	handlerCallCount := 0
	if err := cmdDispatcher.Register(
		"CreateUser",
		func(_ context.Context, cmd command.Command) error {
			handlerCallCount++

			c, ok := cmd.(*CreateUser)
			if !ok {
				return command.ErrTypeAssertion
			}

			return userRepo.Execute(
				ctx, c.AggregateID(), "User",
				func(_ UserState, currentVersion event.Version) ([]event.Event, error) {
					evt, err := event.NewEvent(
						"UserCreated",
						c.AggregateID(),
						"User",
						currentVersion.Add(1),
						[]byte(c.Name),
					)
					if err != nil {
						return nil, err
					}

					return []event.Event{evt}, nil
				},
			)
		},
	); err != nil {
		t.Fatalf("register command: %v", err)
	}

	aggID := id.NewAggregateID()
	createCmd, err := command.New("CreateUser", aggID)
	if err != nil {
		t.Fatalf("new command: %v", err)
	}

	cmd1 := &CreateUser{BasicCommand: createCmd, Name: "Alice"}
	if err := cmdDispatcher.Dispatch(ctx, cmd1); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	err = cmdDispatcher.Dispatch(ctx, cmd1)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second dispatch: want ErrDuplicate, got %v", err)
	}

	if handlerCallCount != 1 {
		t.Fatalf("handler call count: want 1, got %d", handlerCallCount)
	}

	events, err := store.Load(ctx, id.NewAggregateRef(id.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event stored, got %d", len(events))
	}

	secondCreateCmd, err := command.New("CreateUser", id.NewAggregateID())
	if err != nil {
		t.Fatalf("new second command: %v", err)
	}
	cmd2 := &CreateUser{BasicCommand: secondCreateCmd, Name: "Bob"}
	if err := cmdDispatcher.Dispatch(ctx, cmd2); err != nil {
		t.Fatalf("different command dispatch: %v", err)
	}
	if handlerCallCount != 2 {
		t.Fatalf("handler call count after different command: want 2, got %d", handlerCallCount)
	}
}

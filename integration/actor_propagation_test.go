package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestActorPropagationEndToEnd verifies the full actor audit trail:
//
//	command.WithActor → dispatcher middleware.CommandActorContext →
//	event.WithActorContext → decider enricher event.ActorEnricher →
//	stored event metadata → projection delivery.
//
// If any hop in that chain drops the actor, this test fails. It is the
// integration-level guard for the WithActor feature.
func TestActorPropagationEndToEnd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(middleware.CommandActorContext())

	userRepo, err := decider.NewRepository(store, bus, decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   applyUserEvents,
	}, decider.WithEnricher[UserState](event.ActorEnricher))
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	handler := func(hctx context.Context, cmd command.Command) error {
		c, ok := cmd.(*CreateUser)
		if !ok {
			return command.ErrTypeAssertion
		}

		return userRepo.Execute(
			hctx, c.StreamID(), "User",
			func(_ UserState, currentVersion event.Version) ([]event.Event, error) {
				evt, err := event.NewEvent(
					"UserCreated",
					c.StreamID(),
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
	}

	if err := cmdDispatcher.Register("CreateUser", handler); err != nil {
		t.Fatalf("register command: %v", err)
	}

	var mu sync.Mutex
	var projectedActor string

	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		if evt.Type() == "UserCreated" {
			mu.Lock()
			defer mu.Unlock()
			projectedActor = evt.Metadata().ActorID.PrefixedString()
		}

		return nil
	})

	actor := id.NewUserActor(id.NewUserID())

	streamID := id.NewStreamID()

	basic, err := command.New("CreateUser", streamID, command.WithActor(actor))
	if err != nil {
		t.Fatalf("new command: %v", err)
	}

	if err := cmdDispatcher.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	// Hop 1: the handler context carried the actor (middleware).
	// Verified indirectly via hop 2 — the enricher only fires if the context had it.

	// Hop 2: the stored event carries the actor (enricher + store roundtrip).
	stored, err := store.Load(ctx, id.NewStreamRef("User", streamID))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(stored) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(stored))
	}

	if got := stored[0].Metadata().ActorID; !got.Equal(actor) {
		t.Errorf("stored event actor = %q, want %q",
			got.PrefixedString(), actor.PrefixedString())
	}

	// Hop 3: the projection delivery carries the actor (bus roundtrip).
	mu.Lock()
	defer mu.Unlock()

	if projectedActor != actor.PrefixedString() {
		t.Errorf("projected actor = %q, want %q", projectedActor, actor.PrefixedString())
	}
}

package stack_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// runProjectionsUserView is the read model built by the projection under test.
type runProjectionsUserView struct {
	Name string
}

// runProjectionsKey is the typed key used by the Materialize projection.
type runProjectionsKey string

func (k runProjectionsKey) String() string { return string(k) }

// runProjectionsAggregate is a tiny aggregate used to generate events.
type runProjectionsAggregate struct{}

func TestBundle_RunProjections_ReplayAndLive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store := memory.NewMemoryStore()
	defer store.Close()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	checkpointStore := memory.NewMemoryCheckpointStore()
	defer checkpointStore.Close()

	readModels := kv.NewMemStore()
	defer readModels.Close()

	bundle, err := stack.New(
		stack.WithEventStore(store),
		stack.WithBus(bus),
		stack.WithCheckpointStore(checkpointStore),
		stack.WithReadModels(readModels),
	)
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}

	typedStore := kv.NewTypedStore[runProjectionsUserView, runProjectionsKey](readModels)

	mat := stack.Materialize[runProjectionsUserView, runProjectionsKey]{
		Store: typedStore,
		KeyFromEvent: func(evt event.Event) (runProjectionsKey, error) {
			return runProjectionsKey(evt.AggregateID().String()), nil
		},
		OnCreate: func(_ context.Context, evt event.Event) (*runProjectionsUserView, error) {
			name := string(evt.Payload())

			return &runProjectionsUserView{Name: name}, nil
		},
		OnUpdate: func(_ context.Context, evt event.Event, existing *runProjectionsUserView) (*runProjectionsUserView, error) {
			existing.Name = string(evt.Payload())

			return existing, nil
		},
	}

	repo, err := decider.NewRepository[runProjectionsAggregate](
		store,
		bus,
		decider.Decider[runProjectionsAggregate]{
			Initial: runProjectionsAggregate{},
			Apply: func(_ runProjectionsAggregate, _ event.Event) (runProjectionsAggregate, error) {
				return runProjectionsAggregate{}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	// --- Phase 1: historical event, created before RunProjections starts ---
	aggID := id.NewAggregateID()

	if err := repo.Execute(
		ctx,
		aggID,
		"User",
		func(_ runProjectionsAggregate, currentVersion event.Version) ([]event.Event, error) {
			evt, err := event.NewEvent(
				event.Type("user.created"),
				aggID,
				"User",
				currentVersion.Add(1),
				[]byte("Alice"),
			)
			if err != nil {
				return nil, err
			}

			return []event.Event{evt}, nil
		},
	); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// --- Phase 2: start RunProjections and wait for replay ---
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runErr := make(chan error, 1)

	go func() {
		runErr <- bundle.RunProjections(runCtx, &mat)
	}()

	waitForView(t, typedStore, runProjectionsKey(aggID.String()), "Alice")

	// --- Phase 3: live event, published while RunProjections is running ---
	if err := repo.Execute(
		ctx,
		aggID,
		"User",
		func(_ runProjectionsAggregate, currentVersion event.Version) ([]event.Event, error) {
			evt, err := event.NewEvent(
				event.Type("user.renamed"),
				aggID,
				"User",
				currentVersion.Add(1),
				[]byte("Bob"),
			)
			if err != nil {
				return nil, err
			}

			return []event.Event{evt}, nil
		},
	); err != nil {
		t.Fatalf("rename user: %v", err)
	}

	waitForView(t, typedStore, runProjectionsKey(aggID.String()), "Bob")

	// --- Phase 4: shutdown cleanly ---
	cancel()

	select {
	case err := <-runErr:
		if err != nil && !isContextCanceled(err) {
			t.Fatalf("RunProjections returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunProjections did not stop after context cancellation")
	}
}

func waitForView(
	t *testing.T,
	store *kv.TypedStore[runProjectionsUserView, runProjectionsKey],
	key runProjectionsKey,
	want string,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		view, err := store.Get(context.Background(), key)
		if err == nil && view.Name == want {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	view, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("view %q not found: %v", key, err)
	}

	t.Fatalf("view.Name = %q, want %q", view.Name, want)
}

func isContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}

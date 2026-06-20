package decider_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func TestNewRepository_NilChecks(t *testing.T) {
	t.Parallel()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	_, err := decider.NewRepository(nil, eventtest.NewFakeBus(), d)
	if !errors.Is(err, decider.ErrNilStore) {
		t.Fatalf("expected ErrNilStore, got %v", err)
	}

	// Nil publisher is allowed (pure event-sourcing mode).
	_, err = decider.NewRepository(eventtest.NewFakeStore(), nil, d)
	if err != nil {
		t.Fatalf("expected success with nil publisher (pure-ES mode), got %v", err)
	}

	_, err = decider.NewRepository(eventtest.NewFakeStore(), eventtest.NewFakeBus(),
		decider.Decider[counterState]{Initial: counterState{}, Fold: nil})
	if !errors.Is(err, decider.ErrNilFold) {
		t.Fatalf("expected ErrNilFold, got %v", err)
	}
}

func TestNewRepository_IncompleteSnapshotConfig(t *testing.T) {
	t.Parallel()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}
	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	strategy, strategyErr := snapshot.EveryNEvents(5)
	if strategyErr != nil {
		t.Fatal(strategyErr)
	}

	_, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStrategy[counterState](strategy),
	)
	if !errors.Is(err, decider.ErrIncompleteSnapshotConfig) {
		t.Fatalf("expected ErrIncompleteSnapshotConfig, got %v", err)
	}
}

func TestExecute_Create(t *testing.T) {
	t.Parallel()

	repo, _, bus := newTestRepo(t)
	aggID := id.NewAggregateID()

	executeCounter(t, repo, aggID, 0, 0, "CounterCreated", 1)

	if len(bus.Published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(bus.Published))
	}
}

func TestExecute_Update(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	executeCounter(t, repo, aggID, 0, 0, "CounterCreated", 1)
	executeCounter(t, repo, aggID, 1, 1, "CounterIncremented", 2)
}

func TestExecute_DecideError(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	decideErr := errors.New("rejection: email required")

	err := repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, version event.Version) ([]event.Event, error) {
			return nil, decideErr
		},
	)
	if !errors.Is(err, decideErr) {
		t.Fatalf("expected decide error, got %v", err)
	}
}

func TestExecute_FoldError(t *testing.T) {
	t.Parallel()

	repo, store := newFailingRepo(t)
	aggID := id.NewAggregateID()

	existing := makeEvent(t, "CounterCreated", aggID, 1)
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{existing})

	err := repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, version event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, 2)}, nil
		},
	)
	if err == nil {
		t.Fatal("expected fold error")
	}

	if !errors.Is(err, decider.ErrFoldFailed) {
		t.Fatalf("expected ErrFoldFailed, got %v", err)
	}
}

func TestExecute_SaveError(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	store.SaveFn(
		func(_ context.Context, _ event.AggregateRef, _ []event.Event, _ event.Version) error {
			return errors.New("db connection lost")
		},
	)

	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}
	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = executeCreate(t, repo, aggID)
	if err == nil {
		t.Fatal("expected save error")
	}

	if !errors.Is(err, decider.ErrSaveFailed) {
		t.Fatalf("expected ErrSaveFailed, got %v", err)
	}
}

func TestExecute_PublishError(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	bus.PublishErr = errors.New("bus unavailable")

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}
	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = executeCreate(t, repo, aggID)
	if err == nil {
		t.Fatal("expected publish error")
	}

	if !errors.Is(err, bus.PublishErr) {
		t.Fatalf("expected publish error, got %v", err)
	}
}

func TestExecute_NoEvents(t *testing.T) {
	t.Parallel()

	repo, _, bus := newTestRepo(t)
	aggID := id.NewAggregateID()

	err := repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(bus.Published) != 0 {
		t.Fatalf("expected 0 published events, got %d", len(bus.Published))
	}
}

func TestExecute_Concurrent(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	var wg sync.WaitGroup

	for range 5 {
		wg.Go(func() {
			_ = executeAndIncrement(t, repo, aggID, "CounterIncremented")
		})
	}

	wg.Wait()
}

func TestExecute_ContextCancellation(t *testing.T) {
	t.Parallel()

	store := &ctxCheckStore{Store: eventtest.NewFakeStore()}
	bus := eventtest.NewFakeBus()
	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	aggID := id.NewAggregateID()

	err = repo.Execute(
		ctx, aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Increment())}, nil
		},
	)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

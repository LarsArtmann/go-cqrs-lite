package decider_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestNewRepository_NilChecks(t *testing.T) {
	t.Parallel()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	_, err := decider.NewRepository(nil, testhelpers.NewFakeBus(), d)
	if !errors.Is(err, decider.ErrNilStore) {
		t.Fatalf("expected ErrNilStore, got %v", err)
	}

	_, err = decider.NewRepository(testhelpers.NewFakeStore(), nil, d)
	if !errors.Is(err, decider.ErrNilBus) {
		t.Fatalf("expected ErrNilBus, got %v", err)
	}

	_, err = decider.NewRepository(testhelpers.NewFakeStore(), testhelpers.NewFakeBus(),
		decider.Decider[counterState]{Initial: counterState{}, Fold: nil})
	if !errors.Is(err, decider.ErrNilFold) {
		t.Fatalf("expected ErrNilFold, got %v", err)
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

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	aggID := id.NewAggregateID()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold: func(_ counterState, _ event.Event) (counterState, error) {
			return counterState{}, errors.New("corrupted payload")
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	existing := makeEvent(t, "CounterCreated", aggID, 1)
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{existing})

	err = repo.Execute(
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

	store := testhelpers.NewFakeStore()
	store.SaveFn(
		func(_ context.Context, _ event.AggregateType, _ id.AggregateID, _ []event.Event, _ event.Version) error {
			return errors.New("db connection lost")
		},
	)

	bus := testhelpers.NewFakeBus()

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

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
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

func TestExecute_WithOutbox(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	outbox := testhelpers.NewFakeOutbox()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d, decider.WithOutbox[counterState](outbox))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = executeAndIncrement(t, repo, aggID, "CounterIncremented")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(bus.Published) != 0 {
		t.Fatal("expected no bus publishes with outbox")
	}

	if len(outbox.Entries) == 0 {
		t.Fatal("expected outbox entries")
	}
}

func TestExecute_OutboxAppendError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	outbox := testhelpers.NewFakeOutbox()
	outbox.AppendFn(func(_ []event.Event) error { return errors.New("outbox full") })

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d, decider.WithOutbox[counterState](outbox))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = executeAndIncrement(t, repo, aggID, "CounterIncremented")
	if err == nil {
		t.Fatal("expected outbox append error")
	}
}

func TestExecute_Concurrent(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
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

	store := &ctxCheckStore{Store: testhelpers.NewFakeStore()}
	bus := testhelpers.NewFakeBus()
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

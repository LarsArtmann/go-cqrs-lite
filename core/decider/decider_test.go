package decider_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type counterState struct {
	Value int
}

func mustAppendBatch(
	t *testing.T,
	store event.Store,
	aggType event.AggregateType,
	aggID id.AggregateID,
	events []event.Event,
) {
	t.Helper()

	err := store.AppendBatch(t.Context(), aggType, aggID, events)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
}

func foldCounter(state counterState, evt event.Event) (counterState, error) {
	switch evt.Type() {
	case "CounterCreated":
		return counterState{Value: 1}, nil
	case "CounterIncremented":
		return counterState{Value: state.Value + 1}, nil
	}

	return state, nil
}

func newTestRepo(
	t *testing.T,
) (*decider.Repository[counterState], *testhelpers.FakeStore, *testhelpers.FakeBus) {
	t.Helper()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, store, bus
}

func makeEvent(
	t *testing.T,
	eventType string,
	aggID id.AggregateID,
	version event.Version,
) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", version, []byte("{}"))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func executeCounter(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	expectedValue int,
	expectedVersion event.Version,
	eventType string,
	eventVersion event.Version,
) {
	t.Helper()

	err := repo.Execute(
		t.Context(), aggID, "Counter",
		func(state counterState, version event.Version) ([]event.Event, error) {
			if state.Value != expectedValue {
				t.Fatalf("expected Value=%d, got %d", expectedValue, state.Value)
			}

			if version != expectedVersion {
				t.Fatalf("expected version %d, got %d", expectedVersion, version)
			}

			return []event.Event{makeEvent(t, eventType, aggID, eventVersion)}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

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

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, 1)}, nil
		},
	)
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

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, 1)}, nil
		},
	)
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

func TestLoad_NoEvents(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 0 {
		t.Fatalf("expected initial state Value=0, got %d", state.Value)
	}

	if version != 0 {
		t.Fatalf("expected version 0, got %d", version)
	}
}

func TestLoad_WithEvents(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
		makeEvent(t, "CounterIncremented", aggID, 2),
	})

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 2 {
		t.Fatalf("expected Value=2, got %d", state.Value)
	}

	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}
}

func TestLoad_FoldError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	aggID := id.NewAggregateID()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold: func(_ counterState, _ event.Event) (counterState, error) {
			return counterState{}, errors.New("corrupted")
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected fold error")
	}

	if !errors.Is(err, decider.ErrFoldFailed) {
		t.Fatalf("expected ErrFoldFailed, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	err := repo.Delete(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
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

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Increment())}, nil
		},
	)
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

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Increment())}, nil
		},
	)
	if err == nil {
		t.Fatal("expected outbox append error")
	}
}

func TestExecute_WithSnapshot(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](event.MustEveryNEvents(2)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Increment())}, nil
		},
	)
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Increment())}, nil
		},
	)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}

	saved := snapshotStore.Saved()
	if len(saved) == 0 {
		t.Fatal("expected snapshot to be saved")
	}

	if saved[0].Version.Int() != 2 {
		t.Fatalf("expected snapshot version 2, got %d", saved[0].Version.Int())
	}
}

func TestLoad_WithSnapshot(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	snapState, _ := codec.Encode(counterState{Value: 1})
	_ = snapshotStore.Save(t.Context(), event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         snapState,
		CreatedAt:     time.Now(),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 1 {
		t.Fatalf("expected Value=1 from snapshot, got %d", state.Value)
	}

	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
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

	for i := range 5 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = repo.Execute(
				t.Context(), aggID, "Counter",
				func(_ counterState, v event.Version) ([]event.Event, error) {
					return []event.Event{
						makeEvent(t, "CounterIncremented", aggID, v.Increment()),
					}, nil
				},
			)
		}()

		_ = i
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

type ctxCheckStore struct{ event.Store }

func (c *ctxCheckStore) checkCtx(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (c *ctxCheckStore) Save(
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if err := c.checkCtx(ctx); err != nil {
		return err
	}

	return c.Store.Save(ctx, aggType, aggID, events, expectedVersion)
}

func (c *ctxCheckStore) Load(
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
) ([]event.Event, error) {
	if err := c.checkCtx(ctx); err != nil {
		return nil, err
	}

	return c.Store.Load(ctx, aggType, aggID)
}

func TestLoad_SnapshotDecodeError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()
	snapshotStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         []byte("}}}not-json{{{"),
		CreatedAt:     time.Now(),
	})

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}
	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected decode error from malformed snapshot")
	}
}

func TestLoad_SnapshotStoreLoadError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	snapshotStore.SetLoadError(errors.New("db unavailable"))
	codec := event.JSONCodec{}

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}
	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()
	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected error from snapshot store load failure")
	}
}

func TestLoad_SnapshotFoldError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()

	snapState, _ := codec.Encode(counterState{Value: 1})
	_ = snapshotStore.Save(t.Context(), event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         snapState,
		CreatedAt:     time.Now(),
	})

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 2),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold: func(_ counterState, evt event.Event) (counterState, error) {
			if evt.Version() > 1 {
				return counterState{}, errors.New("corrupted event")
			}

			return counterState{Value: 1}, nil
		},
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected fold error during snapshot replay")
	}

	if !errors.Is(err, decider.ErrFoldFailed) {
		t.Fatalf("expected ErrFoldFailed, got %v", err)
	}
}

func TestExecute_SaveSnapshotError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	snapshotStore.SetSaveError(errors.New("disk full"))
	codec := event.JSONCodec{}

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](event.MustEveryNEvents(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Increment())}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute should succeed despite snapshot save error: %v", err)
	}
}

func TestDelete_StoreError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore().DeleteFn(
		func(_ event.AggregateType, _ id.AggregateID) error {
			return errors.New("disk error")
		},
	)
	bus := testhelpers.NewFakeBus()
	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Delete(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected error from store delete failure")
	}
}

func TestEveryNEvents_PanicsOnZero(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EveryNEvents(0)")
		}
	}()

	event.MustEveryNEvents(0)
}

func TestEveryNEvents_PanicsOnNegative(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EveryNEvents(-1)")
		}
	}()

	event.MustEveryNEvents(-1)
}

func TestLoad_SnapshotWithEventsAfter(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()

	snapState, _ := codec.Encode(counterState{Value: 5})
	snapshotStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(5),
		State:         snapState,
		CreatedAt:     time.Now(),
	})

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 6),
		makeEvent(t, "CounterIncremented", aggID, 7),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 7 {
		t.Fatalf("expected Value=7 (5 from snapshot + 2 events), got %d", state.Value)
	}

	if version != 7 {
		t.Fatalf("expected version 7, got %d", version)
	}
}

func TestLoad_SnapshotStoreLoadFromVersionError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()

	snapState, _ := codec.Encode(counterState{Value: 1})
	snapshotStore.SetSnapshot(&event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         snapState,
		CreatedAt:     time.Now(),
	})

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 2),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	store2 := store.LoadFromVersionFn(
		func(_ event.AggregateType, _ id.AggregateID, _ event.Version) ([]event.Event, error) {
			return nil, errors.New("db unavailable")
		},
	)

	repo, err := decider.NewRepository(
		store2,
		bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected error from LoadFromVersion failure")
	}
}

func TestLoad_SnapshotNil(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	aggID := id.NewAggregateID()
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != 1 {
		t.Fatalf("expected Value=1 from store fallback, got %d", state.Value)
	}

	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}
}

func TestLoad_StoreLoadError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore().LoadFn(
		func(_ event.AggregateType, _ id.AggregateID) ([]event.Event, error) {
			return nil, errors.New("db unavailable")
		},
	)
	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()
	_, _, err = repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected error from store load failure")
	}
}

func TestExecute_SaveSnapshotFoldError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	foldErr := errors.New("corrupt payload")

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold: func(_ counterState, _ event.Event) (counterState, error) {
			return counterState{}, foldErr
		},
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](event.MustEveryNEvents(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Increment())}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute should succeed despite fold error during snapshot save: %v", err)
	}

	if len(snapshotStore.Saved()) > 0 {
		t.Fatal("snapshot should not be saved when fold fails")
	}
}

func TestRepository_LoadAtVersion(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := makeEvent(t, "CounterCreated", aggID, 1)
	evt2 := makeEvent(t, "CounterIncremented", aggID, 2)
	evt3 := makeEvent(t, "CounterIncremented", aggID, 3)

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt1, evt2, evt3})

	state, version, err := repo.LoadAtVersion(ctx, aggID, "Counter", 2)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}

	if state.Value != 2 {
		t.Fatalf("expected state value 2 (created + incremented), got %d", state.Value)
	}
}

func TestRepository_LoadAtVersion_NotFound(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()

	state, version, err := repo.LoadAtVersion(ctx, aggID, "Counter", 5)
	if err != nil {
		t.Fatalf("LoadAtVersion not found: %v", err)
	}

	if version != 0 {
		t.Fatalf("expected version 0, got %d", version)
	}

	if state.Value != 0 {
		t.Fatalf("expected initial state, got %d", state.Value)
	}
}

func TestRepository_LoadAtTime(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1, _ := event.NewEvent(
		"CounterCreated",
		aggID,
		"Counter",
		1,
		[]byte("{}"),
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"CounterIncremented",
		aggID,
		"Counter",
		2,
		[]byte("{}"),
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent(
		"CounterIncremented",
		aggID,
		"Counter",
		3,
		[]byte("{}"),
		event.WithOccurredAt(now),
	)

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt1, evt2, evt3})

	state, version, err := repo.LoadAtTime(ctx, aggID, "Counter", now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadAtTime: %v", err)
	}

	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}

	if state.Value != 2 {
		t.Fatalf("expected state value 2, got %d", state.Value)
	}
}

func TestRepository_LoadAtTime_NotFound(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()

	state, version, err := repo.LoadAtTime(ctx, aggID, "Counter", time.Now())
	if err != nil {
		t.Fatalf("LoadAtTime not found: %v", err)
	}

	if version != 0 {
		t.Fatalf("expected version 0, got %d", version)
	}

	if state.Value != 0 {
		t.Fatalf("expected initial state, got %d", state.Value)
	}
}

type errStore struct {
	event.Store

	loadToVersionErr   error
	loadToTimestampErr error
}

func (e *errStore) LoadToVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, e.loadToVersionErr
}

func (e *errStore) LoadToTimestamp(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ time.Time,
) ([]event.Event, error) {
	return nil, e.loadToTimestampErr
}

func TestRepository_LoadAtVersion_StoreError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	store := &errStore{
		Store:            testhelpers.NewFakeStore(),
		loadToVersionErr: errors.New("db connection lost"),
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.LoadAtVersion(context.Background(), id.NewAggregateID(), "Counter", 5)
	if err == nil {
		t.Fatal("expected error from LoadAtVersion")
	}
}

func TestRepository_LoadAtTime_StoreError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()

	d := decider.Decider[counterState]{Initial: counterState{}, Fold: foldCounter}

	store := &errStore{
		Store:              testhelpers.NewFakeStore(),
		loadToTimestampErr: errors.New("db connection lost"),
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.LoadAtTime(context.Background(), id.NewAggregateID(), "Counter", time.Now())
	if err == nil {
		t.Fatal("expected error from LoadAtTime")
	}
}

func TestRepository_LoadAtVersion_FoldError(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()

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

	aggID := id.NewAggregateID()
	evt := makeEvent(t, "CounterCreated", aggID, 1)

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt})

	_, _, err = repo.LoadAtVersion(context.Background(), aggID, "Counter", 5)
	if err == nil {
		t.Fatal("expected fold error from LoadAtVersion")
	}
}

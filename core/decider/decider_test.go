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

func makeEvent(t *testing.T, eventType string, aggID id.AggregateID, version int) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", version, []byte("{}"))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
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

	err := repo.Execute(t.Context(), aggID, "Counter",
		func(state counterState, version event.Version) ([]event.Event, error) {
			if state.Value != 0 {
				t.Fatalf("expected initial state, got Value=%d", state.Value)
			}

			if version != 0 {
				t.Fatalf("expected version 0, got %d", version)
			}

			return []event.Event{makeEvent(t, "CounterCreated", aggID, 1)}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(bus.Published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(bus.Published))
	}
}

func TestExecute_Update(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	err := repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, version event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, 1)}, nil
		},
	)
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(state counterState, version event.Version) ([]event.Event, error) {
			if state.Value != 1 {
				t.Fatalf("expected Value=1 after fold, got %d", state.Value)
			}

			if version != 1 {
				t.Fatalf("expected version 1, got %d", version)
			}

			return []event.Event{makeEvent(t, "CounterIncremented", aggID, 2)}, nil
		},
	)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
}

func TestExecute_DecideError(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	aggID := id.NewAggregateID()

	decideErr := errors.New("rejection: email required")

	err := repo.Execute(t.Context(), aggID, "Counter",
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

	err = repo.Execute(t.Context(), aggID, "Counter",
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

	err = repo.Execute(t.Context(), aggID, "Counter",
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

	err = repo.Execute(t.Context(), aggID, "Counter",
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

	err := repo.Execute(t.Context(), aggID, "Counter",
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

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Int()+1)}, nil
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

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Int()+1)}, nil
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

	repo, err := decider.NewRepository(store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](decider.EveryNEvents(2)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Int()+1)}, nil
		},
	)
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Int()+1)}, nil
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

	repo, err := decider.NewRepository(store, bus, d,
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

			_ = repo.Execute(t.Context(), aggID, "Counter",
				func(_ counterState, v event.Version) ([]event.Event, error) {
					return []event.Event{makeEvent(t, "CounterIncremented", aggID, v.Int()+1)}, nil
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

	err = repo.Execute(ctx, aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Int()+1)}, nil
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
	repo, err := decider.NewRepository(store, bus, d,
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
	repo, err := decider.NewRepository(store, bus, d,
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

	repo, err := decider.NewRepository(store, bus, d,
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

	repo, err := decider.NewRepository(store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](decider.EveryNEvents(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, v.Int()+1)}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute should succeed despite snapshot save error: %v", err)
	}
}

func TestDelete_StoreError(t *testing.T) {
	t.Parallel()

	store := &failingDeleteStore{}
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

type failingDeleteStore struct{ event.Store }

func (f *failingDeleteStore) Save(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
	_ event.Version,
) error {
	return nil
}

func (f *failingDeleteStore) AppendBatch(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
) error {
	return nil
}

func (f *failingDeleteStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) ([]event.Event, error) {
	return nil, nil
}

func (f *failingDeleteStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, nil
}

func (f *failingDeleteStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return errors.New("disk error")
}

func (f *failingDeleteStore) Close() error { return nil }

func TestEveryNEvents_PanicsOnZero(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EveryNEvents(0)")
		}
	}()

	decider.EveryNEvents(0)
}

func TestEveryNEvents_PanicsOnNegative(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EveryNEvents(-1)")
		}
	}()

	decider.EveryNEvents(-1)
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

	repo, err := decider.NewRepository(store, bus, d,
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

	repo, err := decider.NewRepository(
		&failingLoadFromVersionStore{Store: store},
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

	repo, err := decider.NewRepository(store, bus, d,
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

	store := &failingLoadStore{}
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

type failingLoadStore struct{}

func (f *failingLoadStore) Save(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
	_ event.Version,
) error {
	return nil
}

func (f *failingLoadStore) AppendBatch(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
) error {
	return nil
}

func (f *failingLoadStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) ([]event.Event, error) {
	return nil, errors.New("db unavailable")
}

func (f *failingLoadStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, nil
}

func (f *failingLoadStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return nil
}

func (f *failingLoadStore) Close() error { return nil }

type failingLoadFromVersionStore struct{ event.Store }

func (f *failingLoadFromVersionStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("db unavailable")
}

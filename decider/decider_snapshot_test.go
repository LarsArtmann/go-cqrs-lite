package decider_test

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func TestExecute_WithSnapshot(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStrategy[counterState](mustEveryN(2)),
	)

	aggID := id.NewAggregateID()

	err := executeAndIncrement(t, repo, aggID, "CounterCreated")
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = executeAndIncrement(t, repo, aggID, "CounterIncremented")
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

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	saveSnapshot(t, snapshotStore, codec, aggID, 1, event.Version(1))

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, aggID, 1, 1)
}

func TestLoad_SnapshotDecodeError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()
	snapshotStore.SetSnapshot(&snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         []byte("}}}not-json{{{"),
		CreatedAt:     time.Now(),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	_, _, err := repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected decode error from malformed snapshot")
	}
}

func TestLoad_SnapshotStoreLoadError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)
	snapshotStore.SetLoadError(errors.New("db unavailable"))

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	aggID := id.NewAggregateID()
	_, _, err := repo.Load(t.Context(), aggID, "Counter")
	if err == nil {
		t.Fatal("expected error from snapshot store load failure")
	}
}

func TestLoad_SnapshotFoldError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()

	saveSnapshot(t, snapshotStore, codec, aggID, 1, event.Version(1))

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 2),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply: func(_ counterState, evt event.Event) (counterState, error) {
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
		t.Fatal("expected apply error during snapshot replay")
	}

	if !errors.Is(err, decider.ErrApplyFailed) {
		t.Fatalf("expected ErrApplyFailed, got %v", err)
	}
}

func TestExecute_SaveSnapshotError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)
	snapshotStore.SetSaveError(errors.New("disk full"))

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStrategy[counterState](mustEveryN(1)),
	)

	aggID := id.NewAggregateID()

	err := executeAndIncrement(t, repo, aggID, "CounterCreated")
	if err != nil {
		t.Fatalf("Execute should succeed despite snapshot save error: %v", err)
	}
}

func TestExecute_SaveSnapshotFoldError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	foldErr := errors.New("corrupt payload")

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply: func(_ counterState, _ event.Event) (counterState, error) {
			return counterState{}, foldErr
		},
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](mustEveryN(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = executeAndIncrement(t, repo, aggID, "CounterCreated")
	if err != nil {
		t.Fatalf("Execute should succeed despite apply error during snapshot save: %v", err)
	}

	if len(snapshotStore.Saved()) > 0 {
		t.Fatal("snapshot should not be saved when apply fails")
	}
}

func TestEveryNEvents_PanicsOnNegative(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for EveryNEvents(-1)")
		}
	}()

	mustEveryN(-1)
}

func TestLoad_SnapshotWithEventsAfter(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()

	events := make([]event.Event, 5)
	for i := range events {
		events[i] = makeEvent(t, "CounterIncremented", aggID, event.Version(i+1))
	}
	mustAppendBatch(t, store, "Counter", aggID, events)

	applySnapshot(t, snapshotStore, codec, aggID, 5, event.Version(5))

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 6),
		makeEvent(t, "CounterIncremented", aggID, 7),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, aggID, 7, 7)
}

func TestLoad_SnapshotStoreLoadFromVersionError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()

	applySnapshot(t, snapshotStore, codec, aggID, 1, event.Version(1))

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterIncremented", aggID, 2),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   applyCounter,
	}

	store2 := store.LoadFromVersionFn(
		func(_ event.AggregateRef, _ event.Version) ([]event.Event, error) {
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

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	aggID := id.NewAggregateID()
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, aggID, 1, 1)
}

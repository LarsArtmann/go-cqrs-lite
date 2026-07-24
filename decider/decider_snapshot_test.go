package decider_test

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func TestExecute_WithSnapshot(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStrategy[counterState](everyN(2)),
	)

	streamID := id.NewStreamID()

	err := executeAndIncrement(t, repo, streamID, "CounterCreated")
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = executeAndIncrement(t, repo, streamID, "CounterIncremented")
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

	streamID := id.NewStreamID()
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	saveSnapshot(t, snapshotStore, codec, streamID, 1, event.Version(1))

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, streamID, 1, 1)
}

func TestLoad_SnapshotDecodeError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()
	snapshotStore.SetSnapshot(&snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    event.Version(1),
		State:      []byte("}}}not-json{{{"),
		CreatedAt:  time.Now(),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	_, _, err := repo.Load(t.Context(), streamID, "Counter")
	if err == nil {
		t.Fatal("expected decode error from malformed snapshot")
	}
}

func TestLoad_SnapshotStoreLoadError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)
	snapshotStore.SetLoadError(errors.New("db unavailable"))

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	streamID := id.NewStreamID()
	_, _, err := repo.Load(t.Context(), streamID, "Counter")
	if err == nil {
		t.Fatal("expected error from snapshot store load failure")
	}
}

func TestLoad_SnapshotFoldError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()

	saveSnapshot(t, snapshotStore, codec, streamID, 1, event.Version(1))

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterIncremented", streamID, 2),
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

	_, _, err = repo.Load(t.Context(), streamID, "Counter")
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
		decider.WithSnapshotStrategy[counterState](everyN(1)),
	)

	streamID := id.NewStreamID()

	err := executeAndIncrement(t, repo, streamID, "CounterCreated")
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
		decider.WithSnapshotStrategy[counterState](everyN(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	err = executeAndIncrement(t, repo, streamID, "CounterCreated")
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

	everyN(-1)
}

func TestLoad_SnapshotWithEventsAfter(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()

	events := make([]event.Event, 5)
	for i := range events {
		events[i] = makeEvent(t, "CounterIncremented", streamID, event.Version(i+1))
	}
	mustAppendBatch(t, store, "Counter", streamID, events)

	applySnapshot(t, snapshotStore, codec, streamID, 5, event.Version(5))

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterIncremented", streamID, 6),
		makeEvent(t, "CounterIncremented", streamID, 7),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, streamID, 7, 7)
}

func TestLoad_SnapshotStoreLoadFromVersionError(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()

	applySnapshot(t, snapshotStore, codec, streamID, 1, event.Version(1))

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterIncremented", streamID, 2),
	})

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   applyCounter,
	}

	store2 := store.LoadFromVersionFn(
		func(_ id.StreamRef, _ event.Version) ([]event.Event, error) {
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

	_, _, err = repo.Load(t.Context(), streamID, "Counter")
	if err == nil {
		t.Fatal("expected error from LoadFromVersion failure")
	}
}

func TestLoad_SnapshotNil(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	repo := newCounterSnapshotRepo(t, store, bus, snapshotStore, codec)

	requireLoadState(t, repo, streamID, 1, 1)
}

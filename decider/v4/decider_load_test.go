package decider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestLoad_NoEvents(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	requireLoadState(t, repo, streamID, 0, 0)
}

func TestLoad_WithEvents(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
		makeEvent(t, "CounterIncremented", streamID, 2),
	})

	requireLoadState(t, repo, streamID, 2, 2)
}

func TestLoad_FoldError(t *testing.T) {
	t.Parallel()

	repo, store := newFailingRepo(t)
	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	_, _, err := repo.Load(t.Context(), streamID, "Counter")
	if err == nil {
		t.Fatal("expected apply error")
	}

	if !errors.Is(err, decider.ErrApplyFailed) {
		t.Fatalf("expected ErrApplyFailed, got %v", err)
	}
}

func TestLoad_StoreLoadError(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore().LoadFn(
		func(_ id.StreamRef) ([]event.Event, error) {
			return nil, errors.New("db unavailable")
		},
	)
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()
	_, _, err = repo.Load(t.Context(), streamID, "Counter")
	if err == nil {
		t.Fatal("expected error from store load failure")
	}
}

func TestRepository_LoadAtVersion(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	ctx := context.Background()

	streamID := id.NewStreamID()
	evt1 := makeEvent(t, "CounterCreated", streamID, 1)
	evt2 := makeEvent(t, "CounterIncremented", streamID, 2)
	evt3 := makeEvent(t, "CounterIncremented", streamID, 3)

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{evt1, evt2, evt3})

	state, version, err := repo.LoadAtVersion(ctx, streamID, "Counter", 2)
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

	streamID := id.NewStreamID()

	state, version, err := repo.LoadAtVersion(ctx, streamID, "Counter", 5)
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

	streamID := id.NewStreamID()
	now := time.Now()

	evt1 := eventtest.QuickEventOpts("CounterCreated", streamID, "Counter", 1, []byte("{}"),
		event.WithOccurredAt(now.Add(-2*time.Hour)))
	evt2 := eventtest.QuickEventOpts("CounterIncremented", streamID, "Counter", 2, []byte("{}"),
		event.WithOccurredAt(now.Add(-1*time.Hour)))
	evt3 := eventtest.QuickEventOpts("CounterIncremented", streamID, "Counter", 3, []byte("{}"),
		event.WithOccurredAt(now))

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{evt1, evt2, evt3})

	state, version, err := repo.LoadAtTime(ctx, streamID, "Counter", now.Add(-30*time.Minute))
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

	streamID := id.NewStreamID()

	state, version, err := repo.LoadAtTime(ctx, streamID, "Counter", time.Now())
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

func TestRepository_LoadAtVersion_StoreError(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{Initial: counterState{}, Apply: applyCounter}

	store := &errStore{
		Store:            eventtest.NewFakeStore(),
		loadToVersionErr: errors.New("db connection lost"),
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.LoadAtVersion(context.Background(), id.NewStreamID(), "Counter", 5)
	if err == nil {
		t.Fatal("expected error from LoadAtVersion")
	}
}

func TestRepository_LoadAtTime_StoreError(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{Initial: counterState{}, Apply: applyCounter}

	store := &errStore{
		Store:              eventtest.NewFakeStore(),
		loadToTimestampErr: errors.New("db connection lost"),
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, _, err = repo.LoadAtTime(context.Background(), id.NewStreamID(), "Counter", time.Now())
	if err == nil {
		t.Fatal("expected error from LoadAtTime")
	}
}

func TestRepository_LoadAtVersion_FoldError(t *testing.T) {
	t.Parallel()

	repo, store := newFailingRepo(t)

	streamID := id.NewStreamID()
	evt := makeEvent(t, "CounterCreated", streamID, 1)

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{evt})

	_, _, err := repo.LoadAtVersion(context.Background(), streamID, "Counter", 5)
	if err == nil {
		t.Fatal("expected apply error from LoadAtVersion")
	}
}

package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestPebbleEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()

	evt1 := cfg.newTestEvent(t, aggID, 1)
	evt2 := cfg.newTestEvent(t, aggID, 2)
	evt3 := cfg.newTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, cfg.aggType, aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadFromVersion(ctx, cfg.aggType, aggID, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(events))
	}

	if events[0].Version() != 2 {
		t.Fatalf("expected version 2, got %d", events[0].Version())
	}
}

func TestPebbleEventStore_LoadFromVersion_Empty(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()

	evt1 := cfg.newTestEvent(t, aggID, 1)
	err := store.AppendBatch(ctx, cfg.aggType, aggID, []event.Event{evt1})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadFromVersion(ctx, cfg.aggType, aggID, 5)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestPebbleEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()

	evt1 := cfg.newTestEvent(t, aggID, 1)
	evt2 := cfg.newTestEvent(t, aggID, 2)
	evt3 := cfg.newTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, cfg.aggType, aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(ctx, cfg.aggType, aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestPebbleEventStore_LoadToVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)

	_, err := store.LoadToVersion(context.Background(), "Issue", id.NewAggregateID(), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestPebbleEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	now := time.Now()

	evt1 := cfg.newTestEvent(t, aggID, 1, event.WithOccurredAt(now.Add(-2*time.Hour)))
	evt2 := cfg.newTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(-1*time.Hour)))
	evt3 := cfg.newTestEvent(t, aggID, 3, event.WithOccurredAt(now))

	err := store.AppendBatch(ctx, cfg.aggType, aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(ctx, cfg.aggType, aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestPebbleEventStore_LoadToTimestamp_NotFound(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)

	_, err := store.LoadToTimestamp(context.Background(), "Issue", id.NewAggregateID(), time.Now())
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestPebbleEventStore_ConcurrentSave_VersionConflict(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()

	evt1 := cfg.newTestEvent(t, aggID, 1)
	saveCfgEvent(t, store, cfg, aggID, evt1)

	const goroutines = 10
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			evt := cfg.newTestEvent(t, aggID, 2)
			errCh <- store.Save(
				context.Background(),
				cfg.aggType,
				aggID,
				[]event.Event{evt},
				event.Version(1),
			)
		}()
	}

	var successes, conflicts int

	for i := 0; i < goroutines; i++ {
		err := <-errCh
		if err == nil {
			successes++
		} else if errors.Is(err, event.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful save, got %d", successes)
	}

	if conflicts != goroutines-1 {
		t.Fatalf("expected %d conflicts, got %d", goroutines-1, conflicts)
	}

	loaded, err := store.Load(context.Background(), cfg.aggType, aggID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after concurrent save, got %d", len(loaded))
	}
}

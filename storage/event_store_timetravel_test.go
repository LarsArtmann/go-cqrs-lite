package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestSQLiteEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := sqliteTestEvent(t, aggID, 1)
	evt2 := sqliteTestEvent(t, aggID, 2)
	evt3 := sqliteTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(ctx, "Issue", aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestSQLiteEventStore_LoadToVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	_, err := store.LoadToVersion(context.Background(), "Issue", id.NewAggregateID(), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestSQLiteEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1 := sqliteTestEvent(t, aggID, 1, event.WithOccurredAt(now.Add(-2*time.Hour)))
	evt2 := sqliteTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(-1*time.Hour)))
	evt3 := sqliteTestEvent(t, aggID, 3, event.WithOccurredAt(now))

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(ctx, "Issue", aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestSQLiteEventStore_LoadToTimestamp_NotFound(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	_, err := store.LoadToTimestamp(
		context.Background(), "Issue",
		id.NewAggregateID(), time.Now(),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestSQLiteEventStore_LoadAllFromPosition(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()

	evt1 := sqliteTestEvent(t, aggID, 1)

	time.Sleep(2 * time.Millisecond)
	evt2 := sqliteTestEvent(t, aggID, 2)

	time.Sleep(2 * time.Millisecond)
	evt3 := sqliteTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadAllFromPosition(ctx, evt1.ID(), 1)
	if err != nil {
		t.Fatalf("LoadAllFromPosition: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event after position, got %d", len(events))
	}

	if events[0].ID() != evt2.ID() {
		t.Fatalf("expected evt2 (version 2), got event with version %d", events[0].Version())
	}

	all, err := store.LoadAllFromPosition(ctx, evt1.ID(), 0)
	if err != nil {
		t.Fatalf("LoadAllFromPosition no limit: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events after position with no limit, got %d", len(all))
	}
}

func TestSQLiteEventStore_LoadAllFromPosition_ZeroID(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := sqliteTestEvent(t, aggID, 1)

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadAllFromPosition(ctx, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("LoadAllFromPosition zero ID: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSQLiteEventStore_LoadAllFromPosition_NoLimit(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()

	for i := range 5 {
		evt := sqliteTestEvent(t, aggID, event.Version(i+1))
		err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt})
		if err != nil {
			t.Fatalf("AppendBatch: %v", err)
		}
	}

	events, err := store.LoadAllFromPosition(ctx, id.EventID{}, 0)
	if err != nil {
		t.Fatalf("LoadAllFromPosition no limit: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

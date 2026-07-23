package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func appendFiveEvents(t *testing.T, store *SQLEventStore, aggID id.StreamID) {
	t.Helper()
	ctx := context.Background()

	for i := range 5 {
		evt := issueStoreConfig().NewTestEvent(t, aggID, event.Version(i+1))
		if err := store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("Issue"), aggID),
			[]event.Event{evt},
		); err != nil {
			t.Fatalf("AppendBatch: %v", err)
		}
	}
}

func assertEventsMatch(t *testing.T, events []event.Event, expected event.Event, desc string) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("expected 1 event %s, got %d", desc, len(events))
	}
	if events[0].ID() != expected.ID() {
		t.Fatalf("expected event %s, got event with version %d", desc, events[0].Version())
	}
}

func TestSQLiteEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewStreamID()
	evt1 := issueStoreConfig().NewTestEvent(t, aggID, 1)
	evt2 := issueStoreConfig().NewTestEvent(t, aggID, 2)
	evt3 := issueStoreConfig().NewTestEvent(t, aggID, 3)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		2,
	)
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

	_, err := store.LoadToVersion(
		context.Background(),
		id.NewStreamRef("Issue", id.NewStreamID()),
		5,
	)
	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound, got: %v", err)
	}
}

func TestSQLiteEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewStreamID()
	now := time.Now()

	evt1 := issueStoreConfig().NewTestEvent(t, aggID, 1, event.WithOccurredAt(now.Add(-2*time.Hour)))
	evt2 := issueStoreConfig().NewTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(-1*time.Hour)))
	evt3 := issueStoreConfig().NewTestEvent(t, aggID, 3, event.WithOccurredAt(now))

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		now.Add(-30*time.Minute),
	)
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
		context.Background(), id.NewStreamRef("Issue", id.NewStreamID()),
		time.Now(),
	)
	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound, got: %v", err)
	}
}

func TestSQLiteEventStore_ReadFrom(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewStreamID()

	evt1 := issueStoreConfig().NewTestEvent(t, aggID, 1)

	time.Sleep(2 * time.Millisecond)
	evt2 := issueStoreConfig().NewTestEvent(t, aggID, 2)

	time.Sleep(2 * time.Millisecond)
	evt3 := issueStoreConfig().NewTestEvent(t, aggID, 3)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.ReadFrom(ctx, evt1.ID(), 1)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	assertEventsMatch(t, events, evt2, "(version 2)")

	all, err := store.ReadFrom(ctx, evt1.ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom no limit: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events after position with no limit, got %d", len(all))
	}
}

func TestSQLiteEventStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	ctx := context.Background()

	aggID := id.NewStreamID()
	evt := issueStoreConfig().NewTestEvent(t, aggID, 1)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID),
		[]event.Event{evt},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.ReadFrom(ctx, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom zero ID: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSQLiteEventStore_ReadFrom_NoLimit(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewStreamID()
	appendFiveEvents(t, store, aggID)

	events, err := store.ReadFrom(context.Background(), id.EventID{}, 0)
	if err != nil {
		t.Fatalf("ReadFrom no limit: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

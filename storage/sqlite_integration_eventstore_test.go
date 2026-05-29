package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func setupTwoTestEvents(
	t *testing.T,
	store *SQLEventStore,
) (event.Event, event.Event, id.AggregateID, id.AggregateID) {
	t.Helper()
	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := issueStoreConfig().newTestEvent(
		t,
		aggID1,
		1,
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	evt2 := issueStoreConfig().newTestEvent(
		t,
		aggID2,
		1,
		event.WithOccurredAt(time.Now().Add(time.Second).Truncate(time.Microsecond)),
	)

	if err := store.AppendBatch(
		context.Background(),
		"Issue",
		aggID1,
		[]event.Event{evt1},
	); err != nil {
		t.Fatalf("AppendBatch 1: %v", err)
	}
	if err := store.AppendBatch(
		context.Background(),
		"Issue",
		aggID2,
		[]event.Event{evt2},
	); err != nil {
		t.Fatalf("AppendBatch 2: %v", err)
	}

	return evt1, evt2, aggID1, aggID2
}

func TestSQLiteEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testEventStore_SaveAndLoad(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()
	testEventStore_ConcurrencyConflict(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_AppendBatch(t *testing.T) {
	t.Parallel()
	testEventStore_AppendBatch(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()
	testEventStore_LoadFromVersion(t, newSQLiteTestStore(t), issueStoreConfig())
}

func TestSQLiteEventStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	aggID := id.NewAggregateID()

	_, err := store.Load(context.Background(), "Issue", aggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestSQLiteEventStore_LoadAll(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	_, _, _, _ = setupTwoTestEvents(t, store)

	all, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestSQLiteEventStore_ReadAll(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	_, _, _, _ = setupTwoTestEvents(t, store)

	all, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestSQLiteEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()
	testEventStore_MetadataRoundtrip(t, newSQLiteTestStore(t), issueStoreConfig(), "test")
}

package pebble

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func newPebbleTestStore(t *testing.T) *EventStore {
	t.Helper()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testEventStore_SaveAndLoad(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()
	testEventStore_ConcurrencyConflict(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestEventStore_AppendBatch(t *testing.T) {
	t.Parallel()
	testEventStore_AppendBatch(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()
	testEventStore_MetadataRoundtrip(t, newPebbleTestStore(t), issueStoreConfig(), "test")
}

func TestEventStore_Close(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	store, err := NewStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	// Close is a no-op (Backend owns the DB lifecycle).
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}
}

func TestEventStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	store, err := NewStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	aggID := id.NewAggregateID()

	evt := issueStoreConfig().NewTestEvent(t, aggID, 1)

	err = store.Save(
		context.Background(),
		id.NewAggregateRef("Issue", aggID),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}

	db2, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("reopen pebble: %v", err)
	}

	t.Cleanup(func() { _ = db2.Close() })

	store2, err := NewStore(db2, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store2.Load(context.Background(), id.NewAggregateRef("Issue", aggID))
	if err != nil {
		t.Fatalf("Load after reopen: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event after reopen, got %d", len(loaded))
	}

	if loaded[0].Type() != "IssueCreated" {
		t.Errorf("Type = %q, want IssueCreated", loaded[0].Type())
	}
}

func TestEventStore_Save_Mismatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		saveAggType  id.AggregateType
		saveAggID    id.AggregateID
		eventAggID   id.AggregateID
		eventVersion event.Version
	}{
		{
			name:         "aggregate_type",
			saveAggType:  "Project",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 1,
		},
		{
			name:         "aggregate_id",
			saveAggType:  "Issue",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 1,
		},
		{
			name:         "version",
			saveAggType:  "Issue",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newPebbleTestStore(t)
			evt := issueStoreConfig().NewTestEvent(t, tt.eventAggID, tt.eventVersion)

			err := store.Save(
				context.Background(),
				id.NewAggregateRef(tt.saveAggType, tt.saveAggID),
				[]event.Event{evt},
				event.Version(0),
			)
			if err == nil {
				t.Fatalf("expected error for %s mismatch", tt.name)
			}
		})
	}
}

func TestEventStore_NewStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewStore(nil, slog.Default())
	if !errors.Is(err, ErrNilDatabase) {
		t.Fatalf("expected ErrNilDatabase, got: %v", err)
	}
}

func TestEventStore_Save_EmptyEvents(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	err := store.Save(
		context.Background(),
		id.NewAggregateRef("Issue", aggID),
		nil,
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save with empty events should return nil, got %v", err)
	}
}

func TestEventStore_AppendBatch_EmptyEvents(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	err := store.AppendBatch(context.Background(), id.NewAggregateRef("Issue", aggID), nil)
	if err != nil {
		t.Fatalf("AppendBatch with empty events should return nil, got %v", err)
	}
}

func TestEventStore_Load_Empty(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	loaded, err := store.Load(context.Background(), id.NewAggregateRef("Issue", aggID))
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected 0 events for empty aggregate, got %d", len(loaded))
	}
}

func TestEventStore_WithAsyncWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewStore(db, slog.Default(), WithAsyncWrites())
	if err != nil {
		t.Fatal(err)
	}

	if store.syncWrites {
		t.Fatal("syncWrites should be false with WithAsyncWrites")
	}

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)
	evt := issueStoreConfig().NewTestEvent(t, aggID, 1)

	err = store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestEventStore_DefaultSyncWrites(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	if !store.syncWrites {
		t.Fatal("syncWrites should be true by default")
	}
}

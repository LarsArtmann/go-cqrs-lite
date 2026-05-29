package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestSQLiteSnapshotStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := newTestSnapshot(aggID, "Issue", 5, []byte(`{"title":"snapshot-issue"}`))

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), event.NewAggregateRef("Issue", aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	assertSnapshotVersion(t, loaded, 5)

	if string(loaded.State) != `{"title":"snapshot-issue"}` {
		t.Errorf("State = %s, want snapshot-issue", loaded.State)
	}
}

func TestSQLiteSnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	aggID := id.NewAggregateID()
	snap := newTestSnapshot(aggID, "Issue", 10, []byte(`{"title":"v10"}`))

	err = store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = store.LoadAtVersion(
		context.Background(),
		event.NewAggregateRef("Issue", aggID),
		event.Version(5),
	)
	if !errors.Is(err, event.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound for version 5 < snapshot version 10, got %v", err)
	}

	loaded, err := store.LoadAtVersion(
		context.Background(),
		event.NewAggregateRef("Issue", aggID),
		event.Version(15),
	)
	if err != nil {
		t.Fatalf("LoadAtVersion(15): %v", err)
	}

	assertSnapshotVersion(t, loaded, 10)
}

func TestSQLiteCheckpointStore_Roundtrip(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	store, err := NewSQLiteCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}

	loaded, err := store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("expected zero EventID for new projection, got %v", loaded)
	}

	eventID := id.NewEventID()

	err = store.Save(context.Background(), "issue_projection", eventID)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err = store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != eventID {
		t.Errorf("EventID = %v, want %v", loaded, eventID)
	}

	newEventID := id.NewEventID()

	err = store.Save(context.Background(), "issue_projection", newEventID)
	if err != nil {
		t.Fatalf("Save update: %v", err)
	}

	loaded, err = store.Load(context.Background(), "issue_projection")
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}

	if loaded != newEventID {
		t.Errorf("EventID = %v, want %v", loaded, newEventID)
	}
}

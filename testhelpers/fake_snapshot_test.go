package testhelpers

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestFakeSnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateType: "User",
		AggregateID:   aggID,
		Version:       event.Version(5),
		State:         []byte(`{"name":"Alice"}`),
	}

	err := store.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	saved := store.Saved()
	if len(saved) != 1 {
		t.Fatalf("len(Saved) = %d, want 1", len(saved))
	}

	store.SetSnapshot(&snap)

	loaded, err := store.Load(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), aggID),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != event.Version(5) {
		t.Errorf("Version = %d, want 5", loaded.Version)
	}
}

func TestFakeSnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()

	aggID := id.NewAggregateID()
	snap := event.Snapshot{
		AggregateType: "User",
		AggregateID:   aggID,
		Version:       event.Version(3),
		State:         []byte(`{}`),
	}

	store.SetSnapshot(&snap)

	loaded, err := store.LoadAtVersion(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		event.Version(3),
	)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if loaded.Version != event.Version(3) {
		t.Errorf("Version = %d, want 3", loaded.Version)
	}
}

func TestFakeSnapshotStore_LoadError(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()
	store.SetLoadError(errors.New("disk failure"))

	_, err := store.Load(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), id.NewAggregateID()),
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeSnapshotStore_SaveError(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()
	store.SetSaveError(errors.New("disk full"))

	err := store.Save(context.Background(), event.Snapshot{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeSnapshotStore_Close(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFakeSnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	store := NewFakeSnapshotStore()

	err := store.Delete(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), id.NewAggregateID()),
	)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

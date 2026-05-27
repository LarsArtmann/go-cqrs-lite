package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()
	inst := &saga.State{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusPending,
	}

	if err := store.Save(ctx, inst); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, inst.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.ID != inst.ID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, inst.ID)
	}
}

func TestMemoryStore_LoadNotFound(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Load(ctx, id.NewAggregateID())
	if !errors.Is(err, saga.ErrSagaNotFound) {
		t.Fatalf("expected ErrSagaNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadAllRunning(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()

	running := &saga.State{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusRunning,
	}
	if err := store.Save(ctx, running); err != nil {
		t.Fatalf("save running: %v", err)
	}

	completed := &saga.State{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusCompleted,
	}
	if err := store.Save(ctx, completed); err != nil {
		t.Fatalf("save completed: %v", err)
	}

	all, err := store.LoadAllRunning(ctx)
	if err != nil {
		t.Fatalf("load all running: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 running, got %d", len(all))
	}

	if all[0].ID != running.ID {
		t.Errorf("wrong instance: got %v, want %v", all[0].ID, running.ID)
	}
}

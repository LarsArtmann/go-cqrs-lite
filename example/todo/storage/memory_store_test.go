package storage_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
)

func TestMemoryStore_Operations(t *testing.T) {
	store, cleanup := storage.NewMemoryStore()
	defer cleanup()

	todo := &domain.Todo{
		ID: domain.NewTodoID(), Title: "Test", Description: "desc",
		Status: domain.StatusPending, Priority: 1, Tags: []string{"a"},
	}

	if err := store.Put(todo); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := store.Get(todo.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != "Test" {
		t.Errorf("Title = %q, want %q", got.Title, "Test")
	}

	todos, err := store.List(domain.TodoFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("List() = %d todos, want 1", len(todos))
	}

	count, err := store.Count(domain.TodoFilter{})
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}

	pending := domain.StatusPending
	filtered, err := store.List(domain.TodoFilter{Status: &pending})
	if err != nil {
		t.Fatalf("List(pending) error = %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("List(pending) = %d, want 1", len(filtered))
	}

	completed := domain.StatusCompleted
	filteredCompleted, err := store.List(domain.TodoFilter{Status: &completed})
	if err != nil {
		t.Fatalf("List(completed) error = %v", err)
	}
	if len(filteredCompleted) != 0 {
		t.Errorf("List(completed) = %d, want 0", len(filteredCompleted))
	}

	if err := store.Delete(todo.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(todo.ID); err == nil {
		t.Error("Get() should error after delete")
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	t.Parallel()

	store, cleanup := storage.NewMemoryStore()
	defer cleanup()

	_, err := store.Get(domain.NewTodoID())
	if err == nil {
		t.Error("Get() should error for non-existent ID")
	}
}

func TestMemoryStore_Isolation(t *testing.T) {
	t.Parallel()

	store, cleanup := storage.NewMemoryStore()
	defer cleanup()

	todo := &domain.Todo{
		ID: domain.NewTodoID(), Title: "Original",
		Status: domain.StatusPending, Tags: []string{},
	}
	_ = store.Put(todo)

	got, _ := store.Get(todo.ID)
	got.Title = "Modified"

	original, _ := store.Get(todo.ID)
	if original.Title != "Original" {
		t.Error("MemoryStore should return clones, not references")
	}
}

package projections_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
)

type mockReadModel struct {
	todos map[string]*domain.Todo
}

func newMockReadModel() *mockReadModel {
	return &mockReadModel{todos: make(map[string]*domain.Todo)}
}

func (m *mockReadModel) Get(todoID domain.TodoID) (*domain.Todo, error) {
	return m.todos[todoID.String()], nil
}

func (m *mockReadModel) List(_ domain.TodoFilter) ([]*domain.Todo, error) {
	var result []*domain.Todo
	for _, t := range m.todos {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockReadModel) Put(todo *domain.Todo) error {
	m.todos[todo.ID.String()] = todo
	return nil
}

func (m *mockReadModel) Delete(todoID domain.TodoID) error {
	delete(m.todos, todoID.String())
	return nil
}

func (m *mockReadModel) Count(_ domain.TodoFilter) (int, error) {
	return len(m.todos), nil
}

func makeCreatedEvent(
	t *testing.T,
	aggID id.AggregateID,
	payload aggregate.TodoPayload,
) event.Event {
	t.Helper()
	var c codec.JSONCodec
	data, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	evt, err := event.NewEvent(aggregate.EventCreated, aggID, aggregate.AggregateType, 1, data)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	return evt
}

func TestTodoProjection_HandleCreated(t *testing.T) {
	t.Parallel()

	store := newMockReadModel()
	proj := projections.NewTodoProjection(store)
	aggID := id.NewAggregateID()

	payload := aggregate.TodoPayload{
		Title: "Test Todo", Description: "test desc",
		Status: domain.StatusPending, Priority: 1, Tags: []string{"a"},
	}
	evt := makeCreatedEvent(t, aggID, payload)

	err := proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	todo, ok := store.todos[aggID.String()]
	if !ok {
		t.Fatal("todo not stored")
	}
	if todo.Title != "Test Todo" {
		t.Errorf("Title = %q, want %q", todo.Title, "Test Todo")
	}
	if todo.Status != domain.StatusPending {
		t.Errorf("Status = %q, want %q", todo.Status, domain.StatusPending)
	}
}

func TestTodoProjection_HandleDeleted(t *testing.T) {
	t.Parallel()

	store := newMockReadModel()
	proj := projections.NewTodoProjection(store)
	aggID := id.NewAggregateID()

	todoID, _ := domain.ParseTodoID(aggID.String())
	store.todos[aggID.String()] = &domain.Todo{ID: todoID, Title: "To Delete"}

	evt, err := event.NewEvent(aggregate.EventDeleted, aggID, aggregate.AggregateType, 2, nil)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	err = proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if _, exists := store.todos[aggID.String()]; exists {
		t.Error("todo should be deleted from store")
	}
}

func TestTodoProjection_UnknownEventType(t *testing.T) {
	t.Parallel()

	store := newMockReadModel()
	proj := projections.NewTodoProjection(store)

	evt, err := event.NewEvent("todo.unknown", id.NewAggregateID(), aggregate.AggregateType, 1, nil)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	err = proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle() for unknown type should return nil, got %v", err)
	}
}

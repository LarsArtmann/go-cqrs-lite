package aggregate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

func TestTodoState_ToDomain(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)
	state := foldAll(t, events)

	todoID, _ := domain.ParseTodoID(aggID.String())
	todo := state.ToDomain(todoID, int64(len(events)))

	if todo.ID != todoID {
		t.Errorf("ID = %v, want %v", todo.ID, todoID)
	}
	if todo.Title != "Title" {
		t.Errorf("Title = %q, want %q", todo.Title, "Title")
	}
	if todo.Status != domain.StatusCompleted {
		t.Errorf("Status = %v, want %v", todo.Status, domain.StatusCompleted)
	}
	if todo.Version != 2 {
		t.Errorf("Version = %d, want 2", todo.Version)
	}
	if todo.CompletedAt == nil {
		t.Error("CompletedAt is nil, want non-nil")
	}
}

func TestTodoState_IsNew(t *testing.T) {
	t.Parallel()
	if !aggregate.InitialState.IsNew() {
		t.Error("InitialState.IsNew() = false, want true")
	}

	aggID := testAggID()
	events := mustDecide(
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
	state := foldAll(t, events)
	if state.IsNew() {
		t.Error("state after create.IsNew() = true, want false")
	}
}

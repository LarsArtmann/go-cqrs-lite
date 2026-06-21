package aggregate_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

func TestDecideCreate_Success(t *testing.T) {
	t.Parallel()
	events := mustDecide(
		t,
		aggregate.DecideCreate(
			testAggID(),
			domain.Title("Title"),
			domain.Description("desc"),
			domain.Priority(1),
			nil,
		),
	)

	if len(events) != 1 {
		t.Fatalf("events count = %d, want 1", len(events))
	}
	if events[0].Type() != aggregate.EventCreated {
		t.Errorf("Type = %v, want %v", events[0].Type(), aggregate.EventCreated)
	}
}

func TestDecideCreate_EmptyTitle(t *testing.T) {
	t.Parallel()
	_, err := invoke(
		aggregate.DecideCreate(
			testAggID(),
			domain.Title(""),
			domain.Description("desc"),
			domain.Priority(1),
			nil,
		),
	)
	if !errors.Is(err, domain.ErrEmptyTitle) {
		t.Errorf("error = %v, want ErrEmptyTitle", err)
	}
}

func TestDecideCreate_AlreadyExists(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	created := mustDecide(
		t,
		aggregate.DecideCreate(
			aggID,
			domain.Title("First"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
	state := foldAll(t, created)

	_, err := invoke(
		aggregate.DecideCreate(
			aggID,
			domain.Title("Second"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
		withState(state),
		withVersion(1),
	)
	if !errors.Is(err, aggregate.ErrTodoAlreadyExists) {
		t.Errorf("error = %v, want ErrTodoAlreadyExists", err)
	}
}

func TestDecideUpdate_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideUpdate(aggID, domain.Title("Updated"), domain.Description("new desc")),
	)

	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[0].Type() != aggregate.EventCreated {
		t.Errorf("events[0] Type = %v, want %v", events[0].Type(), aggregate.EventCreated)
	}
	if events[1].Type() != aggregate.EventUpdated {
		t.Errorf("events[1] Type = %v, want %v", events[1].Type(), aggregate.EventUpdated)
	}
}

func TestDecideUpdate_EmptyTitle(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	created := mustDecide(
		t,
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
	state := foldAll(t, created)

	_, err := invoke(
		aggregate.DecideUpdate(aggID, domain.Title(""), domain.Description("desc")),
		withState(state),
		withVersion(1),
	)
	if !errors.Is(err, domain.ErrEmptyTitle) {
		t.Errorf("error = %v, want ErrEmptyTitle", err)
	}
}

func TestDecideUpdate_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(
		aggregate.DecideUpdate(aggID, domain.Title("New"), domain.Description("desc")),
		withState(state),
		withVersion(2),
	)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDecideDelete_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideDelete(aggID))

	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[1].Type() != aggregate.EventDeleted {
		t.Errorf("events[1] Type = %v, want %v", events[1].Type(), aggregate.EventDeleted)
	}
}

func TestDecideDelete_AlreadyDeleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(aggregate.DecideDelete(aggID), withState(state), withVersion(2))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDecideDelete_NotExists(t *testing.T) {
	t.Parallel()
	_, err := invoke(aggregate.DecideDelete(testAggID()))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDecideChangeStatus_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusInProgress),
	)

	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[1].Type() != aggregate.EventStatusChanged {
		t.Errorf("events[1] Type = %v, want %v", events[1].Type(), aggregate.EventStatusChanged)
	}
}

func TestDecideChangeStatus_Completed(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)

	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[1].Type() != aggregate.EventCompleted {
		t.Errorf("events[1] Type = %v, want %v", events[1].Type(), aggregate.EventCompleted)
	}
}

func TestDecideChangeStatus_Invalid(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	created := mustDecide(
		t,
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
	state := foldAll(t, created)

	_, err := invoke(
		aggregate.DecideChangeStatus(aggID, domain.TodoStatus("invalid")),
		withState(state),
		withVersion(1),
	)
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("error = %v, want ErrInvalidStatus", err)
	}
}

func TestDecideChangeStatus_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
		withState(state),
		withVersion(2),
	)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

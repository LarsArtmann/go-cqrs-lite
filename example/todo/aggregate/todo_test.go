package aggregate_test

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func testAggID() id.AggregateID { return id.NewAggregateID() }

func TestFold_Created(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	now := time.Now().UTC()
	events := mustDecide(aggregate.DecideCreate(aggID, "Title", "desc", 1, []string{"tag"}))

	state, err := aggregate.Fold(aggregate.InitialState, events[0])
	if err != nil {
		t.Fatalf("Fold: %v", err)
	}

	if state.Title != "Title" {
		t.Errorf("Title = %q, want %q", state.Title, "Title")
	}
	if state.Description != "desc" {
		t.Errorf("Description = %q, want %q", state.Description, "desc")
	}
	eventtest.AssertEqual(t, state.Status, domain.StatusPending, "Status")
	if state.Priority != 1 {
		t.Errorf("Priority = %d, want 1", state.Priority)
	}
	if len(state.Tags) != 1 || state.Tags[0] != "tag" {
		t.Errorf("Tags = %v, want [tag]", state.Tags)
	}
	if now.Sub(state.CreatedAt) > time.Second {
		t.Errorf("CreatedAt = %v, want near %v", state.CreatedAt, now)
	}
	if state.Deleted {
		t.Error("Deleted = true, want false")
	}
	if !state.CreatedAt.Equal(state.UpdatedAt) {
		t.Errorf("CreatedAt != UpdatedAt: %v vs %v", state.CreatedAt, state.UpdatedAt)
	}
}

func TestFold_Updated(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideUpdate(aggID, "New Title", "new desc"))

	state := foldAll(t, events)

	if state.Title != "New Title" {
		t.Errorf("Title = %q, want %q", state.Title, "New Title")
	}
	if state.Description != "new desc" {
		t.Errorf("Description = %q, want %q", state.Description, "new desc")
	}
	eventtest.AssertEqual(t, state.Status, domain.StatusPending, "Status")
	if state.Deleted {
		t.Error("Deleted = true, want false")
	}
}

func TestFold_StatusChanged(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusInProgress),
	)

	state := foldAll(t, events)

	eventtest.AssertEqual(t, state.Status, domain.StatusInProgress, "Status")
}

func TestFold_Completed(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)

	state := foldAll(t, events)

	eventtest.AssertEqual(t, state.Status, domain.StatusCompleted, "Status")
	if state.CompletedAt == nil {
		t.Fatal("CompletedAt is nil, want non-nil")
	}
	if time.Now().UTC().Sub(*state.CompletedAt) > time.Second {
		t.Errorf("CompletedAt = %v, want near now", *state.CompletedAt)
	}
}

func TestFold_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideDelete(aggID))

	state := foldAll(t, events)

	if !state.Deleted {
		t.Error("Deleted = false, want true")
	}
}

func TestDecideCreate_Success(t *testing.T) {
	t.Parallel()
	events := mustDecide(aggregate.DecideCreate(testAggID(), "Title", "desc", 1, nil))

	if len(events) != 1 {
		t.Fatalf("events count = %d, want 1", len(events))
	}
	if events[0].Type() != aggregate.EventCreated {
		t.Errorf("Type = %v, want %v", events[0].Type(), aggregate.EventCreated)
	}
}

func TestDecideCreate_EmptyTitle(t *testing.T) {
	t.Parallel()
	_, err := invoke(aggregate.DecideCreate(testAggID(), "", "desc", 1, nil))
	if !errors.Is(err, domain.ErrEmptyTitle) {
		t.Errorf("error = %v, want ErrEmptyTitle", err)
	}
}

func TestDecideCreate_AlreadyExists(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	created := mustDecide(aggregate.DecideCreate(aggID, "First", "", 1, nil))
	state := foldAll(t, created)

	_, err := invoke(
		aggregate.DecideCreate(aggID, "Second", "", 1, nil),
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
	events := createThenDecide(t, aggID, aggregate.DecideUpdate(aggID, "Updated", "new desc"))

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
	created := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, created)

	_, err := invoke(aggregate.DecideUpdate(aggID, "", "desc"), withState(state), withVersion(1))
	if !errors.Is(err, domain.ErrEmptyTitle) {
		t.Errorf("error = %v, want ErrEmptyTitle", err)
	}
}

func TestDecideUpdate_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(aggregate.DecideUpdate(aggID, "New", "desc"), withState(state), withVersion(2))
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
	created := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
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
	events := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, events)
	if state.IsNew() {
		t.Error("state after create.IsNew() = true, want false")
	}
}

func TestFullLifecycle(t *testing.T) {
	t.Parallel()
	aggID := testAggID()

	created := mustDecide(
		aggregate.DecideCreate(aggID, "Buy milk", "from store", 2, []string{"errands"}),
	)
	state := foldAll(t, created)
	if state.Title != "Buy milk" {
		t.Errorf("Title = %q, want %q", state.Title, "Buy milk")
	}
	if state.Deleted {
		t.Error("Deleted = true, want false")
	}

	updated := mustDecideFrom(t, state, 1, aggregate.DecideUpdate(aggID, "Buy oat milk", "organic"))
	state = foldAllFrom(t, state, updated)
	if state.Title != "Buy oat milk" {
		t.Errorf("Title = %q, want %q", state.Title, "Buy oat milk")
	}
	if state.Description != "organic" {
		t.Errorf("Description = %q, want %q", state.Description, "organic")
	}

	statusChanged := mustDecideFrom(
		t,
		state,
		2,
		aggregate.DecideChangeStatus(aggID, domain.StatusInProgress),
	)
	state = foldAllFrom(t, state, statusChanged)
	eventtest.AssertEqual(t, state.Status, domain.StatusInProgress, "Status")

	completed := mustDecideFrom(
		t,
		state,
		3,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)
	state = foldAllFrom(t, state, completed)
	eventtest.AssertEqual(t, state.Status, domain.StatusCompleted, "Status")
	if state.CompletedAt == nil {
		t.Error("CompletedAt is nil, want non-nil")
	}

	deleted := mustDecideFrom(t, state, 4, aggregate.DecideDelete(aggID))
	state = foldAllFrom(t, state, deleted)
	if !state.Deleted {
		t.Error("Deleted = false, want true")
	}
}

func mustDecide(f decider.DecideFunc[aggregate.TodoState], opts ...decideOption) []event.Event {
	events, err := invoke(f, opts...)
	if err != nil {
		panic(err)
	}

	return events
}

func invoke(
	f decider.DecideFunc[aggregate.TodoState],
	opts ...decideOption,
) ([]event.Event, error) {
	o := &decideOpts{state: aggregate.InitialState, version: 0}
	for _, opt := range opts {
		opt(o)
	}

	return f(o.state, o.version)
}

type decideOpts struct {
	state   aggregate.TodoState
	version event.Version
}

type decideOption func(*decideOpts)

func withState(s aggregate.TodoState) decideOption {
	return func(o *decideOpts) { o.state = s }
}

func withVersion(v event.Version) decideOption {
	return func(o *decideOpts) { o.version = v }
}

func mustDecideFrom(
	t *testing.T,
	state aggregate.TodoState,
	version int,
	f decider.DecideFunc[aggregate.TodoState],
) []event.Event {
	t.Helper()
	events, err := f(state, event.Version(version))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return events
}

func createThenDecide(
	t *testing.T,
	aggID id.AggregateID,
	next decider.DecideFunc[aggregate.TodoState],
) []event.Event {
	t.Helper()
	created := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, created)
	nextEvents := mustDecideFrom(t, state, 1, next)

	return append(created, nextEvents...)
}

func foldAll(t *testing.T, events []event.Event) aggregate.TodoState {
	t.Helper()

	return foldAllFrom(t, aggregate.InitialState, events)
}

func foldAllFrom(
	t *testing.T,
	state aggregate.TodoState,
	events []event.Event,
) aggregate.TodoState {
	t.Helper()
	for _, evt := range events {
		var err error
		state, err = aggregate.Fold(state, evt)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	return state
}

func createDeleteState(t *testing.T, aggID id.AggregateID) aggregate.TodoState {
	t.Helper()
	created := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, created)
	deleted := mustDecideFrom(t, state, 1, aggregate.DecideDelete(aggID))

	return foldAllFrom(t, state, deleted)
}

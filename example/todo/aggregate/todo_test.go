package aggregate_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAggID() id.AggregateID { return id.NewAggregateID() }

func TestFold_Created(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	now := time.Now().UTC()
	events := mustDecide(aggregate.DecideCreate(aggID, "Title", "desc", 1, []string{"tag"}))

	state, err := aggregate.Fold(aggregate.InitialState, events[0])
	require.NoError(t, err)

	assert.Equal(t, "Title", state.Title)
	assert.Equal(t, "desc", state.Description)
	assert.Equal(t, domain.StatusPending, state.Status)
	assert.Equal(t, 1, state.Priority)
	assert.Equal(t, []string{"tag"}, state.Tags)
	assert.WithinDuration(t, now, state.CreatedAt, time.Second)
	assert.False(t, state.Deleted)
	assert.True(t, state.CreatedAt.Equal(state.UpdatedAt))
}

func TestFold_Updated(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideUpdate(aggID, "New Title", "new desc"))

	state := foldAll(t, events)

	assert.Equal(t, "New Title", state.Title)
	assert.Equal(t, "new desc", state.Description)
	assert.Equal(t, domain.StatusPending, state.Status)
	assert.False(t, state.Deleted)
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

	assert.Equal(t, domain.StatusInProgress, state.Status)
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

	assert.Equal(t, domain.StatusCompleted, state.Status)
	require.NotNil(t, state.CompletedAt)
	assert.WithinDuration(t, time.Now().UTC(), *state.CompletedAt, time.Second)
}

func TestFold_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideDelete(aggID))

	state := foldAll(t, events)

	assert.True(t, state.Deleted)
}

func TestDecideCreate_Success(t *testing.T) {
	t.Parallel()
	events := mustDecide(aggregate.DecideCreate(testAggID(), "Title", "desc", 1, nil))

	require.Len(t, events, 1)
	assert.Equal(t, aggregate.EventCreated, events[0].Type())
}

func TestDecideCreate_EmptyTitle(t *testing.T) {
	t.Parallel()
	_, err := invoke(aggregate.DecideCreate(testAggID(), "", "desc", 1, nil))
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
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
	assert.ErrorIs(t, err, aggregate.ErrTodoAlreadyExists)
}

func TestDecideUpdate_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideUpdate(aggID, "Updated", "new desc"))

	require.Len(t, events, 2)
	assert.Equal(t, aggregate.EventCreated, events[0].Type())
	assert.Equal(t, aggregate.EventUpdated, events[1].Type())
}

func TestDecideUpdate_EmptyTitle(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	created := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, created)

	_, err := invoke(aggregate.DecideUpdate(aggID, "", "desc"), withState(state), withVersion(1))
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestDecideUpdate_Deleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(aggregate.DecideUpdate(aggID, "New", "desc"), withState(state), withVersion(2))
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDecideDelete_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(t, aggID, aggregate.DecideDelete(aggID))

	require.Len(t, events, 2)
	assert.Equal(t, aggregate.EventDeleted, events[1].Type())
}

func TestDecideDelete_AlreadyDeleted(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	state := createDeleteState(t, aggID)

	_, err := invoke(aggregate.DecideDelete(aggID), withState(state), withVersion(2))
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDecideDelete_NotExists(t *testing.T) {
	t.Parallel()
	_, err := invoke(aggregate.DecideDelete(testAggID()))
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDecideChangeStatus_Success(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusInProgress),
	)

	require.Len(t, events, 2)
	assert.Equal(t, aggregate.EventStatusChanged, events[1].Type())
}

func TestDecideChangeStatus_Completed(t *testing.T) {
	t.Parallel()
	aggID := testAggID()
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)

	require.Len(t, events, 2)
	assert.Equal(t, aggregate.EventCompleted, events[1].Type())
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
	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
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
	assert.ErrorIs(t, err, domain.ErrNotFound)
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

	assert.Equal(t, todoID, todo.ID)
	assert.Equal(t, "Title", todo.Title)
	assert.Equal(t, domain.StatusCompleted, todo.Status)
	assert.Equal(t, int64(2), todo.Version)
	assert.NotNil(t, todo.CompletedAt)
}

func TestTodoState_IsNew(t *testing.T) {
	t.Parallel()
	assert.True(t, aggregate.InitialState.IsNew())

	aggID := testAggID()
	events := mustDecide(aggregate.DecideCreate(aggID, "Title", "", 1, nil))
	state := foldAll(t, events)
	assert.False(t, state.IsNew())
}

func TestFullLifecycle(t *testing.T) {
	t.Parallel()
	aggID := testAggID()

	created := mustDecide(
		aggregate.DecideCreate(aggID, "Buy milk", "from store", 2, []string{"errands"}),
	)
	state := foldAll(t, created)
	assert.Equal(t, "Buy milk", state.Title)
	assert.False(t, state.Deleted)

	updated := mustDecideFrom(t, state, 1, aggregate.DecideUpdate(aggID, "Buy oat milk", "organic"))
	state = foldAllFrom(t, state, updated)
	assert.Equal(t, "Buy oat milk", state.Title)
	assert.Equal(t, "organic", state.Description)

	statusChanged := mustDecideFrom(
		t,
		state,
		2,
		aggregate.DecideChangeStatus(aggID, domain.StatusInProgress),
	)
	state = foldAllFrom(t, state, statusChanged)
	assert.Equal(t, domain.StatusInProgress, state.Status)

	completed := mustDecideFrom(
		t,
		state,
		3,
		aggregate.DecideChangeStatus(aggID, domain.StatusCompleted),
	)
	state = foldAllFrom(t, state, completed)
	assert.Equal(t, domain.StatusCompleted, state.Status)
	assert.NotNil(t, state.CompletedAt)

	deleted := mustDecideFrom(t, state, 4, aggregate.DecideDelete(aggID))
	state = foldAllFrom(t, state, deleted)
	assert.True(t, state.Deleted)
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
	require.NoError(t, err)
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
		require.NoError(t, err)
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

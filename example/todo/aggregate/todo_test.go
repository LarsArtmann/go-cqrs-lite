package aggregate_test

import (
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
	events := mustDecide(
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description("desc"),
			domain.Priority(1),
			[]string{"tag"},
		),
	)

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
	events := createThenDecide(
		t,
		aggID,
		aggregate.DecideUpdate(aggID, domain.Title("New Title"), domain.Description("new desc")),
	)

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
	created := mustDecide(
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
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
	created := mustDecide(
		aggregate.DecideCreate(
			aggID,
			domain.Title("Title"),
			domain.Description(""),
			domain.Priority(1),
			nil,
		),
	)
	state := foldAll(t, created)
	deleted := mustDecideFrom(t, state, 1, aggregate.DecideDelete(aggID))

	return foldAllFrom(t, state, deleted)
}

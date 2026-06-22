package aggregate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

func TestFullLifecycle(t *testing.T) {
	t.Parallel()
	aggID := testAggID()

	created := mustDecide(
		t,
		aggregate.DecideCreate(
			aggID,
			domain.Title("Buy milk"),
			domain.Description("from store"),
			domain.Priority(2),
			[]string{"errands"},
		),
	)
	state := foldAll(t, created)
	if state.Title != "Buy milk" {
		t.Errorf("Title = %q, want %q", state.Title, "Buy milk")
	}
	if state.Deleted {
		t.Error("Deleted = true, want false")
	}

	updated := mustDecideFrom(
		t,
		state,
		1,
		aggregate.DecideUpdate(aggID, domain.Title("Buy oat milk"), domain.Description("organic")),
	)
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

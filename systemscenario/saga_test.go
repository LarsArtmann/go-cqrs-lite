package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// newSagaScenario boots the archiver-saga domain: a deriver reacts to
// task.updated by dispatching task.archive (which emits task.archived).
func newSagaScenario(t *testing.T) (*systemscenario.Scenario, id.StreamRef, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, sagaDomain(), memoryDeployment())

	return sc, ref, ctx
}

// TestSaga_DeriverEventToCommandChain is the first end-to-end saga story:
// When(complete) emits task.updated → the deriver (bus subscriber) derives
// task.archive → its handler emits task.archived. Await polls the
// asynchronous bus delivery for both the journal trail and the captured
// command chain.
func TestSaga_DeriverEventToCommandChain(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newSagaScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Await().
		Then("task.updated", "task.archived").
		ThenCommands("task.complete", "task.archive")
}

// TestSystem_CombinedPhases chains every phase kind in one scenario:
// given-by-command → external-event act → read-model assertion.
func TestSystem_CombinedPhases(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newTaskScenario(t)

	sc.Given().Command(newTaskCmd("task.create", ref.ID)).
		WhenEvent(
			sc.Event("task.updated", ref, TaskUpdated{ID: ref.ID.String(), Title: "ship it", Status: "completed"}),
		).Then("task.updated").
		ThenQuery(taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "ship it", Status: "completed"})
}

// TestSystem_GivenCommandsRunInOrder proves dispatch-based givens compose:
// create-then-rename seeds a renamed task; the When act renames again and
// its event version proves both givens landed in order.
func TestSystem_GivenCommandsRunInOrder(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given().
		Command(newTaskCmd("task.create", ref.ID)).
		Command(newTaskCmd("task.rename", ref.ID)).
		When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenEvents(func(events []event.Event) {
			// created v1, renamed v2 → the act's completion must be v3.
			if got := events[0].Version(); got != 3 {
				t.Fatalf("want completion at v3 (two ordered givens), got v%d", got)
			}
		})
}

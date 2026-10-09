package systemscenario_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// TestSystem_WhenEventDrivesBusPath exercises the bus act: a published event
// (no journal append) still flows to projections, so the read model reflects
// it via ThenQuery.
func TestSystem_WhenEventDrivesBusPath(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "bus task", Status: "pending"}),
	).WhenEvent(
		sc.Event("task.completed", ref, TaskCompleted{ID: ref.ID.String(), Status: "completed"}),
	).ThenNoEvents(). // bus acts do not journal the published events themselves
				ThenQuery(taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "bus task", Status: "completed"})
}

// TestSystem_ThenCommandsCapturesActs asserts the always-on command capture:
// commands dispatched by the When act are diffed against the baseline, so
// given-phase commands (seed-by-intent) do not leak into the assertion.
func TestSystem_ThenCommandsCapturesActs(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given().Command(newTaskCmd("task.create", ref.ID)).
		When(newTaskCmd("task.rename", ref.ID)).
		Then("task.updated").
		ThenCommands("task.rename").
		ThenCommandsSatisfy(func(cmds []command.Command) {
			if len(cmds) != 1 {
				t.Fatalf("ThenCommandsSatisfy: want 1 act command, got %d", len(cmds))
			}

			if got := cmds[0].StreamID(); got != ref.ID {
				t.Errorf("ThenCommandsSatisfy: want stream %s, got %s", ref.ID, got)
			}
		})
}

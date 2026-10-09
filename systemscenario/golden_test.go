package systemscenario_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestSystem_ThenGoldenTrail(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "golden", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenGolden(t, "golden_task_lifecycle")
}

func TestTrailIsDeterministicShape(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		ThenEvents(func(events []event.Event) {
			want := "- task.updated v2 actor=harness-fixture\n"
			if got := systemscenario.Trail(events); got != want {
				t.Fatalf("Trail:\nwant %q\ngot  %q", want, got)
			}
		})
}

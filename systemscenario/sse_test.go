package systemscenario_test

// SSE-leg self-tests: the SubscribeSSE helper streams a projection
// collection over a real HTTP round trip, so an assertion here pins the
// same wire path a browser EventSource consumer uses.

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestSSE_TaskViewStreamsOverHTTP(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())
	ref := id.NewStreamRef("Task", id.NewStreamID())

	// Subscribe before the acts: notifications fire on projection writes,
	// so a late subscriber would miss the created-then-renamed sequence.
	sub := systemscenario.SubscribeSSE[TaskView](t, ctx, sc, "task_views")

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"},
		),
	).When(newTaskCmd("task.rename", ref.ID)).
		ThenQuery(
			taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "renamed", Status: "pending"},
		)

	sub.Await(t, 5*time.Second, "renamed task view", func(v TaskView) bool {
		return v.Title == "renamed" && v.Status == "pending"
	})

	// The stream is ordered, so the pre-rename create view arrived before
	// the rename view the Await matched — both folds streamed.
	received := sub.Received()

	if len(received) < 2 {
		t.Fatalf("expected the create and rename views to stream, received %d: %+v",
			len(received), received)
	}
}

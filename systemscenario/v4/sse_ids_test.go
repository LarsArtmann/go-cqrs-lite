package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// TestSSEStream_ReceivedIDs proves ReceivedIDs over the real wire path: one
// entry per received value, in arrival order, and all empty because
// SubscribeSSE serves a watcher without a replay journal (ServeSSE stamps ids
// only on the reconnection path).
func TestSSEStream_ReceivedIDs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())
	ref := id.NewStreamRef("Task", id.NewStreamID())

	sub := systemscenario.SubscribeSSE[TaskView](t, ctx, sc, "task_views")

	if ids := sub.ReceivedIDs(); len(ids) != 0 {
		t.Fatalf("ReceivedIDs before any event = %v, want empty", ids)
	}

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

	awaitBudget := 5 * time.Second
	if systemscenario.RaceEnabled {
		awaitBudget = 30 * time.Second
	}

	sub.Await(t, awaitBudget, "renamed task view", func(v TaskView) bool {
		return v.Title == "renamed"
	})

	ids := sub.ReceivedIDs()
	received := sub.Received()

	if len(ids) != len(received) || len(ids) < 2 {
		t.Fatalf("ReceivedIDs has %d entries for %d received values, want one each (at least 2)",
			len(ids), len(received))
	}

	for i, id := range ids {
		if id != "" {
			t.Fatalf("ReceivedIDs[%d] = %q, want empty: the subscribed watcher has no replay journal", i, id)
		}
	}
}

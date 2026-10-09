package systemscenario_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// newTaskScenario boots the fixture domain on memory engines.
func newTaskScenario(t *testing.T) (*systemscenario.Scenario, id.StreamRef, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())

	return sc, ref, ctx
}

// taskViewQuery closes over the booted system for ThenQuery lookups.
func taskViewQuery(
	sc *systemscenario.Scenario,
	ctx context.Context,
	taskID string,
) func() (any, error) {
	return func() (any, error) {
		return metaengine.ExecuteTyped[system.LookupInput[string], TaskView](
			ctx, sc.System().MetaEngine(), system.LookupInput[string]{ID: taskID})
	}
}

func TestSystem_HappyPath(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated")
}

func TestSystem_ThenPayloadAssertsDecodedPayload(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	phase := sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID))

	phase.Then("task.updated")
	systemscenario.ThenPayload(phase, 0, TaskUpdated{ID: ref.ID.String(), Title: "ship it", Status: "completed"})
}

func TestSystem_GivenSeedsDeciderState(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	// No given history: the decider rejects completing a task that never existed.
	sc.When(newTaskCmd("task.complete", ref.ID)).
		ThenError(errTaskMissing).
		ThenNoEvents().
		ThenErrorFamily(errorfamily.Rejection)
}

func TestSystem_ConflictFamilyOnCompletedTask(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
		sc.Event("task.updated", ref, TaskUpdated{ID: ref.ID.String(), Title: "ship it", Status: "completed"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		ThenError(errTaskCompleted).
		ThenErrorFamily(errorfamily.Conflict)
}

func TestSystem_ThenQueryAwaitsProjection(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenQuery(taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "ship it", Status: "completed"})
}

func TestSystem_GivenByCommandSeedsByIntent(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given().Command(newTaskCmd("task.create", ref.ID)).
		When(newTaskCmd("task.rename", ref.ID)).
		Then("task.updated")
}

func TestSystem_ThenEventsAndMetadata(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenEvents(func(events []event.Event) {
			if len(events) != 1 {
				t.Fatalf("ThenEvents: want 1 act event, got %d", len(events))
			}

			if got := events[0].Version(); got != 2 {
				t.Errorf("ThenEvents: want version 2 (created was v1), got %d", got)
			}

			if got := events[0].StreamID(); got != ref.ID {
				t.Errorf("ThenEvents: want stream %s, got %s", ref.ID, got)
			}
		}).
		ThenMetadata(0, func(md event.Metadata) error {
			if md.ActorID != completerActor {
				return fmt.Errorf("want actor %s on task.completed, got %s", completerActor, md.ActorID)
			}

			return nil
		})
}

func TestSystem_WhenQueryActThenResultAndSuccess(t *testing.T) {
	t.Parallel()

	sc, _, _ := newTaskScenario(t)

	sc.WhenQuery(EchoQuery{Value: "hello"}).
		ThenSuccess().
		ThenResult("hello")
}

func TestSystem_WhenQueryUnknownHandlerThenError(t *testing.T) {
	t.Parallel()

	sc, _, _ := newTaskScenario(t)

	sc.WhenQuery(unknownQuery{}).
		ThenError(query.ErrHandlerNotFound)
}

// unknownQuery targets no registered handler.
type unknownQuery struct{}

func (unknownQuery) Type() query.Type { return "task.unknown" }

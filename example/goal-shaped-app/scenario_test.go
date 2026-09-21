package main

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// The task flow's write-side invariants as Given/When/Then scenarios — pure
// decider tests: no engine, no projection host, no config. This is the
// testing-story demo: the SAME Domain() that main.go composes is verified
// here one decision at a time.

// createScenario decides a create against version 0 (empty stream).
func createScenario(cmd CreateTaskCmd) scenario.DecideFunc[CreateTaskCmd, TaskState] {
	return func(s TaskState, c CreateTaskCmd) ([]event.Event, error) {
		return decideCreate(c, s, event.Version(0))
	}
}

// completeScenario decides a complete against the given current version.
func completeScenario(v event.Version) scenario.DecideFunc[CompleteTaskCmd, TaskState] {
	return func(s TaskState, c CompleteTaskCmd) ([]event.Event, error) {
		return decideComplete(c, s, v)
	}
}

// deleteScenario decides a delete against the given current version.
func deleteScenario(v event.Version) scenario.DecideFunc[DeleteTaskCmd, TaskState] {
	return func(s TaskState, c DeleteTaskCmd) ([]event.Event, error) {
		return decideDelete(c, s, v)
	}
}

func scenarioCmd(t *testing.T) *command.BasicCommand {
	t.Helper()

	basic, err := command.New(cmdCreateTask, id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return basic
}

// givenCreated folds nothing but hands the scenario a persisted Created fact
// at version 1, the state every "open task" scenario starts from.
func givenCreated(t *testing.T, title string, priority int) []event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	evt, err := event.New(evtTaskCreated, streamID, streamType, event.Version(1),
		TaskCreated{ID: streamID.String(), Title: title, Priority: priority})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return []event.Event{evt}
}

func TestScenario_CreateOnEmptyStream(t *testing.T) {
	t.Parallel()

	cmd := CreateTaskCmd{BasicCommand: scenarioCmd(t), Title: "Write scenario", Priority: PriorityNormal}

	scenario.Given[CreateTaskCmd](t, applyTask, TaskState{}).
		When(cmd, createScenario(cmd)).
		ThenState(applyTask, TaskState{}, TaskState{
			Exists: true, Title: "Write scenario", Priority: PriorityNormal,
		})
}

func TestScenario_CreateTwiceIsRejected(t *testing.T) {
	t.Parallel()

	cmd := CreateTaskCmd{BasicCommand: scenarioCmd(t), Title: "Duplicate", Priority: PriorityNormal}

	scenario.Given[CreateTaskCmd](t, applyTask, TaskState{}, givenCreated(t, "Duplicate", PriorityNormal)...).
		When(cmd, createScenario(cmd)).
		ThenError(errTaskExists)
}

func TestScenario_CompleteOpenTask(t *testing.T) {
	t.Parallel()

	cmd := CompleteTaskCmd{BasicCommand: scenarioCmd(t)}

	scenario.Given[CompleteTaskCmd](t, applyTask, TaskState{}, givenCreated(t, "Finish me", PriorityHigh)...).
		When(cmd, completeScenario(event.Version(1))).
		ThenState(applyTask, TaskState{}, TaskState{
			Exists: true, Done: true, Title: "Finish me", Priority: PriorityHigh,
		})
}

func TestScenario_CompleteTwiceIsRejected(t *testing.T) {
	t.Parallel()

	cmd := CompleteTaskCmd{BasicCommand: scenarioCmd(t)}

	scenario.Given[CompleteTaskCmd](t, applyTask, TaskState{}, givenCreated(t, "Done once", PriorityNormal)...).
		When(cmd, completeScenario(event.Version(1))).
		ThenError(errTaskDone)
}

func TestScenario_DeleteOpenTask(t *testing.T) {
	t.Parallel()

	cmd := DeleteTaskCmd{BasicCommand: scenarioCmd(t)}

	scenario.Given[DeleteTaskCmd](t, applyTask, TaskState{}, givenCreated(t, "Ephemeral", PriorityNormal)...).
		When(cmd, deleteScenario(event.Version(1))).
		Then(evtTaskDeleted)
}

func TestScenario_DeleteUnknownTaskIsRejected(t *testing.T) {
	t.Parallel()

	cmd := DeleteTaskCmd{BasicCommand: scenarioCmd(t)}

	scenario.Given[DeleteTaskCmd](t, applyTask, TaskState{}).
		When(cmd, deleteScenario(event.Version(0))).
		ThenError(errTaskGone)
}

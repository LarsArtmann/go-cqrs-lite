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
// testing-story demo: the SAME decisions Domain() registers are verified
// here one decision at a time, against the same fold.

func scenarioCmd(t *testing.T, kind command.Type) *command.BasicCommand {
	t.Helper()

	basic, err := command.New(kind, id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return basic
}

// givenCreated hands the scenario a persisted Created fact at version 1 —
// the state every "open task" scenario starts from.
func givenCreated(t *testing.T, title string, priority int) []event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	return []event.Event{mustEvent(t, evtTaskCreated, streamID, event.Version(1),
		TaskCreated{ID: streamID.String(), Title: title, Priority: priority})}
}

// givenDoneTask hands the scenario a persisted create+complete history on
// one stream — the state every "already done" scenario starts from.
func givenDoneTask(t *testing.T, title string, priority int) []event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	return []event.Event{
		mustEvent(t, evtTaskCreated, streamID, event.Version(1),
			TaskCreated{ID: streamID.String(), Title: title, Priority: priority}),
		mustEvent(
			t,
			evtTaskUpdated,
			streamID,
			event.Version(2),
			TaskUpdated{
				ID:       streamID.String(),
				Title:    title,
				Status:   StatusDone,
				Priority: priority,
			},
		),
	}
}

func mustEvent(
	t *testing.T,
	kind event.Type,
	streamID id.StreamID,
	version event.Version,
	payload any,
) event.Event {
	t.Helper()

	evt, err := event.New(kind, streamID, streamType, version, payload)
	if err != nil {
		t.Fatalf("event.New %s: %v", kind, err)
	}

	return evt
}

func TestScenario_CreateOnEmptyStream(t *testing.T) {
	t.Parallel()

	cmd := CreateTaskCmd{
		BasicCommand: scenarioCmd(t, cmdCreateTask),
		Title:        "Write scenario",
		Priority:     PriorityNormal,
	}

	scenario.Given[CreateTaskCmd](t, applyTask, TaskState{}).
		When(cmd, func(s TaskState, c CreateTaskCmd) ([]event.Event, error) {
			return decideCreate(c, s, event.Version(0))
		}).
		ThenState(applyTask, TaskState{}, TaskState{
			Exists: true, Title: "Write scenario", Priority: PriorityNormal,
		})
}

func TestScenario_CreateTwiceIsRejected(t *testing.T) {
	t.Parallel()

	cmd := CreateTaskCmd{
		BasicCommand: scenarioCmd(t, cmdCreateTask),
		Title:        "Duplicate",
		Priority:     PriorityNormal,
	}

	scenario.Given[CreateTaskCmd](
		t,
		applyTask,
		TaskState{},
		givenCreated(t, "Duplicate", PriorityNormal)...).
		When(cmd, func(s TaskState, c CreateTaskCmd) ([]event.Event, error) {
			return decideCreate(c, s, event.Version(1))
		}).
		ThenError(errTaskExists)
}

func TestScenario_CompleteOpenTask(t *testing.T) {
	t.Parallel()

	cmd := CompleteTaskCmd{BasicCommand: scenarioCmd(t, cmdCompleteTask)}

	scenario.Given[CompleteTaskCmd](
		t,
		applyTask,
		TaskState{},
		givenCreated(t, "Finish me", PriorityHigh)...).
		When(cmd, func(s TaskState, c CompleteTaskCmd) ([]event.Event, error) {
			return decideComplete(c, s, event.Version(1))
		}).
		ThenState(applyTask, TaskState{}, TaskState{
			Exists: true, Done: true, Title: "Finish me", Priority: PriorityHigh,
		})
}

func TestScenario_CompleteTwiceIsRejected(t *testing.T) {
	t.Parallel()

	cmd := CompleteTaskCmd{BasicCommand: scenarioCmd(t, cmdCompleteTask)}

	scenario.Given[CompleteTaskCmd](
		t,
		applyTask,
		TaskState{},
		givenDoneTask(t, "Done once", PriorityNormal)...).
		When(cmd, func(s TaskState, c CompleteTaskCmd) ([]event.Event, error) {
			return decideComplete(c, s, event.Version(2))
		}).
		ThenError(errTaskDone)
}

func TestScenario_DeleteOpenTask(t *testing.T) {
	t.Parallel()

	cmd := DeleteTaskCmd{BasicCommand: scenarioCmd(t, cmdDeleteTask)}

	scenario.Given[DeleteTaskCmd](
		t,
		applyTask,
		TaskState{},
		givenCreated(t, "Ephemeral", PriorityNormal)...).
		When(cmd, func(s TaskState, c DeleteTaskCmd) ([]event.Event, error) {
			return decideDelete(c, s, event.Version(1))
		}).
		Then(evtTaskDeleted)
}

func TestScenario_DeleteUnknownTaskIsRejected(t *testing.T) {
	t.Parallel()

	cmd := DeleteTaskCmd{BasicCommand: scenarioCmd(t, cmdDeleteTask)}

	scenario.Given[DeleteTaskCmd](t, applyTask, TaskState{}).
		When(cmd, func(s TaskState, c DeleteTaskCmd) ([]event.Event, error) {
			return decideDelete(c, s, event.Version(0))
		}).
		ThenError(errTaskGone)
}

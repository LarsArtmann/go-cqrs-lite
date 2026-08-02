package main

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Scenario tests — fluent BDD for the decider.
//
// Each test exercises a single decision path: Given some pre-existing events
// (folded into state), When a command is decided, Then specific events are
// emitted (or an error is returned).
//
// The scenario DSL's ThenError uses errors.Is, and the error-family package's
// Is method matches by code+family. So we construct errors with matching codes.
// ──────────────────────────────────────────────────────────────────────────

func mustEvt(typ event.Type, streamID id.StreamID, payload any) event.Event {
	evt, err := event.New(typ, streamID, streamType, event.Version(1), payload)
	if err != nil {
		panic(err)
	}

	return evt
}

// errMatch constructs an error with the same code+family as the decider would
// return, enabling errors.Is matching in ThenError.
//
//nolint:exhaustive // default covers all remaining families via Newf
func errMatch(family errorfamily.Family, code string) error {
	switch family {
	case errorfamily.Rejection:
		return errorfamily.NewRejection(code, "")
	case errorfamily.Conflict:
		return errorfamily.NewConflict(code, "")
	default:
		return errorfamily.Newf(family, code, "")
	}
}

func TestDecider_CreateTask(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()

	t.Run("succeeds for new aggregate", func(t *testing.T) {
		t.Parallel()

		scenario.Given[CreateTask, TaskState](t, applyTask, TaskState{}).
			When(CreateTask{ID: taskID, Title: "Write tests", Priority: PriorityHigh},
				func(s TaskState, cmd CreateTask) ([]event.Event, error) { return Create(cmd)(s, 0) }).
			Then(evtTaskCreated)
	})

	t.Run("rejects empty title", func(t *testing.T) {
		t.Parallel()

		scenario.Given[CreateTask, TaskState](t, applyTask, TaskState{}).
			When(CreateTask{ID: taskID, Title: "  "},
				func(s TaskState, cmd CreateTask) ([]event.Event, error) { return Create(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Rejection, "task.title.empty"))
	})

	t.Run("rejects duplicate creation", func(t *testing.T) {
		t.Parallel()

		scenario.Given[CreateTask, TaskState](
			t,
			applyTask,
			TaskState{},
			mustEvt(
				evtTaskCreated,
				taskID,
				TaskCreatedPayload{Title: "Existing", Priority: PriorityLow},
			),
		).
			When(CreateTask{ID: taskID, Title: "New"},
				func(s TaskState, cmd CreateTask) ([]event.Event, error) { return Create(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Conflict, "task.create.exists"))
	})
}

func TestDecider_Lifecycle(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()
	created := mustEvt(
		evtTaskCreated,
		taskID,
		TaskCreatedPayload{Title: "Demo", Priority: PriorityMedium},
	)

	t.Run("start after create", func(t *testing.T) {
		t.Parallel()

		scenario.Given[StartTask, TaskState](t, applyTask, TaskState{}, created).
			When(StartTask{ID: taskID},
				func(s TaskState, cmd StartTask) ([]event.Event, error) { return Start(cmd)(s, 0) }).
			Then(evtTaskStarted)
	})

	t.Run("complete skips directly to completed", func(t *testing.T) {
		t.Parallel()

		scenario.Given[CompleteTask, TaskState](t, applyTask, TaskState{}, created).
			When(CompleteTask{ID: taskID},
				func(s TaskState, cmd CompleteTask) ([]event.Event, error) { return Complete(cmd)(s, 0) }).
			Then(evtTaskCompleted)
	})

	t.Run("cannot archive pending task directly", func(t *testing.T) {
		t.Parallel()

		scenario.Given[ArchiveTask, TaskState](t, applyTask, TaskState{}, created).
			When(ArchiveTask{ID: taskID},
				func(s TaskState, cmd ArchiveTask) ([]event.Event, error) { return Archive(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Conflict, "task.archive.invalid_transition"))
	})
}

func TestDecider_AssignTask(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()
	created := mustEvt(
		evtTaskCreated,
		taskID,
		TaskCreatedPayload{Title: "Assigned Task", Priority: PriorityLow},
	)

	t.Run("assigns to a user", func(t *testing.T) {
		t.Parallel()

		scenario.Given[AssignTask, TaskState](t, applyTask, TaskState{}, created).
			When(AssignTask{ID: taskID, AssigneeID: "user-123"},
				func(s TaskState, cmd AssignTask) ([]event.Event, error) { return Assign(cmd)(s, 0) }).
			Then(evtTaskAssigned)
	})

	t.Run("rejects same assignee", func(t *testing.T) {
		t.Parallel()

		assigned := mustEvt(evtTaskAssigned, taskID, TaskAssignedPayload{AssigneeID: "user-123"})

		scenario.Given[AssignTask, TaskState](t, applyTask, TaskState{}, created, assigned).
			When(AssignTask{ID: taskID, AssigneeID: "user-123"},
				func(s TaskState, cmd AssignTask) ([]event.Event, error) { return Assign(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Conflict, "task.assign.same"))
	})
}

func TestDecider_DeleteTask(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()
	created := mustEvt(
		evtTaskCreated,
		taskID,
		TaskCreatedPayload{Title: "Doomed", Priority: PriorityLow},
	)

	t.Run("soft-deletes via tombstone", func(t *testing.T) {
		t.Parallel()

		scenario.Given[DeleteTask, TaskState](t, applyTask, TaskState{}, created).
			When(DeleteTask{ID: taskID},
				func(s TaskState, cmd DeleteTask) ([]event.Event, error) { return Delete(cmd)(s, 0) }).
			Then(evtTaskDeleted)
	})

	t.Run("cannot delete twice", func(t *testing.T) {
		t.Parallel()

		deleted := mustEvt(evtTaskDeleted, taskID, TaskDeletedPayload{})
		deleted, _ = event.MarkTombstone(deleted)

		scenario.Given[DeleteTask, TaskState](t, applyTask, TaskState{}, created, deleted).
			When(DeleteTask{ID: taskID},
				func(s TaskState, cmd DeleteTask) ([]event.Event, error) { return Delete(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Conflict, "task.delete.deleted"))
	})
}

func TestDecider_BlockBy(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()
	depID := id.NewStreamID()
	created := mustEvt(
		evtTaskCreated,
		taskID,
		TaskCreatedPayload{Title: "Blocked", Priority: PriorityMedium},
	)

	t.Run("adds dependency", func(t *testing.T) {
		t.Parallel()

		scenario.Given[BlockBy, TaskState](t, applyTask, TaskState{}, created).
			When(BlockBy{ID: taskID, DependencyID: depID},
				func(s TaskState, cmd BlockBy) ([]event.Event, error) { return AddBlocker(cmd)(s, 0) }).
			Then(evtTaskBlockedBy)
	})

	t.Run("rejects self-blocking", func(t *testing.T) {
		t.Parallel()

		scenario.Given[BlockBy, TaskState](t, applyTask, TaskState{}, created).
			When(BlockBy{ID: taskID, DependencyID: taskID},
				func(s TaskState, cmd BlockBy) ([]event.Event, error) { return AddBlocker(cmd)(s, 0) }).
			ThenError(errMatch(errorfamily.Rejection, "task.block.self"))
	})
}

func TestDecider_FoldState(t *testing.T) {
	t.Parallel()

	taskID := id.NewStreamID()

	t.Run("fold produces correct state after full lifecycle", func(t *testing.T) {
		t.Parallel()

		scenario.Given[CompleteTask, TaskState](
			t,
			applyTask,
			TaskState{},
			mustEvt(
				evtTaskCreated,
				taskID,
				TaskCreatedPayload{Title: "Test", Priority: PriorityHigh},
			),
			mustEvt(evtTaskAssigned, taskID, TaskAssignedPayload{AssigneeID: "user-1"}),
			mustEvt(evtTaskStarted, taskID, TaskStartedPayload{}),
		).
			When(CompleteTask{ID: taskID},
				func(s TaskState, cmd CompleteTask) ([]event.Event, error) { return Complete(cmd)(s, 0) }).
			ThenState(applyTask, TaskState{}, TaskState{
				Exists:     true,
				Title:      "Test",
				Priority:   PriorityHigh,
				AssigneeID: "user-1",
				Status:     StatusCompleted,
			})
	})
}

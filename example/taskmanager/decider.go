package main

import (
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Decider — pure fold + decide functions.
//
// The fold (apply) rebuilds state from events. The decide functions take
// the current state and a command, returning new events or a typed error.
//
// Both are pure functions — no side effects, no I/O, no infrastructure.
// This makes them trivially testable via the scenario DSL.
// ──────────────────────────────────────────────────────────────────────────

const maxTitleLength = 500

// applyTask is the fold function: it takes the current state and an event,
// and returns the new state. It never returns an error for valid event
// streams — errors only occur on corrupt/unexpected payloads.
func applyTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case evtTaskCreated:
		p, err := event.DecodePayloadAuto[TaskCreatedPayload](evt)
		if err != nil {
			return state, err
		}

		state.Title = p.Title
		state.Description = p.Description
		state.Priority = p.Priority
		state.Status = StatusPending
		state.Exists = true

	case evtTaskAssigned:
		p, err := event.DecodePayloadAuto[TaskAssignedPayload](evt)
		if err != nil {
			return state, err
		}

		state.AssigneeID = p.AssigneeID

	case evtTaskStarted:
		state.Status = StatusActive

	case evtTaskCompleted:
		state.Status = StatusCompleted

	case evtTaskArchived:
		state.Status = StatusArchived

	case evtTaskTitleUpdated:
		p, err := event.DecodePayloadAuto[TaskTitleUpdatedPayload](evt)
		if err != nil {
			return state, err
		}

		state.Title = p.Title

	case evtTaskPriorityChanged:
		p, err := event.DecodePayloadAuto[TaskPriorityChangedPayload](evt)
		if err != nil {
			return state, err
		}

		state.Priority = p.Priority

	case evtTaskDueDateSet:
		p, err := event.DecodePayloadAuto[TaskDueDateSetPayload](evt)
		if err != nil {
			return state, err
		}

		state.DueDate = p.DueDate

	case evtTaskBlockedBy:
		p, err := event.DecodePayloadAuto[TaskBlockedByPayload](evt)
		if err != nil {
			return state, err
		}

		depID, dErr := id.ParseAggregateID(p.DependencyID)
		if dErr != nil {
			return state, dErr
		}

		state.BlockedBy = append(state.BlockedBy, depID)

	case evtTaskUnblocked:
		p, err := event.DecodePayloadAuto[TaskUnblockedPayload](evt)
		if err != nil {
			return state, err
		}

		depID, dErr := id.ParseAggregateID(p.DependencyID)
		if dErr != nil {
			return state, dErr
		}

		state.BlockedBy = removeTaskID(state.BlockedBy, depID)

	case evtTaskDeleted:
		state.Tombstoned = true
	}

	return state, nil
}

// TaskDecider is the decider definition for the Repository.
var TaskDecider = decider.Decider[TaskState]{
	Initial: TaskState{},
	Apply:   applyTask,
}

// ──────────────────────────────────────────────────────────────────────────
// Repository adapters — each returns a DecideFunc that the Repository
// executes against the current (replayed) state. The version parameter
// enables optimistic concurrency (events are stamped version.Increment()).
// ──────────────────────────────────────────────────────────────────────────

// CreateTask command.
type CreateTask struct {
	ID          TaskID
	Title       string
	Description string
	Priority    Priority
}

func Create(cmd CreateTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if s.Exists {
			return nil, errorfamily.NewConflict("task.create.exists", "task already exists")
		}

		title, err := normaliseTitle(cmd.Title)
		if err != nil {
			return nil, err
		}

		evt, err := event.New(evtTaskCreated, cmd.ID, streamType, v.Increment(),
			TaskCreatedPayload{Title: title, Description: cmd.Description, Priority: cmd.Priority})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.create.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// AssignTask command.
type AssignTask struct {
	ID         TaskID
	AssigneeID string
}

func Assign(cmd AssignTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.assign.not_found",
				"task does not exist or is deleted",
			)
		}

		if s.AssigneeID == cmd.AssigneeID {
			return nil, errorfamily.NewConflict(
				"task.assign.same",
				"task already assigned to this user",
			)
		}

		evt, err := event.New(evtTaskAssigned, cmd.ID, streamType, v.Increment(),
			TaskAssignedPayload{AssigneeID: cmd.AssigneeID})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.assign.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// StartTask command.
type StartTask struct {
	ID TaskID
}

func Start(cmd StartTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.start.not_found",
				"task does not exist or is deleted",
			)
		}

		if !s.CanTransitionTo(StatusActive) {
			return nil, errorfamily.NewConflict("task.start.invalid_transition",
				"cannot start a task in "+string(s.Status)+" status")
		}

		evt, err := event.New(
			evtTaskStarted,
			cmd.ID,
			streamType,
			v.Increment(),
			TaskStartedPayload{},
		)
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.start.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// CompleteTask command.
type CompleteTask struct {
	ID TaskID
}

func Complete(cmd CompleteTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.complete.not_found",
				"task does not exist or is deleted",
			)
		}

		if !s.CanTransitionTo(StatusCompleted) {
			return nil, errorfamily.NewConflict("task.complete.invalid_transition",
				"cannot complete a task in "+string(s.Status)+" status")
		}

		evt, err := event.New(
			evtTaskCompleted,
			cmd.ID,
			streamType,
			v.Increment(),
			TaskCompletedPayload{},
		)
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.complete.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// ArchiveTask command.
type ArchiveTask struct {
	ID TaskID
}

func Archive(cmd ArchiveTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.archive.not_found",
				"task does not exist or is deleted",
			)
		}

		if !s.CanTransitionTo(StatusArchived) {
			return nil, errorfamily.NewConflict("task.archive.invalid_transition",
				"cannot archive a task in "+string(s.Status)+" status")
		}

		evt, err := event.New(
			evtTaskArchived,
			cmd.ID,
			streamType,
			v.Increment(),
			TaskArchivedPayload{},
		)
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.archive.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// UpdateTitle command.
type UpdateTitle struct {
	ID    TaskID
	Title string
}

func UpdateTaskTitle(cmd UpdateTitle) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.title.not_found",
				"task does not exist or is deleted",
			)
		}

		title, err := normaliseTitle(cmd.Title)
		if err != nil {
			return nil, err
		}

		if title == s.Title {
			return nil, errorfamily.NewConflict("task.title.unchanged", "title is the same")
		}

		evt, err := event.New(evtTaskTitleUpdated, cmd.ID, streamType, v.Increment(),
			TaskTitleUpdatedPayload{Title: title})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.title.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// ChangePriority command.
type ChangePriority struct {
	ID       TaskID
	Priority Priority
}

func ChangeTaskPriority(cmd ChangePriority) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.priority.not_found",
				"task does not exist or is deleted",
			)
		}

		if !cmd.Priority.Valid() {
			return nil, errorfamily.NewRejection("task.priority.invalid",
				"priority must be low, medium, high, or urgent")
		}

		if cmd.Priority == s.Priority {
			return nil, errorfamily.NewConflict("task.priority.unchanged", "priority is the same")
		}

		evt, err := event.New(evtTaskPriorityChanged, cmd.ID, streamType, v.Increment(),
			TaskPriorityChangedPayload{Priority: cmd.Priority})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.priority.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// SetDueDate command. A nil DueDate clears the deadline.
type SetDueDate struct {
	ID      TaskID
	DueDate *time.Time
}

func SetTaskDueDate(cmd SetDueDate) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.duedate.not_found",
				"task does not exist or is deleted",
			)
		}

		evt, err := event.New(evtTaskDueDateSet, cmd.ID, streamType, v.Increment(),
			TaskDueDateSetPayload{DueDate: cmd.DueDate})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.duedate.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// BlockBy command — declare a dependency (this task is blocked by another).
type BlockBy struct {
	ID           TaskID
	DependencyID TaskID
}

func AddBlocker(cmd BlockBy) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.block.not_found",
				"task does not exist or is deleted",
			)
		}

		if s.HasDependency(cmd.DependencyID) {
			return nil, errorfamily.NewConflict("task.block.exists", "dependency already exists")
		}

		if cmd.ID == cmd.DependencyID {
			return nil, errorfamily.NewRejection("task.block.self", "a task cannot block itself")
		}

		evt, err := event.New(evtTaskBlockedBy, cmd.ID, streamType, v.Increment(),
			TaskBlockedByPayload{DependencyID: cmd.DependencyID.String()})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.block.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// UnblockBy command — remove a dependency.
type UnblockBy struct {
	ID           TaskID
	DependencyID TaskID
}

func RemoveBlocker(cmd UnblockBy) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.IsActive() {
			return nil, errorfamily.NewRejection(
				"task.unblock.not_found",
				"task does not exist or is deleted",
			)
		}

		if !s.HasDependency(cmd.DependencyID) {
			return nil, errorfamily.NewConflict("task.unblock.missing", "dependency does not exist")
		}

		evt, err := event.New(evtTaskUnblocked, cmd.ID, streamType, v.Increment(),
			TaskUnblockedPayload{DependencyID: cmd.DependencyID.String()})
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.unblock.event",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

// DeleteTask command — soft-delete via tombstone metadata.
type DeleteTask struct {
	ID TaskID
}

func Delete(cmd DeleteTask) decider.DecideFunc[TaskState] {
	return func(s TaskState, v event.Version) ([]event.Event, error) {
		if !s.Exists {
			return nil, errorfamily.NewRejection("task.delete.not_found", "task does not exist")
		}

		if s.Tombstoned {
			return nil, errorfamily.NewConflict("task.delete.deleted", "task already deleted")
		}

		evt, err := event.New(
			evtTaskDeleted,
			cmd.ID,
			streamType,
			v.Increment(),
			TaskDeletedPayload{},
		)
		if err != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.delete.event",
				"build event: %v",
				err,
			)
		}

		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, errorfamily.Newf(
				errorfamily.Infrastructure,
				"task.delete.tombstone",
				"mark: %v",
				markErr,
			)
		}

		return []event.Event{marked}, nil
	}
}

// ──────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────

func removeTaskID(ids []id.StreamID, target id.StreamID) []id.StreamID {
	result := make([]TaskID, 0, len(ids))

	for _, id := range ids {
		if id != target {
			result = append(result, id)
		}
	}

	return result
}

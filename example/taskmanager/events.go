package main

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Events — per-event payloads (NOT a fat payload shared across all events).
//
// Each event carries ONLY the data that changed. This is the correct event
// sourcing pattern: consumers can diff events without replaying full state,
// and adding a field to one event type doesn't require migrating others.
// ──────────────────────────────────────────────────────────────────────────

const (
	evtTaskCreated         = event.Type("task.created")
	evtTaskAssigned        = event.Type("task.assigned")
	evtTaskStarted         = event.Type("task.started")
	evtTaskCompleted       = event.Type("task.completed")
	evtTaskArchived        = event.Type("task.archived")
	evtTaskTitleUpdated    = event.Type("task.title_updated")
	evtTaskPriorityChanged = event.Type("task.priority_changed")
	evtTaskDueDateSet      = event.Type("task.due_date_set")
	evtTaskBlockedBy       = event.Type("task.blocked_by")
	evtTaskUnblocked       = event.Type("task.unblocked")
	evtTaskDeleted         = event.Type("task.deleted")
)

// TaskCreatedPayload carries the initial task data. Created separately from
// assignment — a task can exist without an assignee.
type TaskCreatedPayload struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
}

type TaskAssignedPayload struct {
	AssigneeID string `json:"assigneeId"`
}

type TaskStartedPayload struct{}

type TaskCompletedPayload struct{}

type TaskArchivedPayload struct{}

type TaskTitleUpdatedPayload struct {
	Title string `json:"title"`
}

type TaskPriorityChangedPayload struct {
	Priority Priority `json:"priority"`
}

type TaskDueDateSetPayload struct {
	//cqrs-lint:ignore(C013) library code or intentional pattern
	DueDate *time.Time `json:"dueDate,omitempty"`
}

type TaskBlockedByPayload struct {
	DependencyID string `json:"dependencyId"`
}

type TaskUnblockedPayload struct {
	DependencyID string `json:"dependencyId"`
}

type TaskDeletedPayload struct{}

// AllEventTypes returns every event type the Task aggregate emits, in
// lifecycle order. Used by projections and catalog generation.
func AllEventTypes() []event.Type {
	return []event.Type{
		evtTaskCreated, evtTaskAssigned, evtTaskStarted, evtTaskCompleted,
		evtTaskArchived, evtTaskTitleUpdated, evtTaskPriorityChanged,
		evtTaskDueDateSet, evtTaskBlockedBy, evtTaskUnblocked, evtTaskDeleted,
	}
}

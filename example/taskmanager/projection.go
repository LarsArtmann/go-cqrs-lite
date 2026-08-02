package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

var errViewMissingForUpdate = errors.New("projection: view does not exist for update")

// ──────────────────────────────────────────────────────────────────────────
// Read Model — the materialised view that projections build from events.
//
// TaskView is a flat, query-optimised projection of the Task aggregate.
// It is NOT the aggregate state — it's the data shape the query side serves
// to HTTP clients. Projections rebuild it from the event stream.
// ──────────────────────────────────────────────────────────────────────────

// TaskView is the read-model representation served by the query side.
type TaskView struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Status      Status   `json:"status"`
	AssigneeID  string   `json:"assigneeId,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
	BlockedBy   []string `json:"blockedBy,omitempty"`
	Tombstoned  bool     `json:"tombstoned,omitempty"`
}

// IsTombstoned enables Materialize.List to filter soft-deleted records.
func (v *TaskView) IsTombstoned() bool { return v.Tombstoned }

// taskViewKey extracts the read-model key from an event.
func taskViewKey(evt event.Event) (id.StreamID, error) {
	return evt.StreamID(), nil
}

// configureProjection wires the consumer's event-handling callbacks into
// the Materialize projection. Each event type updates the view incrementally.
func configureProjection(mat *stack.Materialize[TaskView, id.StreamID]) {
	mat.OnCreate = func(_ context.Context, evt event.Event) (*TaskView, error) {
		p, err := event.DecodePayloadAuto[TaskCreatedPayload](evt)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
		}

		return &TaskView{
			ID:          evt.StreamID().String(),
			Title:       p.Title,
			Description: p.Description,
			Priority:    p.Priority,
			Status:      StatusPending,
		}, nil
	}

	mat.OnUpdate = func(_ context.Context, evt event.Event, existing *TaskView) (*TaskView, error) {
		if existing == nil {
			return nil, fmt.Errorf("%w: %s", errViewMissingForUpdate, evt.StreamID())
		}

		updated := *existing

		switch evt.Type() {
		case evtTaskAssigned:
			p, err := event.DecodePayloadAuto[TaskAssignedPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			updated.AssigneeID = p.AssigneeID

		case evtTaskStarted:
			updated.Status = StatusActive

		case evtTaskCompleted:
			updated.Status = StatusCompleted

		case evtTaskArchived:
			updated.Status = StatusArchived

		case evtTaskTitleUpdated:
			p, err := event.DecodePayloadAuto[TaskTitleUpdatedPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			updated.Title = p.Title

		case evtTaskPriorityChanged:
			p, err := event.DecodePayloadAuto[TaskPriorityChangedPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			updated.Priority = p.Priority

		case evtTaskDueDateSet:
			p, err := event.DecodePayloadAuto[TaskDueDateSetPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			if p.DueDate != nil {
				updated.DueDate = p.DueDate.Format("2006-01-02T15:04:05Z")
			} else {
				updated.DueDate = ""
			}

		case evtTaskBlockedBy:
			p, err := event.DecodePayloadAuto[TaskBlockedByPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			updated.BlockedBy = append(updated.BlockedBy, p.DependencyID)

		case evtTaskUnblocked:
			p, err := event.DecodePayloadAuto[TaskUnblockedPayload](evt)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", evt.Type(), err)
			}

			updated.BlockedBy = removeString(updated.BlockedBy, p.DependencyID)
		}

		return &updated, nil
	}

	mat.OnTombstone = func(_ context.Context, _ event.Event, existing *TaskView) (*TaskView, error) {
		if existing == nil {
			return &TaskView{Tombstoned: true}, nil
		}

		updated := *existing
		updated.Tombstoned = true

		return &updated, nil
	}
}

func removeString(slice []string, target string) []string {
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}

	return result
}

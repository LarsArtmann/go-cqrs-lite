package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Metaengine — the cost-based query planner.
//
// This is the STRATEGIC FUTURE of go-cqrs-lite. Instead of the O(N)
// Materialize.List + Go-side filter in handleListTasks, metaengine maintains:
//
//   - task_counts_by_status: a Counter ADT tracking counts by lifecycle status
//     (O(1) aggregate read).
//   - task_views: a Map ADT storing TaskView per task ID, with FilterOnField
//     for Status enabling SQLite json_extract pushdown (O(logN) filtered scan
//     instead of O(N) Go-side filter).
//
// The planner inspects the declared fold return types, infers the ADT
// (Counter vs Map), evaluates available engines (Memory + SQLite), and assigns
// each query to the cheapest engine that supports its operations.
// ──────────────────────────────────────────────────────────────────────────

// taskCountsInput is the query input for the aggregate counter read.
type taskCountsInput struct{}

// listTasksInput carries the optional status filter for list queries.
type listTasksInput struct {
	Status string `json:"status,omitempty"`
}

const estimatedTaskVolume = 10_000

// setupMetaEngine builds a metaengine Store with two queries using the
// modern DX helpers (PlanFromSQLite + TypeDecoder). It returns the store,
// a projection adapter for the projection host, and the *sql.DB for lifecycle.
func setupMetaEngine(
	logger *slog.Logger,
	dsn string,
) (*metaengine.Store, *projectionadapter.Adapter, *sql.DB, error) {
	taskCounts := metaengine.Query[taskCountsInput, map[string]int64](
		"task_counts_by_status",
		metaengine.OnTyped(
			string(evtTaskCreated),
			projectionadapter.EventWithID[TaskCreatedPayload]{},
			func(_ projectionadapter.EventWithID[TaskCreatedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusPending): 1}
			},
		),
		metaengine.OnTyped(
			string(evtTaskStarted),
			projectionadapter.EventWithID[TaskStartedPayload]{},
			func(_ projectionadapter.EventWithID[TaskStartedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusActive): 1, string(StatusPending): -1}
			},
		),
		metaengine.OnTyped(
			string(evtTaskCompleted),
			projectionadapter.EventWithID[TaskCompletedPayload]{},
			func(_ projectionadapter.EventWithID[TaskCompletedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusCompleted): 1, string(StatusActive): -1}
			},
		),
		metaengine.OnTyped(
			string(evtTaskArchived),
			projectionadapter.EventWithID[TaskArchivedPayload]{},
			func(_ projectionadapter.EventWithID[TaskArchivedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusArchived): 1, string(StatusCompleted): -1}
			},
		),
		metaengine.Volume(estimatedTaskVolume),
	)

	taskViews := metaengine.Query[listTasksInput, TaskView](
		"task_views",
		metaengine.OnTyped(
			string(evtTaskCreated),
			projectionadapter.EventWithID[TaskCreatedPayload]{},
			func(e projectionadapter.EventWithID[TaskCreatedPayload]) (string, TaskView) {
				return e.ID, TaskView{
					ID:          e.ID,
					Title:       e.Payload.Title,
					Description: e.Payload.Description,
					Priority:    e.Payload.Priority,
					Status:      StatusPending,
				}
			},
		),
		metaengine.OnTyped(
			string(evtTaskAssigned),
			projectionadapter.EventWithID[TaskAssignedPayload]{},
			func(e projectionadapter.EventWithID[TaskAssignedPayload], prev TaskView) TaskView {
				prev.AssigneeID = e.Payload.AssigneeID

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskStarted),
			projectionadapter.EventWithID[TaskStartedPayload]{},
			func(_ projectionadapter.EventWithID[TaskStartedPayload], prev TaskView) TaskView {
				prev.Status = StatusActive

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskCompleted),
			projectionadapter.EventWithID[TaskCompletedPayload]{},
			func(_ projectionadapter.EventWithID[TaskCompletedPayload], prev TaskView) TaskView {
				prev.Status = StatusCompleted

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskArchived),
			projectionadapter.EventWithID[TaskArchivedPayload]{},
			func(_ projectionadapter.EventWithID[TaskArchivedPayload], prev TaskView) TaskView {
				prev.Status = StatusArchived

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskTitleUpdated),
			projectionadapter.EventWithID[TaskTitleUpdatedPayload]{},
			func(e projectionadapter.EventWithID[TaskTitleUpdatedPayload], prev TaskView) TaskView {
				prev.Title = e.Payload.Title

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskPriorityChanged),
			projectionadapter.EventWithID[TaskPriorityChangedPayload]{},
			func(e projectionadapter.EventWithID[TaskPriorityChangedPayload], prev TaskView) TaskView {
				prev.Priority = e.Payload.Priority

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskDueDateSet),
			projectionadapter.EventWithID[TaskDueDateSetPayload]{},
			func(e projectionadapter.EventWithID[TaskDueDateSetPayload], prev TaskView) TaskView {
				if e.Payload.DueDate != nil {
					prev.DueDate = e.Payload.DueDate.Format("2006-01-02T15:04:05Z")
				} else {
					prev.DueDate = ""
				}

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskBlockedBy),
			projectionadapter.EventWithID[TaskBlockedByPayload]{},
			func(e projectionadapter.EventWithID[TaskBlockedByPayload], prev TaskView) TaskView {
				prev.BlockedBy = append(prev.BlockedBy, e.Payload.DependencyID)

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskUnblocked),
			projectionadapter.EventWithID[TaskUnblockedPayload]{},
			func(e projectionadapter.EventWithID[TaskUnblockedPayload], prev TaskView) TaskView {
				prev.BlockedBy = removeString(prev.BlockedBy, e.Payload.DependencyID)

				return prev
			},
		),
		metaengine.OnTyped(
			string(evtTaskDeleted),
			projectionadapter.EventWithID[TaskDeletedPayload]{},
			metaengine.Remove[TaskView](),
		),
		metaengine.FilterOnField[TaskView]("status", metaengine.FilterEq),
		metaengine.SortOnField[TaskView]("priority", true),
		metaengine.Volume(estimatedTaskVolume),
	)

	store, meDB, err := sqliteengine.PlanFromSQLite(dsn, taskCounts, taskViews)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("metaengine: plan: %w", err)
	}

	store.LogPlan(logger)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(evtTaskCreated, TaskCreatedPayload{}),
		projectionadapter.Register(evtTaskAssigned, TaskAssignedPayload{}),
		projectionadapter.Register(evtTaskStarted, TaskStartedPayload{}),
		projectionadapter.Register(evtTaskCompleted, TaskCompletedPayload{}),
		projectionadapter.Register(evtTaskArchived, TaskArchivedPayload{}),
		projectionadapter.Register(evtTaskTitleUpdated, TaskTitleUpdatedPayload{}),
		projectionadapter.Register(evtTaskPriorityChanged, TaskPriorityChangedPayload{}),
		projectionadapter.Register(evtTaskDueDateSet, TaskDueDateSetPayload{}),
		projectionadapter.Register(evtTaskBlockedBy, TaskBlockedByPayload{}),
		projectionadapter.Register(evtTaskUnblocked, TaskUnblockedPayload{}),
		projectionadapter.Register(evtTaskDeleted, TaskDeletedPayload{}),
	)

	adapter := projectionadapter.NewWithDecoder("metaengine-tasks", store, decoder)

	return store, adapter, meDB, nil
}

// handleGetTaskStats serves GET /api/stats — returns task counts by status
// from the metaengine Counter projection. This is an O(1) aggregate read.
func (s *Server) handleGetTaskStats(w http.ResponseWriter, r *http.Request) {
	if s.MetaEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "metaengine not configured")

		return
	}

	counts, err := metaengine.ExecuteTyped[taskCountsInput, map[string]int64](
		r.Context(), s.MetaEngine, taskCountsInput{},
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "metaengine query: "+err.Error())

		return
	}

	pending := string(StatusPending)
	active := string(StatusActive)
	completed := string(StatusCompleted)
	archived := string(StatusArchived)
	result := map[string]int64{
		pending:   counts[pending],
		active:    counts[active],
		completed: counts[completed],
		archived:  counts[archived],
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"counts": result,
		"total":  result[pending] + result[active] + result[completed] + result[archived],
	})
}

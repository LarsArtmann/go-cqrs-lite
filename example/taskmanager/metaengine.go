package main

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
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

// eventWithID wraps an event payload with the stream ID (entity key).
// metaengine fold handlers need the entity ID as the Map key, but the raw
// event payload does not contain it — it lives in the event's StreamID.
// The projectionadapter's EventDecoder produces these wrappers.
type eventWithID[P any] struct {
	ID      string
	Payload P
}

// taskCountsInput is the query input for the aggregate counter read.
// It has no fields because ReadAggregate returns the entire counter map.
type taskCountsInput struct{}

// listTasksInput carries the optional status filter for list queries.
// When Status is empty, the scan returns all non-deleted tasks.
type listTasksInput struct {
	Status string `json:"status,omitempty"`
}

var errNoFoldForEventType = errors.New("metaengine: no fold for event type")

const estimatedTaskVolume = 10_000

// onTyped wraps metaengine.OnTyped and binds the fold to the CQRS event type
// string (e.g. "task.created") rather than the Go struct name. This is
// necessary because the event store uses semantic type strings.
//
//nolint:ireturn // factory returning metaengine.Fold interface for query declaration
func onTyped[E any](eventType string, sample E, handler any) metaengine.Fold {
	return metaengine.OnTyped[E](eventType, sample, handler)
}

// taskEventDecoder converts a full CQRS event (including StreamID) into a
// typed eventWithID[P] value that metaengine fold handlers expect. This is
// the bridge between the event-sourced world (where the entity ID is the
// stream ID) and the metaengine world (where the fold handler receives a
// plain value).
//
// The adapter calls this for each event, then passes the typed value to
// store.Apply, which routes to all registered queries listening to that
// event type.
func taskEventDecoder(evt event.Event) (any, error) {
	id := evt.StreamID().String()

	switch evt.Type() {
	case evtTaskCreated:
		var p TaskCreatedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskCreatedPayload]{ID: id, Payload: p}, nil

	case evtTaskAssigned:
		var p TaskAssignedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskAssignedPayload]{ID: id, Payload: p}, nil

	case evtTaskStarted:
		return eventWithID[TaskStartedPayload]{ID: id}, nil

	case evtTaskCompleted:
		return eventWithID[TaskCompletedPayload]{ID: id}, nil

	case evtTaskArchived:
		return eventWithID[TaskArchivedPayload]{ID: id}, nil

	case evtTaskTitleUpdated:
		var p TaskTitleUpdatedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskTitleUpdatedPayload]{ID: id, Payload: p}, nil

	case evtTaskPriorityChanged:
		var p TaskPriorityChangedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskPriorityChangedPayload]{ID: id, Payload: p}, nil

	case evtTaskDueDateSet:
		var p TaskDueDateSetPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskDueDateSetPayload]{ID: id, Payload: p}, nil

	case evtTaskBlockedBy:
		var p TaskBlockedByPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskBlockedByPayload]{ID: id, Payload: p}, nil

	case evtTaskUnblocked:
		var p TaskUnblockedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", evt.Type(), err)
		}

		return eventWithID[TaskUnblockedPayload]{ID: id, Payload: p}, nil

	case evtTaskDeleted:
		return eventWithID[TaskDeletedPayload]{ID: id}, nil

	default:
		//cqrs-lint:ignore(C025) library code or intentional pattern
		return nil, fmt.Errorf("%w: %q", errNoFoldForEventType, evt.Type())
	}
}

// setupMetaEngine builds a metaengine Store with two queries:
//
//  1. task_counts_by_status — Counter ADT for O(1) status aggregate reads.
//  2. task_views — Map ADT with FilterOnField("status") for SQL pushdown
//     filtered scans, replacing the O(N) Materialize.List + Go-side filter.
//
// It opens a SQLite engine from the same DSN (separate connection, separate
// tables: meta_map, meta_counter, etc.), plus a Memory engine for the planner
// to choose from. The returned *sql.DB is registered with the bundle for
// lifecycle cleanup via stack.WithCloser.
func setupMetaEngine(
	logger *slog.Logger,
	dsn string,
) (*metaengine.Store, *projectionadapter.Adapter, *sql.DB, error) {
	meDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("metaengine: open sqlite: %w", err)
	}

	meDB.SetMaxOpenConns(1) // SQLite: serialize writes, ensure :memory: visibility.

	// PRAGMA busy_timeout eliminates "database is locked" errors under
	// concurrent access; WAL mode improves write throughput (3-10x faster).
	// On :memory: databases WAL is a no-op (returns "memory") but never errors.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := meDB.ExecContext(context.Background(), pragma); err != nil {
			_ = meDB.Close()

			return nil, nil, nil, fmt.Errorf("metaengine: %s: %w", pragma, err)
		}
	}

	sqliteEng, err := metaengine.NewSQLiteEngine(meDB)
	if err != nil {
		_ = meDB.Close()

		return nil, nil, nil, fmt.Errorf("metaengine: sqlite engine: %w", err)
	}

	taskCounts := metaengine.Query[taskCountsInput, map[string]int64](
		"task_counts_by_status",
		// Counter ADT for O(1) status aggregate reads. Kept alongside the Map
		// query because O(1) counter reads are cheaper than O(N) scan+count
		// from task_views — the right pattern for dashboard/stats endpoints.
			onTyped(string(evtTaskCreated), eventWithID[TaskCreatedPayload]{},
			func(_ eventWithID[TaskCreatedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusPending): 1}
			}),
		onTyped(string(evtTaskStarted), eventWithID[TaskStartedPayload]{},
			func(_ eventWithID[TaskStartedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusActive): 1, string(StatusPending): -1}
			}),
			onTyped(string(evtTaskCompleted), eventWithID[TaskCompletedPayload]{},
			func(_ eventWithID[TaskCompletedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusCompleted): 1, string(StatusActive): -1}
			}),
			onTyped(string(evtTaskArchived), eventWithID[TaskArchivedPayload]{},
			func(_ eventWithID[TaskArchivedPayload]) metaengine.Delta {
				return metaengine.Delta{string(StatusArchived): 1, string(StatusCompleted): -1}
			}),
		// Volume hint: helps the planner estimate cost.
		metaengine.Volume(estimatedTaskVolume),
	)

	taskViews := metaengine.Query[listTasksInput, TaskView](
		"task_views",
		// Map folds: TaskCreated inserts a TaskView keyed by stream ID.
		// Lifecycle events update the view. TaskDeleted removes it.
		// FilterOnField enables SQLite json_extract pushdown on the "status"
		// JSON field, replacing the O(N) Go-side filter in handleListTasks.
		onTyped(string(evtTaskCreated), eventWithID[TaskCreatedPayload]{},
			func(e eventWithID[TaskCreatedPayload]) (string, TaskView) {
				return e.ID, TaskView{
					ID:          e.ID,
					Title:       e.Payload.Title,
					Description: e.Payload.Description,
					Priority:    e.Payload.Priority,
					Status:      StatusPending,
				}
			}),
		onTyped(string(evtTaskAssigned), eventWithID[TaskAssignedPayload]{},
			func(e eventWithID[TaskAssignedPayload], prev TaskView) TaskView {
				prev.AssigneeID = e.Payload.AssigneeID

				return prev
			}),
		onTyped(string(evtTaskStarted), eventWithID[TaskStartedPayload]{},
			func(_ eventWithID[TaskStartedPayload], prev TaskView) TaskView {
				prev.Status = StatusActive

				return prev
			}),
		onTyped(string(evtTaskCompleted), eventWithID[TaskCompletedPayload]{},
			func(_ eventWithID[TaskCompletedPayload], prev TaskView) TaskView {
				prev.Status = StatusCompleted

				return prev
			}),
		onTyped(string(evtTaskArchived), eventWithID[TaskArchivedPayload]{},
			func(_ eventWithID[TaskArchivedPayload], prev TaskView) TaskView {
				prev.Status = StatusArchived

				return prev
			}),
		onTyped(string(evtTaskTitleUpdated), eventWithID[TaskTitleUpdatedPayload]{},
			func(e eventWithID[TaskTitleUpdatedPayload], prev TaskView) TaskView {
				prev.Title = e.Payload.Title

				return prev
			}),
		onTyped(string(evtTaskPriorityChanged), eventWithID[TaskPriorityChangedPayload]{},
			func(e eventWithID[TaskPriorityChangedPayload], prev TaskView) TaskView {
				prev.Priority = e.Payload.Priority

				return prev
			}),
		onTyped(string(evtTaskDueDateSet), eventWithID[TaskDueDateSetPayload]{},
			func(e eventWithID[TaskDueDateSetPayload], prev TaskView) TaskView {
				if e.Payload.DueDate != nil {
					prev.DueDate = e.Payload.DueDate.Format("2006-01-02T15:04:05Z")
				} else {
					prev.DueDate = ""
				}

				return prev
			}),
		onTyped(string(evtTaskBlockedBy), eventWithID[TaskBlockedByPayload]{},
			func(e eventWithID[TaskBlockedByPayload], prev TaskView) TaskView {
				prev.BlockedBy = append(prev.BlockedBy, e.Payload.DependencyID)

				return prev
			}),
		onTyped(string(evtTaskUnblocked), eventWithID[TaskUnblockedPayload]{},
			func(e eventWithID[TaskUnblockedPayload], prev TaskView) TaskView {
				prev.BlockedBy = removeString(prev.BlockedBy, e.Payload.DependencyID)

				return prev
			}),
		onTyped(string(evtTaskDeleted), eventWithID[TaskDeletedPayload]{},
			metaengine.Remove[TaskView]()),
		metaengine.FilterOnField[TaskView]("status", metaengine.FilterEq),
		metaengine.SortOnField[TaskView]("priority", true), // default DESC
		metaengine.Volume(estimatedTaskVolume),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
		taskCounts, taskViews,
	)
	if err != nil {
		_ = meDB.Close()

		return nil, nil, nil, fmt.Errorf("metaengine: plan: %w", err)
	}

	// Log the planner's decision so the optimizer's reasoning is visible.
	if plan := store.Plan(); plan != nil {
		for _, q := range plan.Queries {
			logger.Info(
				"metaengine: query planned",
				"query", q.QueryName,
				"adt", q.ADT,
				"engine", q.EngineName,
				"complexity", q.Complexity,
				"read_pattern", q.ReadPattern,
				"estimated_latency_ms", q.Cost.EstimatedLatencyMs,
			)
		}

		for _, d := range plan.Diagnostics {
			logger.Warn("metaengine: diagnostic",
				"query", d.Query, "level", d.Level, "message", d.Message)
		}
	}

	adapter := projectionadapter.New("metaengine-tasks", store, nil,
		projectionadapter.WithEventDecoder(taskEventDecoder))

	return store, adapter, meDB, nil
}

// handleGetTaskStats serves GET /api/stats — returns task counts by status
// from the metaengine Counter projection. This is an O(1) aggregate read,
// compared to the O(N) scan that Materialize.List would require.
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

	// Ensure all statuses are present even if zero (no events for that status yet).
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

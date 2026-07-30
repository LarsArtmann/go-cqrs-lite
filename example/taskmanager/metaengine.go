package main

import (
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Metaengine — the cost-based query planner.
//
// This is the STRATEGIC FUTURE of go-cqrs-lite. Instead of the O(N)
// Materialize.List + Go-side filter in handleListTasks, metaengine maintains
// a Counter ADT that tracks task counts by lifecycle status — an O(1)
// aggregate read.
//
// The planner inspects the declared fold return types, infers the ADT
// (Counter), evaluates available engines, and assigns this query to the
// cheapest engine that supports Counter operations. The Plan() output is
// logged at startup so you can see the optimizer's decision.
//
// See: docs/planning/meta-engine-design.md (the vision)
//      docs/planning/meta-engine-assumptions-and-query-planning.md (the model)
// ──────────────────────────────────────────────────────────────────────────

// taskCountsInput is the query input for the aggregate counter read.
// It has no fields because ReadAggregate returns the entire counter map.
type taskCountsInput struct{}

// onTyped wraps metaengine.On and overrides the EventType to match the
// CQRS event type string (e.g. "task.created") rather than the Go struct
// name (e.g. "TaskCreatedPayload"). This is necessary because metaengine.On
// infers event types from reflect.Type.Name(), but the event store uses
// semantic type strings.
func onTyped[E any](eventType string, sample E, handler any) metaengine.Fold {
	fold := metaengine.On(sample, handler)
	fold.EventType = eventType

	return fold
}

// taskPayloadDecoder converts CQRS event payloads into typed Go structs
// that the metaengine fold handlers expect. The adapter calls this for each
// event, then passes the typed value to store.Apply.
func taskPayloadDecoder(eventType string, payload []byte) (any, error) {
	switch eventType {
	case string(evtTaskCreated):
		var p TaskCreatedPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("metaengine: decode %s: %w", eventType, err)
		}

		return p, nil

	case string(evtTaskStarted), string(evtTaskCompleted), string(evtTaskArchived):
		return struct{}{}, nil

	default:
		//cqrs-lint:ignore(C025) library code or intentional pattern
		return nil, fmt.Errorf(
			"metaengine: no fold for event type %q",
			eventType,
		) //cqrs-lint:ignore(D006) example code, not production error path
	}
}

// setupMetaEngine builds a metaengine Store with a Counter query that
// tracks task counts by lifecycle status, wraps it in a projectionadapter,
// and returns both the Store (for query reads) and the Adapter (for
// projectionhost registration).
func setupMetaEngine(logger *slog.Logger) (*metaengine.Store, *projectionadapter.Adapter, error) {
	taskCounts := metaengine.Query[taskCountsInput, map[string]int64](
		"task_counts_by_status",
		// Counter folds: each event returns a Delta (map[string]int64) of
		// counter increments/decrements. The planner infers ADTCounter from
		// the Delta return type.
		onTyped(string(evtTaskCreated), TaskCreatedPayload{},
			func(_ TaskCreatedPayload) metaengine.Delta {
				return metaengine.Delta{"pending": 1}
			}),
		onTyped(string(evtTaskStarted), TaskStartedPayload{},
			func(_ TaskStartedPayload) metaengine.Delta {
				return metaengine.Delta{"active": 1, "pending": -1}
			}),
		onTyped(string(evtTaskCompleted), TaskCompletedPayload{},
			func(_ TaskCompletedPayload) metaengine.Delta {
				return metaengine.Delta{"completed": 1, "active": -1}
			}),
		onTyped(string(evtTaskArchived), TaskArchivedPayload{},
			func(_ TaskArchivedPayload) metaengine.Delta {
				return metaengine.Delta{"archived": 1, "completed": -1}
			}),
		// Volume hint: helps the planner estimate cost.
		metaengine.Volume(10_000),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		taskCounts,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("metaengine: plan: %w", err)
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

	adapter := projectionadapter.New("metaengine-task-counts", store, taskPayloadDecoder)

	return store, adapter, nil
}

// handleGetTaskStats serves GET /api/stats — returns task counts by status
// from the metaengine Counter projection. This is an O(1) aggregate read,
// compared to the O(N) Materialize.List + filter in handleListTasks.
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
	result := map[string]int64{
		"pending":   counts["pending"],
		"active":    counts["active"],
		"completed": counts["completed"],
		"archived":  counts["archived"],
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"counts": result,
		"total":  result["pending"] + result["active"] + result["completed"] + result["archived"],
	})
}

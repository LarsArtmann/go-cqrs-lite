package metaengine

import (
	"log/slog"
)

// PlanFromMemory is the one-shot convenience for in-memory testing:
// create a Memory engine and plan queries against it.
//
//	store, err := metaengine.PlanFromMemory(statsQuery, historyQuery)
//	defer store.Close()
func PlanFromMemory(args ...any) (*Store, error) {
	return Plan([]Engine{NewMemoryEngine()}, args...)
}

// LogPlan logs the planner's decisions and diagnostics via slog. Call it
// once after Plan or PlanFromSQLite, at startup, so the optimizer's choices
// are visible in production logs:
//
//	store, _, _ := metaengine.Plan(engines, stats)
//	store.LogPlan(logger)
//
// Each query logs: name, ADT, assigned engine, complexity, read pattern,
// estimated latency. Diagnostics (WARN/SCREAM) are logged separately.
// If no plan exists (nil store or planning error), this is a no-op.
func (s *Store) LogPlan(logger *slog.Logger) {
	plan := s.Plan()
	if plan == nil {
		return
	}

	for _, q := range plan.Queries {
		logger.Info("metaengine: query planned",
			"query", q.QueryName,
			"adt", string(q.ADT),
			"engine", q.EngineName,
			"complexity", string(q.Complexity),
			"read_pattern", string(q.ReadPattern),
			"estimated_latency_ms", q.Cost.EstimatedLatencyMs,
		)
	}

	for _, d := range plan.Diagnostics {
		logger.Warn("metaengine: diagnostic",
			"query", d.Query,
			"level", d.Level,
			"message", d.Message,
		)
	}
}

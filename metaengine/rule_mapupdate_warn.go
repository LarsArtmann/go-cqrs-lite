package metaengine

import "fmt"

// mapUpdateReplicationRule warns when a Map ADT query with update folds is
// routed to a replicated engine with non-zero lag. On replicated engines,
// MapUpdate operations risk conflict (multi-leader concurrent writes to the
// same key) or staleness (single-leader write bottleneck). The rule advises
// using idempotent folds or a single-leader topology for update-heavy
// Map collections.
//
// This is advisory (WARN) — the planner still allows the assignment. Consumers
// who know their fold is idempotent can safely ignore the warning.
type mapUpdateReplicationRule struct{}

func (*mapUpdateReplicationRule) Name() string { return "mapupdate-replication" }

func (r *mapUpdateReplicationRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		if q.ADT != ADTMap {
			continue
		}

		meta, ok := ctx.Store.queries[q.QueryName]
		if !ok {
			continue
		}

		hasUpdate := false
		for _, f := range meta.QueryFolds() {
			if f.Kind() == FoldUpdate {
				hasUpdate = true
				break
			}
		}
		if !hasUpdate {
			continue
		}

		profile := meta.QueryEngine().Profile()
		if !profile.IsReplicated() {
			continue
		}

		lag := profile.EffectiveReplicationLag()
		if lag <= 0 {
			continue
		}

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelWarn,
			Query: q.QueryName,
			Message: fmt.Sprintf(
				"MapUpdate on replicated engine %q (%s, lag=%s) — updates may conflict; use idempotent folds or single-leader topology",
				profile.Name, profile.Replication, lag,
			),
		})

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  q.QueryName,
			Reason: fmt.Sprintf("update fold on %s-replicated engine, lag=%s", profile.Replication, lag),
		})
	}

	return nil
}

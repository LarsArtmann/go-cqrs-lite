package metaengine

import "fmt"

// replicationRule surfaces replication metadata as planning diagnostics. When a
// query is routed to an engine whose data crosses process boundaries
// (IsReplicated) with a non-zero replication lag, the rule emits an INFO
// diagnostic so consumers understand the freshness tradeoff.
//
// Staleness is NOT latency — the cost estimator already adds NetworkRTT to the
// latency estimate. This rule makes the freshness dimension visible in EXPLAIN
// output instead of hiding it inside an opaque EngineProfile.
type replicationRule struct{}

func (*replicationRule) Name() string { return "replication" }

func (r *replicationRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		meta, ok := ctx.Store.queries[q.QueryName]
		if !ok {
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
			Level: DiagLevelInfo,
			Query: q.QueryName,
			Message: fmt.Sprintf(
				"routed to %s engine %q with %s replication — reads may be stale by up to %s",
				profile.Replication, profile.Name, profile.Replication, lag,
			),
		})

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  q.QueryName,
			Reason: fmt.Sprintf("%s replication, lag=%s", profile.Replication, lag),
		})
	}

	return nil
}

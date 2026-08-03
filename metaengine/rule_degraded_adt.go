package metaengine

import "fmt"

// degradedADTRule emits a DEGRADED diagnostic when a query is routed to an
// engine whose profile marks the ADT as degraded (brute-force fallback). This
// makes the "I can do this, but I'm not good at it" tradeoff visible at plan
// time instead of discovering it through poor runtime performance.
//
// The rule does NOT override engine selection — it only surfaces diagnostics.
// The cost-based ranker in planQuery already prefers native engines over
// degraded ones when both are available (native engines typically have lower
// complexity). This rule fires when ONLY degraded engines are available for
// the ADT (ADR-0094).
type degradedADTRule struct{}

func (*degradedADTRule) Name() string { return "degraded-adt" }

func (r *degradedADTRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		meta, ok := ctx.Store.queries[q.QueryName]
		if !ok {
			continue
		}

		profile := meta.QueryEngine().Profile()
		if !profile.IsDegraded(q.ADT) {
			continue
		}

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelDegraded,
			Query: q.QueryName,
			Message: fmt.Sprintf(
				"DEGRADED: %s routed to %s via %s fallback — native engine recommended for production",
				q.ADT, profile.Name, q.Complexity,
			),
		})

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  q.QueryName,
			Reason: fmt.Sprintf("%s degraded on %s", q.ADT, profile.Name),
		})
	}

	return nil
}

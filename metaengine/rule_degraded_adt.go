package metaengine

import "fmt"

// degradedADTRule emits a DEGRADED diagnostic when a query is routed to an
// engine whose profile marks the ADT as degraded (brute-force fallback). This
// makes the "I can do this, but I'm not good at it" tradeoff visible at plan
// time instead of discovering it through poor runtime performance.
//
// The diagnostic includes:
//   - the estimated latency (cost penalty) so operators can judge severity
//   - a recommendation for a native engine if one is available in the store
//
// The rule does NOT override engine selection — it only surfaces diagnostics.
// The cost-based ranker in planQuery already prefers native engines over
// degraded ones when both are available. This rule fires when ONLY degraded
// engines are available for the ADT, or when the ranker chose a degraded
// engine due to other cost factors (ADR-0094).
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

		recommendation := findNativeADTEngine(
			ctx.Store.engines, q.ADT, profile.Name,
		)

		msg := fmt.Sprintf(
			"DEGRADED: %s routed to %s via %s fallback — est %.2fms",
			q.ADT, profile.Name, q.Complexity, q.Cost.EstimatedLatencyMs,
		)

		if recommendation != "" {
			msg += fmt.Sprintf(" — native engine %q recommended", recommendation)
		} else {
			msg += " — no native engine available for this ADT"
		}

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level:   DiagLevelDegraded,
			Query:   q.QueryName,
			Message: msg,
		})

		traceReason := fmt.Sprintf("%s degraded on %s", q.ADT, profile.Name)
		if recommendation != "" {
			traceReason += fmt.Sprintf(", native %s available", recommendation)
		}

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  q.QueryName,
			Reason: traceReason,
		})
	}

	return nil
}

// findNativeADTEngine scans the given engines for one that supports the ADT
// natively (in Supports but NOT in DegradedADTs), excluding the named engine.
// Returns the first native engine name, or "" if none is found.
func findNativeADTEngine(engines []Engine, adt ADT, excludeName string) string {
	for _, eng := range engines {
		p := eng.Profile()
		if p.Name == excludeName {
			continue
		}

		if _, ok := p.Supports[adt]; ok && !p.IsDegraded(adt) {
			return p.Name
		}
	}

	return ""
}

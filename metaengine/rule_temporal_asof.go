package metaengine

import "fmt"

// temporalAsOfRule warns when a query declaring temporal intent (an AsOf
// input field, ADR-0141 §4) is assigned to an engine that does not currently
// record cell versions — the "honest degradation" row of the layered-
// architecture §3 matrix. Such queries fail at execution with
// ErrUnsupportedADT instead of silently returning latest-only; this rule
// surfaces the mismatch at PLAN time so operators can pick a versioned engine
// (memory-with-versioning, sqlite WithCellVersioning, bigtable) or drop the
// AsOf field.
//
// This is advisory (WARN) — the planner still allows the assignment. A
// consumer that only occasionally sets AsOf and handles the error can safely
// ignore the warning.
type temporalAsOfRule struct{}

func (*temporalAsOfRule) Name() string { return "temporal-asof" }

func (r *temporalAsOfRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		meta, ok := ctx.Store.queries[q.QueryName]
		if !ok || !meta.QueryDeclaresAsOf() {
			continue
		}

		eng := meta.QueryEngine()
		if EngineVersionsCells(eng) {
			continue
		}

		profile := eng.Profile()

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelWarn,
			Query: q.QueryName,
			Message: fmt.Sprintf(
				"query declares an AsOf input field but engine %q does not record cell versions — temporal reads fail with ErrUnsupportedADT; use a versioned engine or read latest",
				profile.Name,
			),
		})

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:  r.Name(),
			Query: q.QueryName,
			Reason: fmt.Sprintf(
				"as-of intent on non-versioned engine %q",
				profile.Name,
			),
		})
	}

	return nil
}

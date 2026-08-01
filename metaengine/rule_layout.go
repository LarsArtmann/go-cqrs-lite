package metaengine

import "fmt"

// layoutRule auto-applies layout planning. If the assigned engine supports
// layout planning (LayoutPlanner) and the query declares filter/sort fields
// via FilterOnField/SortOnField, it generates and applies a LayoutPlan
// automatically. This eliminates the need for manual NewPlannedSQLiteEngine
// setup (ADR-0073 consequence).
type layoutRule struct {
	dryRun bool
}

func (*layoutRule) Name() string { return "auto-layout" }

func (r *layoutRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		rt, ok := ctx.Store.queries[q.QueryName]
		if !ok {
			continue
		}

		lp, ok := rt.QueryEngine().(LayoutPlanner)
		if !ok {
			continue
		}

		filterFields, sortFields, err := extractDeclarativeFields(rt.QueryConfig())
		if err != nil {
			return fmt.Errorf("auto-layout for %q: %w", rt.QueryName(), err)
		}

		if len(filterFields) == 0 && len(sortFields) == 0 {
			continue
		}

		layoutPlan := BuildLayoutPlan(rt.QueryName(), filterFields, sortFields)

		result.LayoutPlans = append(result.LayoutPlans, layoutPlan)

		layout := LayoutRow
		if rt.QueryEngine().Profile().Layouts != nil {
			if l, ok := rt.QueryEngine().Profile().Layouts[rt.QueryADT()]; ok {
				layout = l
			}
		}

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  rt.QueryName(),
			Reason: fmt.Sprintf("columns %v", layoutPlan.ColumnNames()),
			Layout: layout,
		})

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelInfo,
			Query: rt.QueryName(),
			Message: fmt.Sprintf(
				"auto-planned table %s with columns %v",
				layoutPlan.Table, layoutPlan.ColumnNames(),
			),
		})

		if !r.dryRun {
			if err := lp.ApplyLayout(rt.QueryName(), filterFields, sortFields); err != nil {
				return fmt.Errorf("auto-layout for %q: %w", rt.QueryName(), err)
			}
		}
	}

	return nil
}

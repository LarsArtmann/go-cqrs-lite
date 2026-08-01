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

		lp, ok := rt.engine.(LayoutPlanner)
		if !ok {
			continue
		}

		filterFields, sortFields, err := extractDeclarativeFields(rt.config)
		if err != nil {
			return fmt.Errorf("auto-layout for %q: %w", rt.name, err)
		}

		if len(filterFields) == 0 && len(sortFields) == 0 {
			continue
		}

		layoutPlan := BuildLayoutPlan(rt.name, filterFields, sortFields)

		result.LayoutPlans = append(result.LayoutPlans, layoutPlan)

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelInfo,
			Query: rt.name,
			Message: fmt.Sprintf(
				"auto-planned table %s with columns %v",
				layoutPlan.Table, layoutPlan.ColumnNames(),
			),
		})

		if !r.dryRun {
			if err := lp.ApplyLayout(rt.name, filterFields, sortFields); err != nil {
				return fmt.Errorf("auto-layout for %q: %w", rt.name, err)
			}
		}
	}

	return nil
}

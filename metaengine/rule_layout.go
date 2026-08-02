package metaengine

import (
	"fmt"
)

// layoutRule auto-applies layout planning. If the assigned engine supports
// layout planning (LayoutPlanner) and the query declares filter/sort fields
// via FilterOnField/SortOnField, or requests WithColumnarLayout, it generates
// and applies a LayoutPlan automatically. This eliminates the need for manual
// NewPlannedSQLiteEngine setup (ADR-0073 consequence).
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

		_, hasPlanner := rt.QueryEngine().(LayoutPlanner)
		_, hasPlanApplier := rt.QueryEngine().(LayoutPlanApplier)
		if !hasPlanner && !hasPlanApplier {
			continue
		}

		filterFields, sortFields, err := extractDeclarativeFields(rt.QueryConfig())
		if err != nil {
			return fmt.Errorf("auto-layout for %q: %w", rt.QueryName(), err)
		}

		layoutPlan := BuildLayoutPlan(rt.QueryName(), filterFields, sortFields)
		if rt.QueryConfig().columnarLayout {
			resultType := rt.QueryResultType()
			if resultType != nil && isStructOrPointerToStruct(resultType) {
				layoutPlan = BuildColumnarLayoutPlan(rt.QueryName(), resultType)
			}
		}

		if len(layoutPlan.Columns) == 0 && len(filterFields) == 0 && len(sortFields) == 0 {
			continue
		}

		result.LayoutPlans = append(result.LayoutPlans, layoutPlan)

		layout := LayoutRow
		if rt.QueryEngine().Profile().Layouts != nil {
			if l, ok := rt.QueryEngine().Profile().Layouts[rt.QueryADT()]; ok {
				layout = l
			}
		}

		reason := fmt.Sprintf("columns %v", layoutPlan.ColumnNames())
		if rt.QueryConfig().columnarLayout {
			reason = fmt.Sprintf("columnar-native columns %v", layoutPlan.ColumnNames())
		}

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  rt.QueryName(),
			Reason: reason,
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
			if err := applyLayoutPlan(rt, layoutPlan, filterFields, sortFields); err != nil {
				return fmt.Errorf("auto-layout for %q: %w", rt.QueryName(), err)
			}
		}
	}

	return nil
}

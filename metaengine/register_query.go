package metaengine

import (
	"fmt"
	"reflect"
)

// RegisterQuery adds a query to the Store at runtime, after Plan().
// The query is classified and assigned to the best available engine,
// exactly as if it had been declared at Plan time.
//
// This enables dynamic query registration for plugins, hot-reload scenarios,
// and incremental adoption (start with one query, add more as needed).
//
//	store, _ := metaengine.Plan(engines, initialQuery)
//	// ... later, at runtime:
//	store.RegisterQuery(additionalQuery)
//
// Returns an error if the query name is already registered or if no engine
// supports the query's ADT.
func (s *Store) RegisterQuery(query any) error {
	meta, ok := asQueryMeta(query)
	if !ok {
		return fmt.Errorf("%w: %T", errNotQueryMeta, query)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	name := meta.QueryName()

	if _, exists := s.queries[name]; exists {
		return fmt.Errorf("%w: %q", errDuplicateQuery, name)
	}

	assignment, err := planQuery(meta, s.engines, planConfig{writeAmplificationBudget: DefaultWriteAmplificationBudget})
	if err != nil {
		return fmt.Errorf("metaengine.RegisterQuery: %w", err)
	}

	if err := s.applyAutoLayoutForQuery(meta); err != nil {
		return err
	}

	s.queries[meta.QueryName()] = meta
	s.byInputType[meta.QueryInputTypeName()] = meta.QueryName()

	if s.plan != nil {
		s.plan.Queries = append(s.plan.Queries, assignment)
	}

	return nil
}

// applyAutoLayoutForQuery generates and applies a LayoutPlan for declarative
// FilterOnField/SortOnField queries, and for queries using WithColumnarLayout,
// when the assigned engine supports it.
func (s *Store) applyAutoLayoutForQuery(meta queryMeta) error {
	filterFields, sortFields, err := extractDeclarativeFields(meta.QueryConfig())
	if err != nil {
		return fmt.Errorf("metaengine.RegisterQuery: %w", err)
	}

	layoutPlan := BuildLayoutPlan(meta.QueryName(), filterFields, sortFields)
	if meta.QueryConfig().columnarLayout {
		rt := meta.QueryResultType()
		if rt != nil && isStructOrPointerToStruct(rt) {
			layoutPlan = BuildColumnarLayoutPlan(meta.QueryName(), rt)
		}
	}

	if len(layoutPlan.Columns) == 0 && len(filterFields) == 0 && len(sortFields) == 0 {
		return nil
	}

	if s.plan != nil {
		s.plan.LayoutPlans = append(s.plan.LayoutPlans, layoutPlan)
	}

	if err := applyLayoutPlan(meta, layoutPlan, filterFields, sortFields); err != nil {
		return fmt.Errorf(
			"metaengine.RegisterQuery: auto-layout for %q: %w",
			meta.QueryName(), err,
		)
	}

	return nil
}

// isStructOrPointerToStruct reports whether t is a struct type or a pointer to one.
func isStructOrPointerToStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct
}

// applyLayoutPlan dispatches a LayoutPlan to the engine. It prefers
// LayoutPlanApplier so the engine receives reflection-derived column types;
// otherwise it falls back to LayoutPlanner with field names.
func applyLayoutPlan(
	meta queryMeta,
	plan LayoutPlan,
	filterFields, sortFields []string,
) error {
	if lpa, ok := meta.QueryEngine().(LayoutPlanApplier); ok {
		if err := lpa.ApplyLayoutPlan(plan); err != nil {
			return fmt.Errorf("apply layout plan %q: %w", plan.Collection, err)
		}

		return nil
	}

	lp, ok := meta.QueryEngine().(LayoutPlanner)
	if !ok {
		return nil
	}

	fields := filterFields
	if meta.QueryConfig().columnarLayout {
		fields = plan.ColumnNames()
	}

	if err := lp.ApplyLayout(meta.QueryName(), fields, sortFields); err != nil {
		return fmt.Errorf("apply layout %q: %w", meta.QueryName(), err)
	}

	return nil
}

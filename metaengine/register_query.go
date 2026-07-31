package metaengine

import (
	"fmt"
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
	meta, ok := query.(queryMeta)
	if !ok {
		return fmt.Errorf("%w: %T", errNotQueryMeta, query)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	name := meta.QueryName()

	if _, exists := s.queries[name]; exists {
		return fmt.Errorf("%w: %q", errDuplicateQuery, name)
	}

	runtime, assignment, err := planQuery(meta, s.engines)
	if err != nil {
		return fmt.Errorf("metaengine.RegisterQuery: %w", err)
	}

	// Auto-apply layout planning (same as Plan()).
	if lp, ok := runtime.engine.(LayoutPlanner); ok {
		filterFields, sortFields, err := extractDeclarativeFields(meta.QueryConfig())
		if err != nil {
			return fmt.Errorf("metaengine.RegisterQuery: %w", err)
		}

		if len(filterFields) > 0 || len(sortFields) > 0 {
			layoutPlan := BuildLayoutPlan(runtime.name, filterFields, sortFields)

			if s.plan != nil {
				s.plan.LayoutPlans = append(s.plan.LayoutPlans, layoutPlan)
			}

			if err := lp.ApplyLayout(runtime.name, filterFields, sortFields); err != nil {
				return fmt.Errorf(
					"metaengine.RegisterQuery: auto-layout for %q: %w",
					runtime.name, err,
				)
			}
		}
	}

	s.queries[runtime.name] = runtime
	s.byInputType[runtime.inputTypeName] = runtime.name

	if s.plan != nil {
		s.plan.Queries = append(s.plan.Queries, assignment)
	}

	return nil
}

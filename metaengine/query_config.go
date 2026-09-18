package metaengine

// Declarative query configuration: volume/latency/layout options, priority
// resolution, and the typed filter/sort accessors (closures with optional
// pushdown specs).

import (
	"reflect"
)

// QueryOption tunes a query declaration.
type QueryOption func(*QueryConfig)

// QueryConfig holds declarative options for a query.
type QueryConfig struct {
	Volume          int64
	LatencyBudgetMs int64
	TTL             int64 // nanoseconds; 0 = no TTL
	filterAccessors []filterAccessor
	sortAccessor    sortAccessor
	columnarLayout  bool
	layoutPriority  Priority // developer per-query layout priority (ADR-0124 Layer 4)
}

// FilterCount returns the number of declarative filters on this query.
// Used by the cost model for selectivity estimation.
func (c QueryConfig) FilterCount() int { return len(c.filterAccessors) }

// Volume sets the expected query volume (events/sec) for cost estimation.
func Volume(n int64) QueryOption {
	return func(c *QueryConfig) { c.Volume = n }
}

// WithLatencyBudget sets the target latency budget for engine selection.
func WithLatencyBudget(ms int64) QueryOption {
	return func(c *QueryConfig) { c.LatencyBudgetMs = ms }
}

// WithColumnarLayout requests a fully columnar physical layout for this query.
// When the assigned engine supports LayoutPlanner/LayoutPlanApplier, the
// planner extracts ALL exported fields of the result type R into native SQL
// columns (not only the filtered/sorted fields). This lets columnar engines
// such as DuckDB run vectorized scans, GROUP BY, and aggregations directly on
// native column values instead of decoding JSON blobs.
//
// The result type R is already known from the Query[Q, R] declaration — the
// planner reflects on it during Plan(). The layout is applied automatically
// during Plan() or RegisterQuery(). The engine must implement LayoutPlanner;
// accurate SQL types require LayoutPlanApplier (currently implemented by DuckDB).
func WithColumnarLayout() QueryOption {
	return func(c *QueryConfig) { c.columnarLayout = true }
}

// WithLayoutPriority sets a per-query layout priority override (ADR-0124
// Layer 4, clarified by ADR-0125). This is the developer-side counterpart to
// the operator's DeploymentConfig priorities: the developer pins the layout
// objective for ONE query, the operator still controls Global/per-Engine
// priorities.
//
// LAYOUT-ONLY: This option influences the physical layout (Embed vs Normalize)
// via SelectLayout. It does NOT influence engine ranking — engine selection is
// 100% the operator's call via PriorityConfig (ADR-0125).
//
// The most specific priority wins:
//
//	per-Query (operator config) > per-Query (this) > per-Engine > Global
//
// The operator's PriorityConfig.PerQuery map takes precedence over this option:
// operator wins over developer. Use WithLayoutPriority when a single query has
// a different optimization objective than the rest of the deployment.
func WithLayoutPriority(p Priority) QueryOption {
	return func(c *QueryConfig) {
		if p.Valid() {
			c.layoutPriority = p
		}
	}
}

// layoutPriority returns the developer-declared layout priority for this
// query, or PriorityBalanced when none was set.
func (c QueryConfig) layoutPriorityOr(p Priority) Priority {
	if c.layoutPriority.Valid() {
		return c.layoutPriority
	}

	return p
}

// resolvePriority is the shared priority-resolution function used by both
// planQuery (engine routing + layout scoring) and ReplanLayout (layout diffing).
// Resolution order: operator per-Query → operator per-Engine/Global → developer
// WithLayoutPriority → Balanced.
func resolvePriority(
	pc *PriorityConfig,
	engineName, queryName string,
	devFallback Priority,
) Priority {
	if pc != nil {
		if p, ok := pc.PerQuery[queryName]; ok && p.Valid() {
			return p
		}

		if pc.PerEngine != nil || pc.Global != "" {
			return pc.Resolve(engineName, queryName)
		}
	}

	return devFallback
}

// priorityForQuery returns the most specific priority for a query, combining
// the operator's PriorityConfig with the developer's WithLayoutPriority
// option. Resolution order: per-Query (operator config) → developer
// WithLayoutPriority → per-Engine → Global → Balanced.
func (s *Store) priorityForQuery(engineName, queryName string, cfg QueryConfig) Priority {
	return resolvePriority(
		s.priorityConfig,
		engineName,
		queryName,
		cfg.layoutPriorityOr(PriorityBalanced),
	)
}

// filterAccessor stores a typed closure that extracts a filterable field value
// from a result item at runtime. The returnType is used to match against query
// input fields by TYPE — the engine never knows field names.
//
// When spec is non-nil, the filter is declarative (FilterOnField) and can be
// pushed down to SQL-aware engines. When spec is nil, only the closure path
// is available (in-Go filtering).
type filterAccessor struct {
	closure    any // func(r R) T — extracts field from result
	returnType reflect.Type
	spec       *FilterSpec // declarative spec for pushdown (nil = closure-only)
}

// sortAccessor stores a typed closure that extracts the sort key from a result item.
// When spec is non-nil, the sort is declarative (SortOnField) and can be pushed
// down to SQL-aware engines.
type sortAccessor struct {
	closure any       // func(r R) time.T / comparable — extracts sort key from result
	spec    *SortSpec // declarative spec for pushdown (nil = closure-only)
}

// FilterOn declares a filter on the result type using a typed closure accessor.
// The closure extracts the comparable value from a result item. At runtime,
// the engine calls the closure on each result and compares against the matching
// field in the query input (matched by TYPE, never by name):
//
//	metaengine.FilterOn(func(r FindUserResult) string { return r.Status })
//
// The query input must have a field of the same type (string) to carry the
// filter value. Multiple FilterOn closures are AND-combined.
func FilterOn[R any, T any](accessor func(r R) T) QueryOption {
	return func(c *QueryConfig) {
		var zero T

		c.filterAccessors = append(c.filterAccessors, filterAccessor{
			closure:    accessor,
			returnType: reflect.TypeOf(zero),
		})
	}
}

// SortOn declares the sort field using a typed closure accessor.
// The closure extracts the sort key from a result item. At runtime,
// the engine calls the closure to compare items:
//
//	metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt })
func SortOn[R any, T any](accessor func(r R) T) QueryOption {
	return func(c *QueryConfig) {
		c.sortAccessor = sortAccessor{closure: accessor}
	}
}

// FilterOnField declares a filter on the result type using a declarative field
// name and operator. Unlike FilterOn (closure-based), FilterOnField produces a
// FilterSpec that can be pushed down to SQL-aware engines (SQLite json_extract).
// The filter value is extracted from the query input by matching the field's
// Go type — the same type-matching mechanism as FilterOn.
//
//	metaengine.FilterOnField[FindUserResult]("status", metaengine.FilterEq)
func FilterOnField[R any](field string, op FilterOp) QueryOption {
	return func(c *QueryConfig) {
		c.filterAccessors = append(c.filterAccessors, filterAccessor{
			spec: &FilterSpec{Column: field, Op: op},
		})
	}
}

// SortOnField declares the sort field using a declarative field name. Unlike
// SortOn (closure-based), SortOnField produces a SortSpec that can be pushed
// down to SQL-aware engines (SQLite json_extract + ORDER BY).
//
//	metaengine.SortOnField[FindUserResult]("priority", true) // DESC
func SortOnField[R any](field string, desc bool) QueryOption {
	return func(c *QueryConfig) {
		c.sortAccessor = sortAccessor{
			spec: &SortSpec{Column: field, Desc: desc},
		}
	}
}

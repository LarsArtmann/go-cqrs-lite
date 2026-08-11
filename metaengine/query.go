package metaengine

import (
	"fmt"
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

// priorityForQuery returns the most specific priority for a query, combining
// the operator's PriorityConfig with the developer's WithLayoutPriority
// option. Resolution order: per-Query (operator config) → developer
// WithLayoutPriority → per-Engine → Global → Balanced.
func (s *Store) priorityForQuery(engineName, queryName string, cfg QueryConfig) Priority {
	if s.priorityConfig != nil {
		if p, ok := s.priorityConfig.PerQuery[queryName]; ok && p.Valid() {
			return p
		}

		if s.priorityConfig.PerEngine != nil || s.priorityConfig.Global != "" {
			return s.priorityConfig.Resolve(engineName, queryName)
		}
	}

	return cfg.layoutPriorityOr(PriorityBalanced)
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

// QueryDecl is a fully analyzed query declaration.
// Each query owns its own folds, ADT, and projection — there is no shared
// ReadModel. This follows the design doc principle: "each query has its own
// independent projection.".
type QueryDecl[Q any, R any] struct {
	Name          string
	Folds         []Fold
	ADT           ADT
	ReadPattern   ReadPattern
	IsPaginated   bool
	Config        QueryConfig
	InputTypeName string

	querySample  Q
	resultSample R

	// Inference support (ADR-0116 Layer 1). When needsInference is true,
	// Folds/ADT/ReadPattern are populated at Plan() time by ensureFolds().
	eventSamples   []any
	namedSamples   []NamedSample
	needsInference bool
	overrides      []overrideFold

	// Runtime-assigned by planQuery — eliminates the queryRuntime twin.
	engine      Engine
	complexity  Complexity
	foldByEvent map[string]int
}

// Query declares a query with its folds and options as variadic arguments.
// Folds and QueryOptions are separated by type at construction time:
//
//	findUser := metaengine.Query[FindUser, FindUserResult]("find_user",
//	    metaengine.OnRecord(UserCreated{}, func(_ record.Record, e UserCreated) (UserID, FindUserResult) { ... }),
//	    metaengine.OnRecord(UserSuspended{}, func(_ record.Record, e UserSuspended, prev FindUserResult) FindUserResult { ... }),
//	    metaengine.OnRecord(UserDeleted{}, metaengine.Remove[FindUserResult]()),
//	    metaengine.Volume(1_000_000),
//	)
//
// Query panics on construction errors (bad folds, ambiguous keys), following
// the MustCompile convention for package-level declarations.
func Query[Q any, R any](name string, args ...any) QueryDecl[Q, R] {
	cfg := QueryConfig{}

	var folds []Fold

	var eventSamples []any

	var namedSamples []NamedSample

	needsInference := false

	var inferenceOverrides []overrideFold

	for _, arg := range args {
		switch a := arg.(type) {
		case overrideFold:
			inferenceOverrides = append(inferenceOverrides, a)
		case Fold:
			folds = append(folds, a)
		case QueryOption:
			a(&cfg)
		case inferenceRequest:
			eventSamples = a.samples
			needsInference = true
		case namedInferenceRequest:
			namedSamples = a.samples
			needsInference = true
		default:
			panic(fmt.Sprintf(
				"metaengine.Query(%q): unexpected argument type %T (expected Fold, QueryOption, Infer, or Override)",
				name,
				arg,
			))
		}
	}

	if needsInference && len(folds) > 0 {
		panic(fmt.Sprintf(
			"metaengine.Query(%q): Infer() cannot be combined with explicit folds (use Override instead)",
			name,
		))
	}

	if len(inferenceOverrides) > 0 && !needsInference {
		panic(fmt.Sprintf(
			"metaengine.Query(%q): Override() requires Infer() — use explicit folds instead", name,
		))
	}

	if !needsInference && len(folds) == 0 {
		panic(fmt.Sprintf("metaengine.Query(%q): at least one fold required", name))
	}

	q := QueryDecl[Q, R]{
		Name:           name,
		Config:         cfg,
		eventSamples:   eventSamples,
		namedSamples:   namedSamples,
		needsInference: needsInference,
		overrides:      inferenceOverrides,
		querySample:    *new(Q),
		resultSample:   *new(R),
	}
	q.InputTypeName = qualifiedTypeName(q.querySample)

	if needsInference {
		return q
	}

	q.Folds = folds

	var err error

	q.ADT, err = classifyADT(folds)
	if err != nil {
		panic(fmt.Sprintf("metaengine.Query(%q): %v", name, err))
	}

	if err := deriveKeys(folds); err != nil {
		panic(fmt.Sprintf("metaengine.Query(%q): %v", name, err))
	}

	q.infer()

	return q
}

func (q *QueryDecl[Q, R]) infer() {
	// Detect pagination: input struct has Limit int and/or After *Cursor.
	q.IsPaginated = detectPagination(q.querySample)

	hasInputFields := len(nonMetaFields(q.querySample)) > 0

	switch {
	// ADT-specific read patterns take priority — Counter, Graph, Multimap, Log,
	// Vector, Search have fixed access patterns regardless of input struct fields.
	case q.ADT == ADTCounter:
		q.ReadPattern = ReadAggregate
	case q.ADT == ADTGraph:
		q.ReadPattern = ReadTraversal
	case q.ADT == ADTMultimap:
		q.ReadPattern = ReadMultiLookup
	case q.ADT == ADTLog:
		q.ReadPattern = ReadLogTail
	case q.ADT == ADTVector:
		q.ReadPattern = ReadVectorSearch
	case q.ADT == ADTSearch:
		q.ReadPattern = ReadFullTextSearch
	case q.ADT == ADTSpatial:
		q.ReadPattern = ReadSpatialRange
	// Map and Set can be overridden by pagination/filters into filtered scans.
	case q.IsPaginated || len(q.Config.filterAccessors) > 0:
		q.ReadPattern = ReadFilteredScan
	case hasInputFields && q.ADT == ADTSet:
		q.ReadPattern = ReadMembership
	case hasInputFields && q.ADT == ADTMap:
		q.ReadPattern = ReadPointLookup
	default:
		q.ReadPattern = ReadScan
	}
}

// extractDeclarativeFields returns the filter and sort field names declared
// via FilterOnField/SortOnField (specs with non-nil Column). Closure-only
// filters (FilterOn/SortOn) are excluded — they cannot be pushed to SQL.
func extractDeclarativeFields(
	cfg QueryConfig,
) ([]string, []string, error) {
	var filterFields, sortFields []string

	for _, acc := range cfg.filterAccessors {
		if acc.spec != nil {
			if acc.spec.Column == "" {
				return nil, nil, fmt.Errorf(
					"%w: FilterOnField has empty column name",
					errEmptyField,
				)
			}

			filterFields = append(filterFields, acc.spec.Column)
		}
	}

	if cfg.sortAccessor.spec != nil {
		if cfg.sortAccessor.spec.Column == "" {
			return nil, nil, fmt.Errorf("%w: SortOnField has empty column name", errEmptyField)
		}

		sortFields = append(sortFields, cfg.sortAccessor.spec.Column)
	}

	return filterFields, sortFields, nil
}

// queryMeta is the planner-facing interface.
//
//nolint:interfacebloat // every method is required for planning + execution
type queryMeta interface {
	QueryName() string
	QueryADT() ADT
	QueryFolds() []Fold
	QueryReadPattern() ReadPattern
	QueryIsPaginated() bool
	QueryInputTypeName() string
	QueryConfig() QueryConfig
	QueryKeyType() reflect.Type
	QueryResultType() reflect.Type

	// Runtime-assigned by planQuery.
	QueryEngine() Engine
	QueryComplexity() Complexity
	QueryFoldByEvent() map[string]int
	assignPlan(engine Engine, complexity Complexity, foldByEvent map[string]int)

	// ensureFolds runs planner-time fold inference for queries declared with
	// Infer(). For queries with explicit folds, this is a no-op. Called by
	// Plan() before planQuery().
	ensureFolds() error
}

// asQueryMeta adapts a value to queryMeta. Query() returns a value type
// (QueryDecl), but assignPlan has a pointer receiver, so QueryDecl values
// don't directly satisfy queryMeta. This helper creates a heap-allocated
// pointer copy when the value doesn't already implement the interface,
// keeping the public API (Plan/RegisterQuery accept any) unchanged.
func asQueryMeta(query any) (queryMeta, bool) {
	meta, ok := query.(queryMeta)
	if ok {
		return meta, true
	}

	rv := reflect.ValueOf(query)
	if !rv.IsValid() {
		return nil, false
	}

	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)

	meta, ok = ptr.Interface().(queryMeta)

	return meta, ok
}

func (q QueryDecl[Q, R]) QueryName() string             { return q.Name }
func (q QueryDecl[Q, R]) QueryADT() ADT                 { return q.ADT }
func (q QueryDecl[Q, R]) QueryFolds() []Fold            { return q.Folds }
func (q QueryDecl[Q, R]) QueryReadPattern() ReadPattern { return q.ReadPattern }
func (q QueryDecl[Q, R]) QueryIsPaginated() bool        { return q.IsPaginated }
func (q QueryDecl[Q, R]) QueryInputTypeName() string    { return q.InputTypeName }
func (q QueryDecl[Q, R]) QueryConfig() QueryConfig      { return q.Config }

func (q QueryDecl[Q, R]) QueryEngine() Engine              { return q.engine }
func (q QueryDecl[Q, R]) QueryComplexity() Complexity      { return q.complexity }
func (q QueryDecl[Q, R]) QueryFoldByEvent() map[string]int { return q.foldByEvent }

func (q *QueryDecl[Q, R]) assignPlan(
	engine Engine,
	complexity Complexity,
	foldByEvent map[string]int,
) {
	q.engine = engine
	q.complexity = complexity
	q.foldByEvent = foldByEvent
}

func (q QueryDecl[Q, R]) QueryKeyType() reflect.Type {
	for _, f := range q.Folds {
		switch fold := f.(type) {
		case *insertFold:
			return fold.keyType
		case *setFold:
			return fold.keyType
		default:
			// These folds do not declare a value-typed key; only insert/set do.
		}
	}

	return nil
}

// QueryResultType returns the reflect.Type of the query's result type R.
// Used by Plan() for schema enforcement: validating that fold return types
// match the declared result type.
func (q QueryDecl[Q, R]) QueryResultType() reflect.Type {
	return reflect.TypeOf(q.resultSample)
}

func (q QueryDecl[Q, R]) String() string {
	pagination := ""
	if q.IsPaginated {
		pagination = " [paginated]"
	}

	filterCount := len(q.Config.filterAccessors)

	filters := ""
	if filterCount > 0 {
		filters = fmt.Sprintf(" filter=[%d]", filterCount)
	}

	hasSort := q.Config.sortAccessor.closure != nil

	sortStr := ""
	if hasSort {
		sortStr = " [sorted]"
	}

	return fmt.Sprintf("%s: %s/%s%s%s%s",
		q.Name, q.ADT, q.ReadPattern, filters, sortStr, pagination)
}

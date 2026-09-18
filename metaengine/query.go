package metaengine

import (
	"fmt"
	"reflect"
)


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
	DeclaresAsOf  bool
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

	// Detect temporal intent: input struct has AsOf time.Time (ADR-0141 §4).
	q.DeclaresAsOf = detectAsOfInput(q.querySample)

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
	QueryDeclaresAsOf() bool
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

	// isShadow reports whether fold writes on this query must skip watcher
	// notifications. True only for the replication shim that redirects writes
	// to a shadow (Backup/Migration) engine — primaries already notified.
	isShadow() bool
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
func (q QueryDecl[Q, R]) QueryDeclaresAsOf() bool       { return q.DeclaresAsOf }
func (q QueryDecl[Q, R]) QueryInputTypeName() string    { return q.InputTypeName }
func (q QueryDecl[Q, R]) QueryConfig() QueryConfig      { return q.Config }

func (q QueryDecl[Q, R]) QueryEngine() Engine              { return q.engine }
func (q QueryDecl[Q, R]) QueryComplexity() Complexity      { return q.complexity }
func (q QueryDecl[Q, R]) QueryFoldByEvent() map[string]int { return q.foldByEvent }
func (q QueryDecl[Q, R]) isShadow() bool                   { return false }

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


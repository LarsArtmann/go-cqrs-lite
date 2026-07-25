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
	filterAccessors []filterAccessor
	sortAccessor    sortAccessor
}

// Volume sets the expected query volume (events/sec) for cost estimation.
func Volume(n int64) QueryOption {
	return func(c *QueryConfig) { c.Volume = n }
}

// WithLatencyBudget sets the target latency budget for engine selection.
func WithLatencyBudget(ms int64) QueryOption {
	return func(c *QueryConfig) { c.LatencyBudgetMs = ms }
}

// filterAccessor stores a typed closure that extracts a filterable field value
// from a result item at runtime. The returnType is used to match against query
// input fields by TYPE — the engine never knows field names.
type filterAccessor struct {
	closure    any // func(r R) T — extracts field from result
	returnType reflect.Type
}

// sortAccessor stores a typed closure that extracts the sort key from a result item.
type sortAccessor struct {
	closure any // func(r R) time.T / comparable — extracts sort key from result
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
}

// Query declares a query with its folds and options as variadic arguments.
// Folds and QueryOptions are separated by type at construction time:
//
//	findUser := metaengine.Query[FindUser, FindUserResult]("find_user",
//	    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) { ... }),
//	    metaengine.On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult { ... }),
//	    metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),
//	    metaengine.Volume(1_000_000),
//	)
//
// Query panics on construction errors (bad folds, ambiguous keys), following
// the MustCompile convention for package-level declarations.
func Query[Q any, R any](name string, args ...any) QueryDecl[Q, R] {
	cfg := QueryConfig{}

	var folds []Fold

	for _, arg := range args {
		switch a := arg.(type) {
		case Fold:
			folds = append(folds, a)
		case QueryOption:
			a(&cfg)
		default:
			panic(fmt.Sprintf(
				"metaengine.Query(%q): unexpected argument type %T (expected Fold or QueryOption)",
				name, arg,
			))
		}
	}

	if len(folds) == 0 {
		panic(fmt.Sprintf("metaengine.Query(%q): at least one fold required", name))
	}

	adt, err := classifyADT(folds)
	if err != nil {
		panic(fmt.Sprintf("metaengine.Query(%q): %v", name, err))
	}

	if err := deriveKeys(folds); err != nil {
		panic(fmt.Sprintf("metaengine.Query(%q): %v", name, err))
	}

	q := QueryDecl[Q, R]{
		Name:         name,
		Folds:        folds,
		ADT:          adt,
		Config:       cfg,
		querySample:  *new(Q),
		resultSample: *new(R),
	}
	q.InputTypeName = qualifiedTypeName(q.querySample)
	q.infer()

	return q
}

func (q *QueryDecl[Q, R]) infer() {
	// Detect pagination: input struct has Limit int and/or After *Cursor.
	q.IsPaginated = detectPagination(q.querySample)

	hasInputFields := len(nonMetaFields(q.querySample)) > 0

	switch {
	// ADT-specific read patterns take priority — Counter, Graph, Multimap, Log
	// have fixed access patterns regardless of input struct fields.
	case q.ADT == ADTCounter:
		q.ReadPattern = ReadAggregate
	case q.ADT == ADTGraph:
		q.ReadPattern = ReadTraversal
	case q.ADT == ADTMultimap:
		q.ReadPattern = ReadMultiLookup
	case q.ADT == ADTLog:
		q.ReadPattern = ReadLogTail
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

// queryMeta is the planner-facing interface.
type queryMeta interface {
	QueryName() string
	QueryADT() ADT
	QueryFolds() []Fold
	QueryReadPattern() ReadPattern
	QueryIsPaginated() bool
	QueryInputTypeName() string
	QueryConfig() QueryConfig
	QueryKeyType() reflect.Type
}

func (q QueryDecl[Q, R]) QueryName() string             { return q.Name }
func (q QueryDecl[Q, R]) QueryADT() ADT                 { return q.ADT }
func (q QueryDecl[Q, R]) QueryFolds() []Fold            { return q.Folds }
func (q QueryDecl[Q, R]) QueryReadPattern() ReadPattern { return q.ReadPattern }
func (q QueryDecl[Q, R]) QueryIsPaginated() bool        { return q.IsPaginated }
func (q QueryDecl[Q, R]) QueryInputTypeName() string    { return q.InputTypeName }
func (q QueryDecl[Q, R]) QueryConfig() QueryConfig      { return q.Config }

func (q QueryDecl[Q, R]) QueryKeyType() reflect.Type {
	for _, f := range q.Folds {
		switch f.Kind {
		case FoldInsert, FoldSet:
			return f.keyType
		case FoldUpdate, FoldRemove, FoldCount, FoldEdge, FoldSkip, FoldMultiInsert, FoldAppend:
			// These folds do not declare a value-typed key; only insert/set do.
		}
	}

	return nil
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

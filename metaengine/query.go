package metaengine

import (
	"fmt"
	"reflect"
	"strings"
)

// QueryOption tunes a query declaration.
type QueryOption func(*QueryConfig)

// QueryConfig holds declarative options for a query.
type QueryConfig struct {
	Volume          int64
	LatencyBudgetMs int64
	Filters         []string
	SortField       string
}

// Volume sets the expected query volume (events/sec) for cost estimation.
func Volume(n int64) QueryOption {
	return func(c *QueryConfig) { c.Volume = n }
}

// WithLatencyBudget sets the target latency budget for engine selection.
func WithLatencyBudget(ms int64) QueryOption {
	return func(c *QueryConfig) { c.LatencyBudgetMs = ms }
}

// FilterOn declares a filter field on the result type for filtered scans.
// Overrides auto-detected filters from query input/result field matching.
func FilterOn(field string) QueryOption {
	return func(c *QueryConfig) { c.Filters = append(c.Filters, field) }
}

// SortOn declares the sort field for a filtered scan.
// Overrides auto-detected sort field from result type inspection.
func SortOn(field string) QueryOption {
	return func(c *QueryConfig) { c.SortField = field }
}

// QueryDecl is a fully analyzed query declaration.
// Each query owns its own folds, ADT, and projection — there is no shared
// ReadModel. This follows the design doc principle: "each query has its own
// independent projection."
type QueryDecl[Q any, R any] struct {
	Name          string
	Folds         []Fold
	ADT           ADT
	ReadPattern   ReadPattern
	Filters       []FieldPath
	SortField     string
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
	resultType := reflect.TypeOf(q.resultSample)
	info, isCollection := collectionResultInfo(resultType)
	q.IsPaginated = isCollection

	queryInputFields := reflectFields(q.querySample)
	hasInputFields := len(queryInputFields) > 0

	// Apply explicit FilterOn/SortOn overrides.
	if len(q.Config.Filters) > 0 {
		resultName := ""
		if resultType != nil {
			resultName = resultType.Name()
		}

		for _, f := range q.Config.Filters {
			q.Filters = append(q.Filters, FieldPath{
				Struct: resultName,
				Field:  f,
			})
		}
	}

	if q.Config.SortField != "" {
		q.SortField = q.Config.SortField
	}

	switch {
	case isCollection:
		q.ReadPattern = ReadFilteredScan

		if len(q.Filters) == 0 && info != nil && info.itemsElemType != nil {
			elemSample := reflect.Zero(info.itemsElemType).Interface()
			q.Filters = matchFilterFields(q.querySample, elemSample)
		}

		if q.SortField == "" && info != nil && info.itemsElemType != nil {
			elemSample := reflect.Zero(info.itemsElemType).Interface()
			q.SortField = detectSortField(elemSample)
		}

	case hasInputFields && q.ADT == ADTSet:
		q.ReadPattern = ReadMembership
	case hasInputFields && q.ADT == ADTMap:
		q.ReadPattern = ReadPointLookup
	case q.ADT == ADTCounter:
		q.ReadPattern = ReadAggregate
	case q.ADT == ADTGraph:
		q.ReadPattern = ReadTraversal
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
	QueryFilters() []FieldPath
	QuerySortField() string
	QueryIsPaginated() bool
	QueryInputTypeName() string
}

func (q QueryDecl[Q, R]) QueryName() string             { return q.Name }
func (q QueryDecl[Q, R]) QueryADT() ADT                 { return q.ADT }
func (q QueryDecl[Q, R]) QueryFolds() []Fold            { return q.Folds }
func (q QueryDecl[Q, R]) QueryReadPattern() ReadPattern { return q.ReadPattern }
func (q QueryDecl[Q, R]) QueryFilters() []FieldPath     { return q.Filters }
func (q QueryDecl[Q, R]) QuerySortField() string        { return q.SortField }
func (q QueryDecl[Q, R]) QueryIsPaginated() bool        { return q.IsPaginated }
func (q QueryDecl[Q, R]) QueryInputTypeName() string    { return q.InputTypeName }

func (q QueryDecl[Q, R]) String() string {
	pagination := ""
	if q.IsPaginated {
		pagination = " [paginated]"
	}

	filters := ""
	if len(q.Filters) > 0 {
		names := make([]string, len(q.Filters))
		for i, f := range q.Filters {
			names[i] = f.Field
		}

		filters = fmt.Sprintf(" filter=[%s]", strings.Join(names, ","))
	}

	sortStr := ""
	if q.SortField != "" {
		sortStr = " sort=" + q.SortField
	}

	return fmt.Sprintf("%s: %s/%s%s%s%s",
		q.Name, q.ADT, q.ReadPattern, filters, sortStr, pagination)
}

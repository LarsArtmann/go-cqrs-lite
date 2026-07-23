package metaengine

import (
	"fmt"
	"reflect"
)

// QueryOption tunes a query declaration.
type QueryOption func(*QueryConfig)

type QueryConfig struct {
	Volume          int64
	LatencyBudgetMs int64
}

func Volume(n int64) QueryOption {
	return func(c *QueryConfig) { c.Volume = n }
}

func WithLatencyBudget(ms int64) QueryOption {
	return func(c *QueryConfig) { c.LatencyBudgetMs = ms }
}

// QueryDecl is a fully analyzed query declaration.
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

// Query declares a query with type inference.
func Query[Q any, R any](name string, folds []Fold, opts ...QueryOption) QueryDecl[Q, R] {
	cfg := QueryConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	q := QueryDecl[Q, R]{
		Name:         name,
		Folds:        folds,
		Config:       cfg,
		querySample:  *new(Q),
		resultSample: *new(R),
	}
	q.InputTypeName = reflect.TypeOf(q.querySample).Name()
	q.infer()
	return q
}

func (q *QueryDecl[Q, R]) infer() {
	adt, err := classifyADT(q.Folds)
	if err != nil {
		panic(fmt.Sprintf("metaengine.Query[%s]: %s", q.Name, err))
	}
	q.ADT = adt

	resultType := reflect.TypeOf(q.resultSample)
	elemType, isPage := unwrapPageType(resultType)
	q.IsPaginated = isPage

	queryInputFields := reflectFields(q.querySample)
	hasInputFields := len(queryInputFields) > 0

	switch {
	case isPage:
		q.ReadPattern = ReadFilteredScan
		q.ADT = ADTSortedMap
		if elemType != nil {
			elemSample := reflect.Zero(elemType).Interface()
			q.Filters = matchFilterFields(q.querySample, elemSample)
			q.SortField = detectSortField(elemSample)
		}
	case hasInputFields && adt == ADTSet:
		q.ReadPattern = ReadMembership
	case hasInputFields && adt == ADTMap:
		q.ReadPattern = ReadPointLookup
	case adt == ADTCounter:
		q.ReadPattern = ReadAggregate
	case adt == ADTGraph:
		q.ReadPattern = ReadTraversal
	default:
		q.ReadPattern = ReadScan
	}
}

// queryMeta is the planner-facing interface.
type queryMeta interface {
	QueryName() string
	QueryADT() ADT
	QueryReadPattern() ReadPattern
	QueryFilters() []FieldPath
	QuerySortField() string
	QueryIsPaginated() bool
	QueryFolds() []Fold
	QueryInputTypeName() string
}

func (q QueryDecl[Q, R]) QueryName() string             { return q.Name }
func (q QueryDecl[Q, R]) QueryADT() ADT                 { return q.ADT }
func (q QueryDecl[Q, R]) QueryReadPattern() ReadPattern { return q.ReadPattern }
func (q QueryDecl[Q, R]) QueryFilters() []FieldPath     { return q.Filters }
func (q QueryDecl[Q, R]) QuerySortField() string        { return q.SortField }
func (q QueryDecl[Q, R]) QueryIsPaginated() bool        { return q.IsPaginated }
func (q QueryDecl[Q, R]) QueryFolds() []Fold            { return q.Folds }
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
		filters = fmt.Sprintf(" filter=[%s]", joinStrings(names, ","))
	}
	sortStr := ""
	if q.SortField != "" {
		sortStr = fmt.Sprintf(" sort=%s", q.SortField)
	}
	return fmt.Sprintf("%s: %s/%s%s%s%s",
		q.Name, q.ADT, q.ReadPattern, filters, sortStr, pagination)
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}

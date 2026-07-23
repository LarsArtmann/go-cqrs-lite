package metaengine

import (
	"fmt"
	"reflect"
	"strings"
)

// QueryOption tunes a query declaration.
type QueryOption func(*QueryConfig)

type QueryConfig struct {
	Volume          int64
	LatencyBudgetMs int64
}

// Volume sets the expected query volume (events/sec) for cost estimation.
func Volume(n int64) QueryOption {
	return func(c *QueryConfig) { c.Volume = n }
}

// WithLatencyBudget sets the target latency budget for engine selection.
func WithLatencyBudget(ms int64) QueryOption {
	return func(c *QueryConfig) { c.LatencyBudgetMs = ms }
}

// QueryDecl is a fully analyzed query declaration.
// A query references a ReadModel (which owns the folds/ADT) and declares
// how it reads from it (read pattern, filters, sort, pagination).
type QueryDecl[Q any, R any] struct {
	Name          string
	Model         ReadModel
	ReadPattern   ReadPattern
	Filters       []FieldPath
	SortField     string
	IsPaginated   bool
	Config        QueryConfig
	InputTypeName string

	querySample  Q
	resultSample R
}

// Query declares a query that reads from a ReadModel.
// The read pattern, filters, and sort field are inferred from the query's
// input type (Q) and result type (R) combined with the model's ADT.
func Query[Q any, R any](name string, model ReadModel, opts ...QueryOption) QueryDecl[Q, R] {
	cfg := QueryConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	q := QueryDecl[Q, R]{
		Name:         name,
		Model:        model,
		Config:       cfg,
		querySample:  *new(Q),
		resultSample: *new(R),
	}
	q.InputTypeName = reflect.TypeOf(q.querySample).Name()
	q.infer()

	return q
}

func (q *QueryDecl[Q, R]) infer() {
	resultType := reflect.TypeOf(q.resultSample)
	elemType, isPage := unwrapPageType(resultType)
	q.IsPaginated = isPage

	queryInputFields := reflectFields(q.querySample)
	hasInputFields := len(queryInputFields) > 0

	switch {
	case isPage:
		q.ReadPattern = ReadFilteredScan

		if elemType != nil {
			elemSample := reflect.Zero(elemType).Interface()
			q.Filters = matchFilterFields(q.querySample, elemSample)
			q.SortField = detectSortField(elemSample)
		}

	case hasInputFields && q.Model.ADT == ADTSet:
		q.ReadPattern = ReadMembership
	case hasInputFields && q.Model.ADT == ADTMap:
		q.ReadPattern = ReadPointLookup
	case q.Model.ADT == ADTCounter:
		q.ReadPattern = ReadAggregate
	case q.Model.ADT == ADTGraph:
		q.ReadPattern = ReadTraversal
	default:
		q.ReadPattern = ReadScan
	}
}

// queryMeta is the planner-facing interface.
type queryMeta interface {
	QueryName() string
	QueryModel() ReadModel
	QueryReadPattern() ReadPattern
	QueryFilters() []FieldPath
	QuerySortField() string
	QueryIsPaginated() bool
	QueryInputTypeName() string
}

func (q QueryDecl[Q, R]) QueryName() string             { return q.Name }
func (q QueryDecl[Q, R]) QueryModel() ReadModel         { return q.Model }
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

	return fmt.Sprintf("%s → %s: %s/%s%s%s%s",
		q.Name, q.Model.Name, q.Model.ADT, q.ReadPattern, filters, sortStr, pagination)
}

package metaengine

// ADT is the Abstract Data Type the planner infers from fold return types.
type ADT string

const (
	ADTMap       ADT = "map"
	ADTSet       ADT = "set"
	ADTCounter   ADT = "counter"
	ADTGraph     ADT = "graph"
	ADTLog       ADT = "log"
	ADTSortedMap ADT = "sorted_map"
)

// ReadPattern describes how a query reads its projection.
type ReadPattern string

const (
	ReadPointLookup  ReadPattern = "point_lookup"
	ReadMembership   ReadPattern = "membership"
	ReadFilteredScan ReadPattern = "filtered_scan"
	ReadAggregate    ReadPattern = "aggregate"
	ReadTraversal    ReadPattern = "traversal"
	ReadScan         ReadPattern = "scan"
)

// Delta is a counter update: key to delta.
// For typed keys, use Delta[K ~string] — typo'd keys become compile errors.
type Delta map[string]int64

// TypedDelta is a type-safe counter delta where keys are a named string type.
type TypedDelta[K ~string] map[K]int64

// Edge is a graph edge returned by OnEdge folds.
type Edge struct {
	From any
	To   any
}

// Cursor marks a position in a paginated stream for continuation.
type Cursor struct {
	Value any
}

// Skip signals that an event does not apply to this projection.
type Skip struct{}

// remove is an internal sentinel for OnRemove folds.
type remove struct{}

// Remove marks a fold as a deletion. Use OnRemove[E](event) to register.
func Remove[V any]() any { return remove{} }

// FieldPath describes a typed field on a struct, used for filter/sort inference.
type FieldPath struct {
	Struct string
	Field  string
	GoType string
}

// Complexity is a Big-O class for cost estimation.
type Complexity string

const (
	ComplexityO1      Complexity = "O(1)"
	ComplexityOLogN   Complexity = "O(logN)"
	ComplexityON      Complexity = "O(N)"
	ComplexityONLogN  Complexity = "O(NlogN)"
	ComplexityODegree Complexity = "O(degree^depth)"
)

// Page is the standard paginated result wrapper for collection queries.
// When a query's result type is Page[T], the planner knows:
//   - The query returns a collection (not a single record)
//   - Pagination mechanics (limit, cursor) are handled automatically
//   - The projection needs to support filtered/ordered scanning
//
// The developer writes ONLY domain fields in the query input.
// Limit and cursor are passed as Execute options, not as domain fields.
type Page[T any] struct {
	Items   []T
	Next    *Cursor
	HasMore bool
}

// ExecOption tunes a single Execute call (pagination, consistency, etc.).
type ExecOption func(*execConfig)

type execConfig struct {
	limit  int
	cursor *Cursor
}

// WithLimit sets the maximum number of items to return in a Page.
func WithLimit(n int) ExecOption {
	return func(c *execConfig) { c.limit = n }
}

// After sets the cursor to continue from in a paginated query.
func After(cursor *Cursor) ExecOption {
	return func(c *execConfig) { c.cursor = cursor }
}

func applyExecOpts(opts []ExecOption) execConfig {
	cfg := execConfig{limit: 100}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

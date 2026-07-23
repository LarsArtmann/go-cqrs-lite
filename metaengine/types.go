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
type Delta map[string]int64

// Edge is a graph edge returned by On folds with Edge return type.
type Edge struct {
	From any
	To   any
}

// Skip is a sentinel return type signaling that an event does not apply
// to this projection (no-op). Return it from an On fold handler:
//
//	metaengine.On(SomeEvent{}, func(e SomeEvent) metaengine.Skip { return metaengine.Skip{} })
type Skip struct{}

// Cursor marks a position in a paginated stream for continuation.
type Cursor struct {
	Value any
}

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

// ExecOption tunes a single Execute call (pagination, consistency, etc.).
type ExecOption func(*execConfig)

type execConfig struct {
	limit  int
	cursor *Cursor
}

// WithLimit sets the maximum number of items to return in a paginated result.
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

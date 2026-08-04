package metaengine

// ADT is the Abstract Data Type the planner infers from fold return types.
type ADT string

const (
	ADTMap       ADT = "map"
	ADTSet       ADT = "set"
	ADTCounter   ADT = "counter"
	ADTGraph     ADT = "graph"
	ADTLog       ADT = "log"
	ADTStreamLog ADT = "stream_log"
	ADTSortedMap ADT = "sorted_map"
	ADTMultimap  ADT = "multimap"
)

// ReadPattern describes how a query reads its projection.
type ReadPattern string

const (
	ReadPointLookup    ReadPattern = "point_lookup"
	ReadMembership     ReadPattern = "membership"
	ReadFilteredScan   ReadPattern = "filtered_scan"
	ReadAggregate      ReadPattern = "aggregate"
	ReadTraversal      ReadPattern = "traversal"
	ReadScan           ReadPattern = "scan"
	ReadMultiLookup    ReadPattern = "multi_lookup"
	ReadLogTail        ReadPattern = "log_tail"
	ReadVectorSearch   ReadPattern = "vector_search"
	ReadFullTextSearch ReadPattern = "full_text_search"
	ReadSpatialRange   ReadPattern = "spatial_range"
)

// Delta is a counter update: key to delta.
type Delta map[string]int64

// Edge is a graph edge returned by On folds with Edge return type.
type Edge struct {
	From any
	To   any
}

// MultiEntry is a sentinel return type for multimap folds: one key maps to many values.
// Return it from On to classify the fold as a multimap insert:
//
//	metaengine.On(TaskAssigned{}, func(e TaskAssigned) metaengine.MultiEntry {
//	    return metaengine.MultiEntry{Key: e.Assignee, Value: e.TaskID}
//	})
type MultiEntry struct {
	Key   any
	Value any
}

// Append is a sentinel return type for log folds: append a value to an ordered log.
// Return it from On to classify the fold as a log append:
//
//	metaengine.On(TaskCreated{}, func(e TaskCreated) metaengine.Append {
//	    return metaengine.Append{Value: e}
//	})
type Append struct {
	Value any
}

// Skip is a sentinel return type signaling that an event does not apply
// to this projection (no-op). Return it from an On fold handler:
//
//	metaengine.On(SomeEvent{}, func(e SomeEvent) metaengine.Skip { return metaengine.Skip{} })
type Skip struct{}

// Cursor marks a position in a paginated stream for continuation.
// In a query input struct, a field of type *Cursor named "After" signals
// keyset pagination continuation.
type Cursor struct {
	Value any
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
